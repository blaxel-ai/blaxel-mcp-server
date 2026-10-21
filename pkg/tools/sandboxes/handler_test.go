package sandboxes

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/toolkit/sdk"
	"github.com/mark3labs/mcp-go/server"
)

func TestSDKHandlerRejectsInvalidMemoryBeforeAPIRequest(t *testing.T) {
	// Explicit zero is rejected by the MCP binder; zero at this compatibility
	// interface means the optional field was omitted.
	for _, memory := range []float64{-100, 1023, 262145, 1024.5, 999999999999, math.NaN(), math.Inf(1), math.Inf(-1)} {
		handler, requestCount := newCreateSandboxHandlerTestServer(t)

		_, err := handler.CreateSandbox(context.Background(), "invalid-memory", "", memory, "", "")
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "memory") {
			t.Fatalf("expected actionable memory error for %v, got %v", memory, err)
		}
		if got := requestCount.Load(); got != 0 {
			t.Fatalf("expected no API request, got %d", got)
		}
	}
}

func TestSDKHandlerRejectsInvalidPortsBeforeAPIRequest(t *testing.T) {
	for _, ports := range []string{"not-a-port", "0", "65536"} {
		handler, requestCount := newCreateSandboxHandlerTestServer(t)

		_, err := handler.CreateSandbox(context.Background(), "invalid-port", "", 0, ports, "")
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "port") {
			t.Fatalf("expected actionable port error for %q, got %v", ports, err)
		}
		if got := requestCount.Load(); got != 0 {
			t.Fatalf("expected no API request, got %d", got)
		}
	}
}

func TestSDKHandlerAllowsOmittedMemory(t *testing.T) {
	handler, requestCount := newCreateSandboxHandlerTestServer(t)

	if _, err := handler.CreateSandbox(context.Background(), "default-memory", "", 0, "", ""); err != nil {
		t.Fatalf("expected omitted memory to use the API default: %v", err)
	}
	if got := requestCount.Load(); got != 1 {
		t.Fatalf("expected one API request, got %d", got)
	}
}

func TestSDKHandlerAllowsMemoryBoundariesAndValidPorts(t *testing.T) {
	for _, memory := range []float64{1024, 262144} {
		handler, requestCount := newCreateSandboxHandlerTestServer(t)
		if _, err := handler.CreateSandbox(context.Background(), "valid", "", memory, "8080,65535", ""); err != nil {
			t.Fatalf("expected memory %v to reach API: %v", memory, err)
		}
		if got := requestCount.Load(); got != 1 {
			t.Fatalf("memory %v made %d API requests, want 1", memory, got)
		}
	}
}

func newCreateSandboxHandlerTestServer(t *testing.T) (SandboxHandler, *atomic.Int32) {
	t.Helper()
	var requestCount atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"metadata":{"name":"valid"},"status":"DEPLOYING"}`))
	}))
	t.Cleanup(api.Close)

	handler, err := NewSDKHandler(&config.Config{
		APIEndpoint: api.URL,
		RunEndpoint: api.URL,
		Workspace:   "test-workspace",
		Credentials: sdk.Credentials{APIKey: "test-api-key"},
	})
	if err != nil {
		t.Fatalf("create SDK handler: %v", err)
	}
	return handler, &requestCount
}

func TestCreateSandboxRegionReachesAPI(t *testing.T) {
	var got struct {
		Spec struct {
			Region string `json:"region"`
		} `json:"spec"`
	}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"metadata":{"name":"regional"},"status":"DEPLOYING"}`))
	}))
	defer api.Close()
	cfg := &config.Config{APIEndpoint: api.URL, RunEndpoint: api.URL, Workspace: "test", Credentials: sdk.Credentials{APIKey: "test-key"}}
	handler, err := NewSDKHandler(cfg)
	if err != nil {
		t.Fatal(err)
	}
	srv := server.NewMCPServer("test", "1")
	RegisterSandboxTools(srv, handler, cfg)
	result := callCreateSandboxTool(t, srv, map[string]any{"name": "regional", "region": "us-was-1"})
	if result.IsError {
		t.Fatal(toolResultText(result))
	}
	if got.Spec.Region != "us-was-1" {
		t.Fatalf("region lost: %q", got.Spec.Region)
	}
}

