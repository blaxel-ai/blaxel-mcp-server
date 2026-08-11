package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	blaxel "github.com/blaxel-ai/sdk-go"
	"github.com/blaxel-ai/sdk-go/option"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestBuildSandboxCommandBodyIncludesFalseBooleans(t *testing.T) {
	body := buildSandboxCommandBody(runSandboxCommandArguments{
		Command:           "echo hello",
		WaitForCompletion: false,
		RestartOnFailure:  false,
		KeepAlive:         false,
	})

	for _, key := range []string{"waitForCompletion", "restartOnFailure", "keepAlive"} {
		value, ok := body[key]
		if !ok {
			t.Fatalf("expected %q to be included even when false; body=%v", key, body)
		}
		boolValue, ok := value.(bool)
		if !ok {
			t.Fatalf("expected %q to be a bool, got %T", key, value)
		}
		if boolValue {
			t.Fatalf("expected %q to be false, got true", key)
		}
	}
}

func TestBuildSandboxCommandBodyIncludesOptionalFields(t *testing.T) {
	body := buildSandboxCommandBody(runSandboxCommandArguments{
		Command:           "npm start",
		ProcessName:       "dev",
		WorkingDir:        "/app",
		Env:               map[string]string{"NODE_ENV": "test"},
		WaitForCompletion: true,
		Timeout:           30,
		WaitForPorts:      []int{3000},
		RestartOnFailure:  true,
		MaxRestarts:       2,
		KeepAlive:         true,
	})

	expected := map[string]interface{}{
		"command":           "npm start",
		"name":              "dev",
		"workingDir":        "/app",
		"env":               map[string]string{"NODE_ENV": "test"},
		"waitForCompletion": true,
		"timeout":           30,
		"waitForPorts":      []int{3000},
		"restartOnFailure":  true,
		"maxRestarts":       2,
		"keepAlive":         true,
	}

	for key, expectedValue := range expected {
		value, ok := body[key]
		if !ok {
			t.Fatalf("expected %q to be included; body=%v", key, body)
		}
		switch want := expectedValue.(type) {
		case map[string]string:
			got, ok := value.(map[string]string)
			if !ok || got["NODE_ENV"] != want["NODE_ENV"] {
				t.Fatalf("expected %q=%v, got %T %v", key, want, value, value)
			}
		case []int:
			got, ok := value.([]int)
			if !ok || len(got) != len(want) || got[0] != want[0] {
				t.Fatalf("expected %q=%v, got %T %v", key, want, value, value)
			}
		default:
			if value != expectedValue {
				t.Fatalf("expected %q=%v, got %v", key, expectedValue, value)
			}
		}
	}
}

func TestRunAgentToolMapsMessageShorthandToExactBody(t *testing.T) {
	mcpServer, handler := newRuntimeToolTestServer(t)

	result := callRuntimeTool(t, mcpServer, "run_agent", map[string]any{
		"name":    "fixture-agent",
		"message": "hello from fixture",
		"context": `{"trace":"local","nested":{"enabled":true}}`,
	})
	if result.IsError {
		t.Fatalf("run_agent returned tool error: %s", toolResultText(result))
	}
	if len(handler.agentCalls) != 1 {
		t.Fatalf("expected one agent call, got %d", len(handler.agentCalls))
	}

	call := handler.agentCalls[0]
	if call.name != "fixture-agent" {
		t.Fatalf("expected agent name fixture-agent, got %q", call.name)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(call.body), &body); err != nil {
		t.Fatalf("expected JSON body, got %q: %v", call.body, err)
	}
	if len(body) != 2 || body["input"] != "hello from fixture" {
		t.Fatalf("expected exact input/context body, got %#v", body)
	}
	contextData, ok := body["context"].(map[string]any)
	if !ok || contextData["trace"] != "local" {
		t.Fatalf("expected decoded context object, got %T %#v", body["context"], body["context"])
	}
	if call.path != "" {
		t.Fatalf("expected default agent path, got %q", call.path)
	}
}

func TestRunAgentToolForwardsArbitraryBodyAndPath(t *testing.T) {
	mcpServer, handler := newRuntimeToolTestServer(t)
	input := map[string]any{
		"marker": "fixture",
		"nested": map[string]any{"number": 42.0, "enabled": true},
	}

	result := callRuntimeTool(t, mcpServer, "run_agent", map[string]any{
		"name": "fixture-agent",
		"body": input,
		"path": "/e2e/echo",
	})
	if result.IsError {
		t.Fatalf("run_agent returned tool error: %s", toolResultText(result))
	}
	call := handler.agentCalls[0]
	var body map[string]any
	if err := json.Unmarshal([]byte(call.body), &body); err != nil {
		t.Fatalf("expected JSON body, got %q: %v", call.body, err)
	}
	if fmt.Sprint(body) != fmt.Sprint(input) {
		t.Fatalf("expected arbitrary body %#v, got %#v", input, body)
	}
	if call.path != "/e2e/echo" {
		t.Fatalf("expected path /e2e/echo, got %q", call.path)
	}
}