type capturedCreateRequest struct {
	query url.Values
	body  map[string]any
}

// newLifecycleAPIServer serves POST /sandboxes with the given status and body
// and records each request.
func newLifecycleAPIServer(t *testing.T, status int, response string) (*server.MCPServer, *[]capturedCreateRequest) {
	t.Helper()
	var requests []capturedCreateRequest
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		requests = append(requests, capturedCreateRequest{query: r.URL.Query(), body: body})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(api.Close)

	cfg := &config.Config{APIEndpoint: api.URL, RunEndpoint: api.URL, Workspace: "test", Credentials: sdk.Credentials{APIKey: "test-key"}}
	handler, err := NewSDKHandler(cfg)
	if err != nil {
		t.Fatal(err)
	}
	srv := server.NewMCPServer("test", "1")
	RegisterSandboxTools(srv, handler, cfg)
	return srv, &requests
}

func jsonPath(t *testing.T, value any, path ...any) any {
	t.Helper()
	for _, key := range path {
		switch typed := key.(type) {
		case string:
			object, ok := value.(map[string]any)
			if !ok {
				t.Fatalf("at %q: %T is not an object", typed, value)
			}
			value = object[typed]
		case int:
			list, ok := value.([]any)
			if !ok || typed >= len(list) {
				t.Fatalf("at [%d]: %v is not a long enough list", typed, value)
			}
			value = list[typed]
		}
	}
	return value
}

func TestCreateSandboxRequestBodyCarriesLifecycleSettings(t *testing.T) {
	srv, requests := newLifecycleAPIServer(t, http.StatusOK, `{"metadata":{"name":"lifecycle"},"status":"DEPLOYING"}`)

	result := callCreateSandboxTool(t, srv, lifecycleArguments())
	if result.IsError {
		t.Fatal(toolResultText(result))
	}
	if len(*requests) != 1 {
		t.Fatalf("got %d API requests, want 1", len(*requests))
	}
	request := (*requests)[0]
	if got := request.query.Get("createIfNotExist"); got != "true" {
		t.Errorf("createIfNotExist query = %q, want true", got)
	}

	body := request.body
	for _, check := range []struct {
		path []any
		want any
	}{
		{[]any{"metadata", "name"}, "lifecycle"},
		{[]any{"metadata", "labels", "task"}, "build"},
		{[]any{"spec", "region"}, "us-pdx-1"},
		{[]any{"spec", "runtime", "image"}, "blaxel/base-image:latest"},
		{[]any{"spec", "runtime", "memory"}, float64(2048)},
		{[]any{"spec", "runtime", "ports", 0, "target"}, float64(3000)},
		{[]any{"spec", "runtime", "envs", 0, "name"}, "FOO"},
		{[]any{"spec", "runtime", "ttl"}, "24h"},
		{[]any{"spec", "runtime", "standbyGracePeriodSeconds"}, float64(10)},
		{[]any{"spec", "runtime", "terminationGracePeriodSeconds"}, float64(30)},
		{[]any{"spec", "lifecycle", "terminatedRetention"}, "1h"},
		{[]any{"spec", "lifecycle", "expirationPolicies", 0}, map[string]any{"type": "ttl-idle", "value": "1h", "action": "delete"}},
		{[]any{"spec", "lifecycle", "expirationPolicies", 1}, map[string]any{"type": "date", "value": "2030-01-01T00:00:00Z", "action": "delete"}},
		{[]any{"spec", "volumes", 0}, map[string]any{"name": "data", "mountPath": "/data", "readOnly": true}},
		{[]any{"spec", "volumes", 1}, map[string]any{"name": "scratch", "mountPath": "/scratch", "type": "ephemeral", "sizeMb": float64(2048)}},
	} {
		if got := jsonPath(t, body, check.path...); !reflect.DeepEqual(got, check.want) {
			t.Errorf("%v = %#v, want %#v", check.path, got, check.want)
		}
	}

	structured, _ := result.StructuredContent.(map[string]any)
	steps, _ := structured["nextSteps"].([]any)
	if len(steps) == 0 || !strings.Contains(fmt.Sprint(steps), "deleted automatically") {
		t.Fatalf("result next steps = %v, want the expiry rules", steps)
	}
}

func TestCreateSandboxRequestBodyOmitsUnsetLifecycleSettings(t *testing.T) {
	srv, requests := newLifecycleAPIServer(t, http.StatusOK, `{"metadata":{"name":"plain"},"status":"DEPLOYING"}`)

	result := callCreateSandboxTool(t, srv, map[string]any{"name": "plain"})
	if result.IsError {
		t.Fatal(toolResultText(result))
	}
	request := (*requests)[0]
	if request.query.Has("createIfNotExist") {
		t.Errorf("createIfNotExist sent although not requested: %v", request.query)
	}
	spec := jsonPath(t, request.body, "spec").(map[string]any)
	for _, key := range []string{"lifecycle", "volumes", "region"} {
		if _, ok := spec[key]; ok {
			t.Errorf("spec.%s sent although not requested: %v", key, spec)
		}
	}
	runtime := jsonPath(t, request.body, "spec", "runtime").(map[string]any)
	if len(runtime) != 0 {
		t.Errorf("spec.runtime = %v, want empty so API defaults apply", runtime)
	}
	if _, ok := jsonPath(t, request.body, "metadata").(map[string]any)["labels"]; ok {
		t.Error("metadata.labels sent although not requested")
	}

	structured, _ := result.StructuredContent.(map[string]any)
	if !strings.Contains(fmt.Sprint(structured["nextSteps"]), "until you call delete_sandbox") {
		t.Fatalf("next steps = %v, want the no-expiry warning", structured["nextSteps"])
	}
}

func TestCreateSandboxRelaysAPIValidationErrorWithNextStep(t *testing.T) {
	const message = "Invalid sandbox configuration: ttl 1m is too short: the minimum is 5m"
	srv, requests := newLifecycleAPIServer(t, http.StatusBadRequest,
		`{"code":"INVALID_SANDBOX_CONFIGURATION","message":"`+message+`","status_code":400}`)

	result := callCreateSandboxTool(t, srv, map[string]any{"name": "short", "ttl": "1m"})
	text := toolResultText(result)
	if !result.IsError || !strings.Contains(text, "status 400: "+message) {
		t.Fatalf("expected the API message, got error=%v text=%q", result.IsError, text)
	}
	if !strings.Contains(text, "Next step: Set ttl and ttl-max-age to at least 5m") {
		t.Fatalf("expected a next step, got %q", text)
	}
	if got := jsonPath(t, (*requests)[0].body, "spec", "runtime", "ttl"); got != "1m" {
		t.Fatalf("ttl sent = %v, want the caller's value passed through", got)
	}
}

func TestCreateSandboxDoesNotRelayServerErrorBodies(t *testing.T) {
	srv, _ := newLifecycleAPIServer(t, http.StatusInternalServerError, `{"message":"panic at internal-host:1234"}`)

	result := callCreateSandboxTool(t, srv, map[string]any{"name": "broken"})
	if text := toolResultText(result); !result.IsError || strings.Contains(text, "internal-host") {
		t.Fatalf("server error body leaked or call succeeded: error=%v text=%q", result.IsError, text)
	}
}

// A hint that suggests delete_sandbox must limit it to a sandbox this session
// created, and send anything else to the user.
func TestCreateSandboxErrorHintsOnlyDeleteWhatThisSessionCreated(t *testing.T) {
	for _, message := range []string{
		"Sandbox shared already exists",
		"Volume data is already attached to another sandbox",
		"Sandbox quota exceeded. You currently have 5/5 sandbox(es).",
	} {
		hint := CreateSandboxErrorHint(message, CreateSandboxOptions{Name: "new"})
		if !strings.Contains(hint, "delete") {
			t.Fatalf("%q: expected a hint about freeing the resource, got %q", message, hint)
		}
		if !strings.Contains(hint, "in this session") {
			t.Fatalf("%q: hint suggests a delete without limiting it to this session's sandboxes: %q", message, hint)
		}
		if strings.Contains(message, "quota") {
			continue
		}
		if !strings.Contains(hint, "ask the user") {
			t.Fatalf("%q: hint does not send other deletes to the user: %q", message, hint)
		}
	}
}