func TestRunAgentToolRequiresExactlyOneInputMode(t *testing.T) {
	for _, test := range []struct {
		name string
		args map[string]any
	}{
		{name: "neither", args: map[string]any{"name": "fixture-agent"}},
		{name: "both", args: map[string]any{"name": "fixture-agent", "message": "hello", "body": map[string]any{"raw": true}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			mcpServer, handler := newRuntimeToolTestServer(t)
			result := callRuntimeTool(t, mcpServer, "run_agent", test.args)
			if !result.IsError || !strings.Contains(toolResultText(result), "exactly one of message or body") {
				t.Fatalf("expected input-mode error, got error=%v text=%q", result.IsError, toolResultText(result))
			}
			if len(handler.agentCalls) != 0 {
				t.Fatalf("expected no agent call, got %d", len(handler.agentCalls))
			}
		})
	}
}

func TestRunAgentToolRejectsInvalidContext(t *testing.T) {
	for _, agentContext := range []string{`not-json`, `["not","an","object"]`} {
		mcpServer, handler := newRuntimeToolTestServer(t)
		result := callRuntimeTool(t, mcpServer, "run_agent", map[string]any{
			"name": "fixture-agent", "message": "hello", "context": agentContext,
		})
		if !result.IsError || !strings.Contains(toolResultText(result), "context must be a JSON object") {
			t.Fatalf("expected context error for %q, got error=%v text=%q", agentContext, result.IsError, toolResultText(result))
		}
		if len(handler.agentCalls) != 0 {
			t.Fatalf("expected no agent call, got %d", len(handler.agentCalls))
		}
	}
}

func TestRunAgentSchemaSupportsBothInputModes(t *testing.T) {
	mcpServer, _ := newRuntimeToolTestServer(t)
	tool := mcpServer.GetTool("run_agent")
	if tool == nil {
		t.Fatal("expected run_agent tool")
	}
	for _, property := range []string{"name", "message", "context", "body", "path", "workspace"} {
		if _, ok := tool.Tool.InputSchema.Properties[property]; !ok {
			t.Errorf("expected run_agent schema property %q", property)
		}
	}
	if len(tool.Tool.InputSchema.Required) != 1 || tool.Tool.InputSchema.Required[0] != "name" {
		t.Fatalf("expected only name to be schema-required, got %v", tool.Tool.InputSchema.Required)
	}
}

func TestRunModelToolAcceptsJSONStringBody(t *testing.T) {
	mcpServer, handler := newRuntimeToolTestServer(t)

	body := `{"messages":[{"role":"user","content":"ping"}]}`
	result := callRuntimeTool(t, mcpServer, "run_model", map[string]any{
		"name": "fixture-model",
		"path": "/v1/chat/completions",
		"body": body,
	})
	if result.IsError {
		t.Fatalf("run_model returned tool error: %s", toolResultText(result))
	}
	if len(handler.modelCalls) != 1 {
		t.Fatalf("expected one model call, got %d", len(handler.modelCalls))
	}

	call := handler.modelCalls[0]
	if call.body != body {
		t.Fatalf("expected JSON string body to pass through unchanged, got %q", call.body)
	}
	if call.path != "/v1/chat/completions" {
		t.Fatalf("expected path /v1/chat/completions, got %q", call.path)
	}
	if call.method != http.MethodPost {
		t.Fatalf("expected POST method, got %q", call.method)
	}
}

func TestRunModelToolAcceptsObjectBody(t *testing.T) {
	mcpServer, handler := newRuntimeToolTestServer(t)

	result := callRuntimeTool(t, mcpServer, "run_model", map[string]any{
		"name": "fixture-model",
		"body": map[string]any{
			"messages": []any{
				map[string]any{"role": "user", "content": "ping"},
			},
			"temperature": 0.2,
		},
	})
	if result.IsError {
		t.Fatalf("run_model returned tool error: %s", toolResultText(result))
	}
	if len(handler.modelCalls) != 1 {
		t.Fatalf("expected one model call, got %d", len(handler.modelCalls))
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(handler.modelCalls[0].body), &body); err != nil {
		t.Fatalf("expected object body to be marshaled as JSON, got %q: %v", handler.modelCalls[0].body, err)
	}
	if got := body["temperature"]; got != 0.2 {
		t.Fatalf("expected temperature 0.2, got %v", got)
	}
	if handler.modelCalls[0].path != "/v1/chat/completions" {
		t.Fatalf("expected default model path /v1/chat/completions, got %q", handler.modelCalls[0].path)
	}
}