// The API answers 429 for both the sandbox count quota and the plan's
// lifetime cap. Only the lifetime cap is fixed by a shorter ttl.
func TestCreateSandboxQuotaErrorsGetTheirOwnNextStep(t *testing.T) {
	const lifetimeHint = "Next step: Your plan tier caps sandbox lifetime"
	const countHint = "Next step: Your account has reached its sandbox limit"
	for _, test := range []struct {
		name, message, want string
		args                map[string]any
	}{
		{
			name:    "sandbox count from the create task",
			message: "Sandbox quota exceeded. You currently have 5/5 sandbox(es).",
			want:    countHint,
		},
		{
			name:    "sandbox count from the quota service",
			message: "Quota exceeded: 5/5 sandboxes",
			want:    countHint,
		},
		{
			name:    "sandbox count without numbers",
			message: "Sandbox quota exceeded",
			args:    map[string]any{"ttl": "10m"},
			want:    countHint,
		},
		{
			name:    "ttl above the plan cap",
			message: "TTL quota exceeded: 7 days. Please upgrade your tier or request additional quota at https://app.blaxel.ai/account/plan-quotas",
			args:    map[string]any{"ttl": "30d"},
			want:    lifetimeHint,
		},
		{
			name:    "expiration date above the plan cap",
			message: "User expiration date is greater than default expiration date: 7 days. Please upgrade your tier or request additional quota at https://app.blaxel.ai/account/plan-quotas",
			want:    lifetimeHint,
		},
		{
			name:    "another quota",
			message: "Quota exceeded: 4096/2048 memory. Please upgrade your tier or request additional quota at https://app.blaxel.ai/account/plan-quotas?id=account",
			args:    map[string]any{"ttl": "10m"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			srv, _ := newLifecycleAPIServer(t, http.StatusTooManyRequests, `{"code":"QUOTA_EXCEEDED","message":"`+test.message+`"}`)
			args := map[string]any{"name": "quota"}
			for key, value := range test.args {
				args[key] = value
			}

			result := callCreateSandboxTool(t, srv, args)
			text := toolResultText(result)
			if !result.IsError || !strings.HasPrefix(text, "failed to create sandbox with status 429: "+test.message) {
				t.Fatalf("expected the API message, got error=%v text=%q", result.IsError, text)
			}
			if test.want == "" {
				if strings.Contains(text, "Next step:") {
					t.Fatalf("unexpected next step for a quota create_sandbox cannot change: %q", text)
				}
				return
			}
			if !strings.Contains(text, test.want) {
				t.Fatalf("got %q, want %q", text, test.want)
			}
		})
	}
}

func TestCreateSandboxConflictNamesItsCause(t *testing.T) {
	cases := []struct {
		name, body, want, hint string
	}{
		{
			name: "name clash without a message",
			body: `{}`,
			want: "sandbox with name 'clash' already exists",
			hint: "Next step: Choose another name, or pass createIfNotExist: true",
		},
		{
			name: "volume attached elsewhere",
			body: `{"code":409,"error":"Conflict","message":"Volume data is already attached to other-sandbox"}`,
			want: "status 409: Volume data is already attached to other-sandbox",
			hint: "Next step: Attach one volume",
		},
		{
			name: "deletion in progress",
			body: `{"code":409,"error":"Conflict","message":"Sandbox deletion is still in progress"}`,
			want: "status 409: Sandbox deletion is still in progress",
			hint: "Next step: A sandbox with this name is still being deleted",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := newLifecycleAPIServer(t, http.StatusConflict, tc.body)
			result := callCreateSandboxTool(t, srv, map[string]any{"name": "clash"})
			text := toolResultText(result)
			if !result.IsError || !strings.Contains(text, tc.want) || !strings.Contains(text, tc.hint) {
				t.Fatalf("got error=%v text=%q, want %q and %q", result.IsError, text, tc.want, tc.hint)
			}
		})
	}
}