func TestRunModelBodySchemaAllowsObjectOrString(t *testing.T) {
	mcpServer, _ := newRuntimeToolTestServer(t)
	tool := mcpServer.GetTool("run_model")
	if tool == nil {
		t.Fatal("expected run_model tool")
		return
	}

	bodySchema, ok := tool.Tool.InputSchema.Properties["body"].(map[string]any)
	if !ok {
		t.Fatalf("expected body schema map, got %T", tool.Tool.InputSchema.Properties["body"])
	}
	assertObjectOrStringType(t, "run_model", bodySchema)
	description, _ := bodySchema["description"].(string)
	if !strings.Contains(description, "object or JSON string") {
		t.Fatalf("expected body description to document object-or-string input, got %q", description)
	}
}

// assertObjectOrStringType requires an explicit JSON Schema type that permits
// both objects and strings. Directory validators reject typeless parameters,
// and a single scalar type would break one of the two supported body forms.
func assertObjectOrStringType(t *testing.T, toolName string, schema map[string]any) {
	t.Helper()
	types, ok := schema["type"].([]string)
	if !ok {
		t.Fatalf("%s body schema must declare an explicit JSON Schema type list, got %T (%v)", toolName, schema["type"], schema["type"])
	}
	seen := map[string]bool{}
	for _, typ := range types {
		seen[typ] = true
	}
	if !seen["object"] || !seen["string"] || len(types) != 2 {
		t.Fatalf("%s body schema type must allow exactly object and string, got %v", toolName, types)
	}
}

func TestRunAgentBodySchemaAllowsObjectOrString(t *testing.T) {
	mcpServer, _ := newRuntimeToolTestServer(t)
	tool := mcpServer.GetTool("run_agent")
	if tool == nil {
		t.Fatal("expected run_agent tool")
		return
	}
	bodySchema, ok := tool.Tool.InputSchema.Properties["body"].(map[string]any)
	if !ok {
		t.Fatalf("expected body schema map, got %T", tool.Tool.InputSchema.Properties["body"])
	}
	assertObjectOrStringType(t, "run_agent", bodySchema)
}

func TestSDKHandlerRunAgentSendsBodyAndPathUnchanged(t *testing.T) {
	handler, requests := newSDKHandlerCapture(t, "agent", "fixture-agent", "/runtime/agent")
	body := `{"arbitrary":{"nested":true},"count":2}`

	result, err := handler.RunAgent(context.Background(), "fixture-agent", body, "/e2e/echo")
	if err != nil {
		t.Fatalf("RunAgent returned error: %v", err)
	}
	if !strings.Contains(result, `"ok": true`) {
		t.Fatalf("expected formatted JSON response, got %q", result)
	}

	request := requests.last(t)
	if request.method != http.MethodPost {
		t.Fatalf("expected POST runtime request, got %s", request.method)
	}
	if request.path != "/runtime/agent/e2e/echo" {
		t.Fatalf("expected runtime agent path, got %q", request.path)
	}

	var got, want any
	if err := json.Unmarshal([]byte(request.body), &got); err != nil {
		t.Fatalf("expected JSON request body, got %q: %v", request.body, err)
	}
	if err := json.Unmarshal([]byte(body), &want); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("expected body %#v, got %#v", want, got)
	}
}

func TestSDKHandlerRunJobTargetsExecutionsPath(t *testing.T) {
	handler, requests := newSDKHandlerCapture(t, "job", "fixture-job", "/runtime/job")

	result, err := handler.RunJob(context.Background(), "fixture-job", `{"input":"ping"}`)
	if err != nil {
		t.Fatalf("RunJob returned error: %v", err)
	}
	if !strings.Contains(result, `"ok": true`) {
		t.Fatalf("expected formatted JSON response, got %q", result)
	}

	request := requests.last(t)
	if request.method != http.MethodPost {
		t.Fatalf("expected POST runtime request, got %s", request.method)
	}
	if request.path != "/runtime/job/executions" {
		t.Fatalf("expected job execution path, got %q", request.path)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(request.body), &decoded); err != nil {
		t.Fatalf("expected JSON body, got %q: %v", request.body, err)
	}
	if decoded["input"] != "ping" {
		t.Fatalf("expected input ping, got %v", decoded["input"])
	}
}

func TestSDKHandlerRunModelSendsDecodedJSONBody(t *testing.T) {
	handler, requests := newSDKHandlerCapture(t, "model", "fixture-model", "/runtime/model")

	body := `{"messages":[{"role":"user","content":"ping"}],"temperature":0.2}`
	result, err := handler.RunModel(context.Background(), "fixture-model", body, "/v1/chat/completions", http.MethodPost)
	if err != nil {
		t.Fatalf("RunModel returned error: %v", err)
	}
	if !strings.Contains(result, `"ok": true`) {
		t.Fatalf("expected formatted JSON response, got %q", result)
	}

	request := requests.last(t)
	if request.method != http.MethodPost {
		t.Fatalf("expected POST runtime request, got %s", request.method)
	}
	if request.path != "/runtime/model/v1/chat/completions" {
		t.Fatalf("expected model runtime path, got %q", request.path)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(request.body), &decoded); err != nil {
		t.Fatalf("expected JSON body, got %q: %v", request.body, err)
	}
	if decoded["temperature"] != 0.2 {
		t.Fatalf("expected temperature 0.2, got %v", decoded["temperature"])
	}
	if _, ok := decoded["messages"].([]any); !ok {
		t.Fatalf("expected messages array, got %T %v", decoded["messages"], decoded["messages"])
	}
}

type runtimeToolCall struct {
	name   string
	body   string
	path   string
	method string
}

type recordingRuntimeHandler struct {
	agentCalls []runtimeToolCall
	modelCalls []runtimeToolCall
}

func (h *recordingRuntimeHandler) RunAgent(ctx context.Context, name, body, path string) (string, error) {
	h.agentCalls = append(h.agentCalls, runtimeToolCall{name: name, body: body, path: path})
	return `{"ok":true}`, nil
}

func (h *recordingRuntimeHandler) RunJob(ctx context.Context, name, parameters string) (string, error) {
	return `{"ok":true}`, nil
}

func (h *recordingRuntimeHandler) RunModel(ctx context.Context, name, body, path, method string) (string, error) {
	h.modelCalls = append(h.modelCalls, runtimeToolCall{name: name, body: body, path: path, method: method})
	return `{"ok":true}`, nil
}

func (h *recordingRuntimeHandler) RunSandbox(ctx context.Context, name, body, method, path string) (string, error) {
	return `{"ok":true}`, nil
}

func newRuntimeToolTestServer(t *testing.T) (*server.MCPServer, *recordingRuntimeHandler) {
	t.Helper()
	mcpServer := server.NewMCPServer("runtime-test", "0.0.0")
	handler := &recordingRuntimeHandler{}
	RegisterRuntimeTools(mcpServer, handler, &config.Config{Workspace: "fixture-workspace"})
	return mcpServer, handler
}

func callRuntimeTool(t *testing.T, mcpServer *server.MCPServer, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	tool := mcpServer.GetTool(name)
	if tool == nil {
		t.Fatalf("expected tool %q to be registered", name)
		return nil
	}

	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	})
	if err != nil {
		t.Fatalf("tool %q returned handler error: %v", name, err)
	}
	return result
}

func toolResultText(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	if text, ok := mcp.AsTextContent(result.Content[0]); ok {
		return text.Text
	}
	return fmt.Sprintf("%v", result.Content[0])
}

type capturedRuntimeRequest struct {
	method string
	path   string
	body   string
}

type capturedRuntimeRequests struct {
	mu       sync.Mutex
	requests []capturedRuntimeRequest
}

func (c *capturedRuntimeRequests) append(request capturedRuntimeRequest) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests = append(c.requests, request)
}

func (c *capturedRuntimeRequests) last(t *testing.T) capturedRuntimeRequest {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.requests) == 0 {
		t.Fatal("expected a captured runtime request")
	}
	return c.requests[len(c.requests)-1]
}

func newSDKHandlerCapture(t *testing.T, resourceType, resourceName, runtimePath string) (*SDKHandler, *capturedRuntimeRequests) {
	t.Helper()

	requests := &capturedRuntimeRequests{}
	var testServer *httptest.Server
	testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/"+resourceType+"s/"+resourceName:
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"metadata":{"name":%q,"url":%q}}`, resourceName, testServer.URL+runtimePath)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, runtimePath):
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			requests.append(capturedRuntimeRequest{
				method: r.Method,
				path:   r.URL.Path,
				body:   string(body),
			})
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.Error(w, fmt.Sprintf("unexpected request %s %s", r.Method, r.URL.Path), http.StatusNotFound)
		}
	}))
	t.Cleanup(testServer.Close)

	client := blaxel.NewClient(
		option.WithBaseURL(testServer.URL),
		option.WithAPIKey("fixture-api-key"),
	)
	return &SDKHandler{
		blaxelClient: &client,
		cfg:          &config.Config{Workspace: "fixture-workspace"},
	}, requests
}
