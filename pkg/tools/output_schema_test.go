package tools_test

import (
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/agents"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/integrations"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/jobs"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/mcpservers"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/modelapis"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/runtime"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/sandboxes"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/serviceaccounts"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/users"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// hostedTools is every tool the hosted Blaxel MCP endpoint serves.
var hostedTools = []string{
	"list_agents", "get_agent", "delete_agent",
	"list_integrations", "get_integration", "create_integration", "delete_integration",
	"list_jobs", "get_job", "delete_job",
	"list_mcp_servers", "get_mcp_server", "create_mcp_server", "delete_mcp_server",
	"list_model_apis", "get_model_api", "create_model_api", "delete_model_api",
	"list_sandboxes", "get_sandbox", "create_sandbox", "delete_sandbox",
	"list_service_accounts", "get_service_account", "create_service_account", "delete_service_account", "update_service_account",
	"list_workspace_users", "get_workspace_user", "invite_workspace_user", "update_workspace_user_role", "remove_workspace_user",
	"run_agent", "run_job", "run_model", "run_sandbox_command",
	"list_sandbox_processes", "get_sandbox_process", "get_sandbox_process_logs", "stop_sandbox_process", "kill_sandbox_process",
}

// outputFixtures holds representative handler responses per tool: the shapes
// this repository's SDK handlers return and the shapes the hosted control
// plane handlers return. Each must validate against the tool's schema.
var outputFixtures = map[string][]string{
	"list_agents": {
		"Found 1 agent(s):\n\nAgent #1:\n  Name: fixture\n  Status: DEPLOYED\n\n",
		"No agents found",
		`[{"metadata":{"name":"fixture","labels":null},"spec":{"runtime":{"image":"img","memory":2048}},"status":"DEPLOYED","events":null}]`,
	},
	"get_agent": {
		`{"metadata":{"name":"fixture","workspace":"ws","labels":{"team":"a"},"createdAt":"2026-01-01T00:00:00Z"},"spec":{"enabled":true,"runtime":{"image":"img","memory":2048,"generation":"mk3","envs":[{"name":"TOKEN","value":"****"}]},"integrationConnections":null},"status":"DEPLOYED","events":[{"type":"deploy"}]}`,
	},
	"delete_agent": {`{"success":true,"message":"Agent 'fixture' deleted successfully"}`},
	"list_integrations": {
		"Found 1 integration(s):\n\nIntegration #1:\n  Name: fixture\n  Secrets: token=***\n",
		`{"content":"Found 1 integration(s):\n\nIntegration #1:\n  Name: fixture\n","count":1}`,
	},
	"get_integration": {
		`{"metadata":{"name":"fixture"},"spec":{"integration":"github","config":{"org":"blaxel"},"secret":{"token":"[REDACTED]"}}}`,
		`{"metadata":{"name":"fixture"},"spec":{"integration":"github","config":null,"secret":{"token":"dum*********use"}}}`,
		`{"metadata":{"name":"fixture"},"spec":{"integration":"github"}}`,
	},
	"create_integration": {
		`{"success":true,"message":"Integration 'fixture' created successfully","integration":{"name":"fixture","type":"github"}}`,
		`{"success":true,"message":"created","integration":{"metadata":{"name":"fixture"},"spec":{"integration":"github","secret":{"token":"dum*********use"}}}}`,
	},
	"delete_integration": {
		`{"success":true,"message":"Integration 'fixture' deleted successfully"}`,
		`{"success":true,"message":"deleted","integration":{"name":"fixture"}}`,
	},
	"list_jobs": {
		"Found 1 job(s):\n\nJob #1:\n  Name: fixture\n",
		`{"content":"Found 1 job(s):\n\nJob #1:\n  Name: fixture\n","count":1}`,
	},
	"get_job": {
		`{"metadata":{"name":"fixture"},"spec":{"runtime":{"image":"img","memory":1024},"triggers":[{"type":"schedule"}]},"status":"DEPLOYED","events":null}`,
	},
	"delete_job": {
		`{"success":true,"message":"Job 'fixture' deleted successfully"}`,
		`{"success":true,"message":"deleted","job":{"name":"fixture","status":"DELETING"}}`,
	},
	"list_mcp_servers": {
		"Found 1 MCP server(s):\n\nMCP Server #1:\n  Name: fixture\n",
		`{"content":"Found 1 MCP server(s)","count":1}`,
	},
	"get_mcp_server": {
		`{"metadata":{"name":"fixture","url":"https://run.blaxel.ai/ws/functions/fixture"},"spec":{"runtime":{"type":"mcp","image":"img"},"integrationConnections":["github"]},"status":"DEPLOYED","events":[]}`,
	},
	"create_mcp_server": {
		`{"success":true,"message":"MCP server 'fixture' created and deployed successfully","mcp_server":{"name":"fixture","status":"DEPLOYED","integrationConnection":"github","integrationType":"github"}}`,
		`{"success":true,"message":"creation accepted","server":{"metadata":{"name":"fixture"},"spec":{"runtime":{"type":"mcp"}},"status":"DEPLOYING","events":null}}`,
	},
	"delete_mcp_server": {
		`{"success":true,"message":"MCP server 'fixture' deleted successfully"}`,
		`{"success":true,"message":"deletion initiated","server":{"name":"fixture","status":"DELETING"}}`,
	},
	"list_model_apis": {
		"Found 1 model API(s):\n\nModel API #1:\n  Name: fixture\n",
		"No model APIs found",
	},
	"get_model_api": {
		`{"metadata":{"name":"fixture"},"spec":{"runtime":{"type":"openai","model":"gpt-4o"}},"status":"DEPLOYED","events":null}`,
	},
	"create_model_api": {
		`{"success":true,"message":"Model API 'fixture' creation accepted","model_api":{"name":"fixture","model":"gpt-4o","endpoint":"https://api.openai.com","provider":"openai","integrationConnection":"openai"}}`,
		`{"success":true,"message":"creation accepted","model":{"metadata":{"name":"fixture"},"spec":{"runtime":{"model":"gpt-4o"}},"status":"DEPLOYING","events":null}}`,
	},
	"delete_model_api": {`{"success":true,"message":"Model API 'fixture' deletion initiated"}`},
	"list_sandboxes": {
		"Found 1 sandbox(es):\n\nSandbox #1:\n  Name: fixture\n",
		"Found 1 sandbox(es):\n\nSandbox #1:\n  Name: fixture\n\n\nShowing the first 1 of more than 1 matching sandboxes. To see the next page, call list_sandboxes again with cursor=abc",
	},
	"get_sandbox": {
		`{"metadata":{"name":"fixture"},"spec":{"region":"us-pdx-1","runtime":{"image":"blaxel/base:latest","memory":4096,"ports":[{"target":3000}]}},"status":"DEPLOYED","lastUsedAt":"2026-01-01T00:00:00Z"}`,
	},
	"create_sandbox": {
		`{"success":true,"message":"Sandbox 'fixture' created successfully","sandbox":{"name":"fixture","status":"DEPLOYING"}}`,
		`{"success":true,"message":"Sandbox 'fixture' created successfully","sandbox":{"metadata":{"name":"fixture"},"spec":{"runtime":{"image":"img"}},"status":"DEPLOYING"}}`,
	},
	"delete_sandbox": {`{"success":true,"message":"Sandbox 'fixture' deleted successfully"}`},
	"list_service_accounts": {
		"Found 1 service account(s):\n\nService Account #1:\n  Name: fixture\n  Client ID: cid\n",
		`{"content":"Found 1 service account(s)","count":1}`,
	},
	"get_service_account": {
		`{"client_id":"cid","name":"fixture","description":"d","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`,
		`{"client_id":"cid","name":"fixture","client_secret":"****abcd","redirect_uris":null,"created_at":"2026-01-01T00:00:00Z","createdBy":"u"}`,
	},
	"create_service_account": {
		`{"success":true,"message":"Service account 'fixture' created successfully. Save the client_secret securely because it will not be shown again.","service_account":{"name":"fixture","client_id":"cid","client_secret":"one-time-secret"}}`,
		`{"success":true,"message":"Service Account created successfully","serviceAccount":{"client_id":"cid","client_secret":"one-time-secret","workspace":"ws","name":"fixture","redirect_uris":null,"createdAt":"2026-01-01T00:00:00Z"}}`,
	},
	"delete_service_account": {
		`{"success":true,"message":"Service account with client ID 'cid' deleted successfully"}`,
		`{"success":true,"message":"deleted","serviceAccount":{"clientId":"cid"}}`,
	},
	"update_service_account": {
		`{"success":true,"message":"Service account 'cid' updated successfully","service_account":{"client_id":"cid","name":"renamed"}}`,
		`{"success":true,"message":"updated","serviceAccount":{"client_id":"cid","name":"renamed","client_secret":"****abcd"}}`,
	},
	"list_workspace_users": {
		`{"users":[{"email":"user@example.com","sub":"s","name":"A User","role":"admin","accepted":true,"email_verified":true}],"count":1}`,
		`{"users":null,"count":0}`,
		`{"content":"Found 1 user(s)","count":1}`,
	},
	"get_workspace_user": {
		`{"user":{"email":"user@example.com","sub":"s","role":"member","accepted":true}}`,
		`{"sub":"s","given_name":"A","family_name":"User","email":"user@example.com","email_verified":true,"role":"member","accepted":true,"mfa_enabled":false}`,
	},
	"invite_workspace_user": {
		`{"success":true,"message":"Invited 'user@example.com' to the workspace as member.","email":"user@example.com","role":"member"}`,
		`{"success":true,"message":"invited","invitation":{"workspace":"ws","email":"user@example.com","invitedBy":"admin","role":"member","expiresAt":"2026-01-08T00:00:00Z"}}`,
	},
	"update_workspace_user_role": {
		`{"success":true,"message":"Successfully updated role for user 'user@example.com' to 'admin'"}`,
		`{"success":true,"message":"updated","user":{"email":"user@example.com","role":"admin","accepted":true}}`,
	},
	"remove_workspace_user": {`{"success":true,"message":"Successfully removed user 'user@example.com' from the workspace"}`},
	"run_agent": {
		"{\n  \"output\": \"hello\"\n}",
		"plain text reply",
		`["streamed","chunks"]`,
	},
	"run_job": {
		"Job triggered successfully:\n{\n  \"id\": \"execution-1\"\n}",
		`{"id":"execution-1","status":"PENDING"}`,
	},
	"run_model": {
		`{"id":"chatcmpl-1","choices":[{"message":{"role":"assistant","content":"hi"}}]}`,
	},
	"run_sandbox_command": {
		`{"pid":"42","name":"dev","command":"echo hi","status":"completed","exitCode":0,"workingDir":"/","startedAt":"s","completedAt":"c","logs":"hi","stdout":"hi","stderr":"","keepAlive":false,"restartOnFailure":false,"maxRestarts":0,"restartCount":0}`,
	},
	"list_sandbox_processes": {
		`[{"pid":"42","name":"dev","command":"echo hi","status":"running"}]`,
		`[]`,
	},
	"get_sandbox_process": {`{"pid":"42","name":"dev","command":"sleep 10","status":"running","exitCode":0}`},
	"get_sandbox_process_logs": {
		`{"logs":"one\ntwo\nthree","stdout":"one\ntwo\nthree","stderr":""}`,
		"not json logs",
	},
	"stop_sandbox_process": {`{"message":"Process stopped","path":"/process/42"}`},
	"kill_sandbox_process": {`{"message":"Process killed","path":"/process/42/kill"}`},
}

// toolArguments returns arguments that pass each tool's own validation.
func toolArguments(name string) map[string]any {
	args := map[string]any{
		"name":            "fixture",
		"id":              "fixture",
		"email":           "user@example.com",
		"role":            "member",
		"client_id":       "cid",
		"integrationType": "github",
		"command":         "echo hi",
		"identifier":      "42",
	}
	switch name {
	case "run_agent":
		args["message"] = "hello"
	case "run_model":
		args["body"] = map[string]any{"messages": []any{}}
	}
	return args
}

func TestHostedToolsDeclareOutputSchemas(t *testing.T) {
	s := newOutputSchemaTestServer(&fixtureHandler{})

	registered := s.ListTools()
	if len(registered) != len(hostedTools) {
		names := make([]string, 0, len(registered))
		for name := range registered {
			names = append(names, name)
		}
		sort.Strings(names)
		t.Fatalf("registered %d tools, want %d: %v", len(registered), len(hostedTools), names)
	}

	for _, name := range hostedTools {
		t.Run(name, func(t *testing.T) {
			tool := s.GetTool(name)
			if tool == nil {
				t.Fatalf("tool %q is not registered", name)
			}
			schema := compileOutputSchema(t, tool.Tool)
			if schema == nil {
				t.Fatalf("tool %q has no output schema", name)
			}

			// The schema must reach clients in tools/list.
			encoded, err := json.Marshal(tool.Tool)
			if err != nil {
				t.Fatalf("marshal tool: %v", err)
			}
			var listed struct {
				OutputSchema map[string]any `json:"outputSchema"`
			}
			if err := json.Unmarshal(encoded, &listed); err != nil {
				t.Fatalf("unmarshal tool: %v", err)
			}
			if listed.OutputSchema["type"] != "object" {
				t.Fatalf("tool %q outputSchema.type = %v, want object", name, listed.OutputSchema["type"])
			}
			if _, ok := outputFixtures[name]; !ok {
				t.Fatalf("tool %q has no output fixtures", name)
			}
		})
	}
}

func TestHostedToolResultsMatchOutputSchemas(t *testing.T) {
	handler := &fixtureHandler{}
	s := newOutputSchemaTestServer(handler)

	for _, name := range hostedTools {
		tool := s.GetTool(name)
		if tool == nil {
			t.Fatalf("tool %q is not registered", name)
		}
		schema := compileOutputSchema(t, tool.Tool)

		for i, payload := range outputFixtures[name] {
			t.Run(name+"/"+string(rune('a'+i)), func(t *testing.T) {
				handler.payload = payload
				result := callTool(t, tool, toolArguments(name))

				text := resultText(t, result)
				if text != payload {
					t.Fatalf("text content changed:\n got %q\nwant %q", text, payload)
				}
				validateStructured(t, schema, result.StructuredContent)
			})
		}
	}
}

func TestStructuredContentKeepsResponseFields(t *testing.T) {
	handler := &fixtureHandler{}
	s := newOutputSchemaTestServer(handler)

	cases := []struct {
		tool    string
		payload string
		check   func(t *testing.T, structured map[string]any)
	}{
		{
			tool:    "get_agent",
			payload: `{"metadata":{"name":"fixture"},"status":"DEPLOYED"}`,
			check: func(t *testing.T, structured map[string]any) {
				metadata, _ := structured["metadata"].(map[string]any)
				if metadata["name"] != "fixture" || structured["status"] != "DEPLOYED" {
					t.Fatalf("structured = %v", structured)
				}
			},
		},
		{
			tool:    "list_sandbox_processes",
			payload: `[{"pid":"1"},{"pid":"2"}]`,
			check: func(t *testing.T, structured map[string]any) {
				items, _ := structured["items"].([]any)
				if len(items) != 2 || structured["count"] != 2 {
					t.Fatalf("structured = %v", structured)
				}
			},
		},
		{
			tool:    "list_agents",
			payload: "No agents found",
			check: func(t *testing.T, structured map[string]any) {
				if structured["text"] != "No agents found" {
					t.Fatalf("structured = %v", structured)
				}
			},
		},
		{
			tool:    "run_agent",
			payload: "plain text reply",
			check: func(t *testing.T, structured map[string]any) {
				if structured["format"] != "text" || structured["body"] != "plain text reply" {
					t.Fatalf("structured = %v", structured)
				}
			},
		},
		{
			tool:    "run_job",
			payload: "Job triggered successfully:\n{\"id\":\"execution-1\"}",
			check: func(t *testing.T, structured map[string]any) {
				body, _ := structured["body"].(map[string]any)
				if structured["format"] != "json" || body["id"] != "execution-1" {
					t.Fatalf("structured = %v", structured)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			handler.payload = tc.payload
			result := callTool(t, s.GetTool(tc.tool), toolArguments(tc.tool))
			structured, ok := result.StructuredContent.(map[string]any)
			if !ok {
				t.Fatalf("structured content is %T, want an object", result.StructuredContent)
			}
			tc.check(t, structured)
		})
	}
}

// Structured content comes from the same bytes as the text block, so a masked
// secret stays masked and a secret absent from the text never appears.
func TestStructuredContentNeverAddsSecrets(t *testing.T) {
	handler := &fixtureHandler{}
	s := newOutputSchemaTestServer(handler)

	for _, tc := range []struct{ tool, payload, masked string }{
		{"get_integration", `{"metadata":{"name":"fixture"},"spec":{"integration":"github","secret":{"token":"[REDACTED]"}}}`, "[REDACTED]"},
		{"get_service_account", `{"client_id":"cid","name":"fixture","client_secret":"****abcd"}`, "****abcd"},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			handler.payload = tc.payload
			result := callTool(t, s.GetTool(tc.tool), toolArguments(tc.tool))
			encoded, err := json.Marshal(result.StructuredContent)
			if err != nil {
				t.Fatalf("marshal structured content: %v", err)
			}
			if !strings.Contains(string(encoded), tc.masked) {
				t.Fatalf("structured content %s lost the masked value %q", encoded, tc.masked)
			}
		})
	}
}

func TestSandboxLogTailTrimsStructuredContent(t *testing.T) {
	handler := &fixtureHandler{payload: `{"logs":"one\ntwo\nthree","stdout":"one\ntwo\nthree","stderr":""}`}
	s := newOutputSchemaTestServer(handler)
	tool := s.GetTool("get_sandbox_process_logs")

	args := toolArguments("get_sandbox_process_logs")
	args["tail"] = "1"
	result := callTool(t, tool, args)

	validateStructured(t, compileOutputSchema(t, tool.Tool), result.StructuredContent)
	structured := result.StructuredContent.(map[string]any)
	if structured["logs"] != "three" || structured["stdout"] != "three" || structured["truncated"] != true {
		t.Fatalf("structured = %v", structured)
	}
}

func TestErrorResultsCarryNoStructuredContent(t *testing.T) {
	s := newOutputSchemaTestServer(&fixtureHandler{})
	result := callTool(t, s.GetTool("get_agent"), map[string]any{})
	if !result.IsError {
		t.Fatal("expected an error result when name is missing")
	}
	if result.StructuredContent != nil {
		t.Fatalf("error result has structured content %v", result.StructuredContent)
	}
}

func newOutputSchemaTestServer(handler *fixtureHandler) *server.MCPServer {
	s := server.NewMCPServer("output-schema-test", "1.0.0")
	cfg := &config.Config{}

	agents.RegisterAgentTools(s, handler, cfg)
	integrations.RegisterIntegrationTools(s, handler, cfg)
	jobs.RegisterJobTools(s, handler, cfg)
	mcpservers.RegisterMCPServerTools(s, handler, cfg)
	modelapis.RegisterModelAPITools(s, handler, cfg)
	sandboxes.RegisterSandboxTools(s, handler, cfg)
	serviceaccounts.RegisterServiceAccountTools(s, handler, cfg)
	users.RegisterUserTools(s, handler, cfg)
	runtime.RegisterRuntimeTools(s, handler, cfg)

	return s
}

func compileOutputSchema(t *testing.T, tool mcp.Tool) *jsonschema.Schema {
	t.Helper()
	if len(tool.RawOutputSchema) == 0 {
		return nil
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(tool.RawOutputSchema))
	if err != nil {
		t.Fatalf("tool %q output schema is not JSON: %v", tool.Name, err)
	}
	compiler := jsonschema.NewCompiler()
	url := "urn:blaxel-mcp:" + tool.Name + ":output"
	if err := compiler.AddResource(url, doc); err != nil {
		t.Fatalf("tool %q output schema: %v", tool.Name, err)
	}
	schema, err := compiler.Compile(url)
	if err != nil {
		t.Fatalf("tool %q output schema does not compile: %v", tool.Name, err)
	}
	return schema
}

func callTool(t *testing.T, tool *server.ServerTool, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	var request mcp.CallToolRequest
	request.Params.Name = tool.Tool.Name
	request.Params.Arguments = args
	result, err := tool.Handler(context.Background(), request)
	if err != nil {
		t.Fatalf("tool %q returned error: %v", tool.Tool.Name, err)
	}
	if result == nil {
		t.Fatalf("tool %q returned no result", tool.Tool.Name)
	}
	return result
}

func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result.IsError {
		t.Fatalf("unexpected error result: %v", result.Content)
	}
	if len(result.Content) != 1 {
		t.Fatalf("got %d content blocks, want 1", len(result.Content))
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content is %T, want text", result.Content[0])
	}
	return text.Text
}

func validateStructured(t *testing.T, schema *jsonschema.Schema, structured any) {
	t.Helper()
	if structured == nil {
		t.Fatal("result has no structured content")
	}
	encoded, err := json.Marshal(structured)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	if !bytes.HasPrefix(bytes.TrimSpace(encoded), []byte("{")) {
		t.Fatalf("structured content %s is not a JSON object", encoded)
	}
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("decode structured content: %v", err)
	}
	if err := schema.Validate(value); err != nil {
		t.Fatalf("structured content %s does not match output schema: %v", encoded, err)
	}
}

// fixtureHandler implements every hosted handler interface and returns the
// same payload from each method.
type fixtureHandler struct{ payload string }

func (h *fixtureHandler) bytes() ([]byte, error) { return []byte(h.payload), nil }

func (h *fixtureHandler) ListAgents(context.Context, string) ([]byte, error)  { return h.bytes() }
func (h *fixtureHandler) GetAgent(context.Context, string) ([]byte, error)    { return h.bytes() }
func (h *fixtureHandler) DeleteAgent(context.Context, string) ([]byte, error) { return h.bytes() }

func (h *fixtureHandler) ListIntegrations(context.Context, string) ([]byte, error) { return h.bytes() }
func (h *fixtureHandler) GetIntegration(context.Context, string) ([]byte, error)   { return h.bytes() }
func (h *fixtureHandler) CreateIntegration(context.Context, string, string, map[string]string, map[string]string) ([]byte, error) {
	return h.bytes()
}
func (h *fixtureHandler) DeleteIntegration(context.Context, string) ([]byte, error) { return h.bytes() }

func (h *fixtureHandler) ListJobs(context.Context, string) ([]byte, error)  { return h.bytes() }
func (h *fixtureHandler) GetJob(context.Context, string) ([]byte, error)    { return h.bytes() }
func (h *fixtureHandler) DeleteJob(context.Context, string) ([]byte, error) { return h.bytes() }

func (h *fixtureHandler) ListMCPServers(context.Context, string) ([]byte, error) { return h.bytes() }
func (h *fixtureHandler) GetMCPServer(context.Context, string) ([]byte, error)   { return h.bytes() }
func (h *fixtureHandler) CreateMCPServer(context.Context, string, string, string, string, map[string]string, map[string]string) ([]byte, error) {
	return h.bytes()
}
func (h *fixtureHandler) DeleteMCPServer(context.Context, string, string) ([]byte, error) {
	return h.bytes()
}

func (h *fixtureHandler) ListModelAPIs(context.Context, string) ([]byte, error) { return h.bytes() }
func (h *fixtureHandler) GetModelAPI(context.Context, string) ([]byte, error)   { return h.bytes() }
func (h *fixtureHandler) CreateModelAPI(context.Context, string, string, string, string, string, string, string, map[string]interface{}) ([]byte, error) {
	return h.bytes()
}
func (h *fixtureHandler) DeleteModelAPI(context.Context, string, string) ([]byte, error) {
	return h.bytes()
}

func (h *fixtureHandler) ListSandboxes(context.Context, string) ([]byte, error) { return h.bytes() }
func (h *fixtureHandler) GetSandbox(context.Context, string) ([]byte, error)    { return h.bytes() }
func (h *fixtureHandler) CreateSandbox(context.Context, string, string, float64, string, string) ([]byte, error) {
	return h.bytes()
}
func (h *fixtureHandler) DeleteSandbox(context.Context, string) ([]byte, error) { return h.bytes() }

func (h *fixtureHandler) ListServiceAccounts(context.Context, string) ([]byte, error) {
	return h.bytes()
}
func (h *fixtureHandler) GetServiceAccount(context.Context, string) ([]byte, error) {
	return h.bytes()
}
func (h *fixtureHandler) CreateServiceAccount(context.Context, string) ([]byte, error) {
	return h.bytes()
}
func (h *fixtureHandler) DeleteServiceAccount(context.Context, string) ([]byte, error) {
	return h.bytes()
}
func (h *fixtureHandler) UpdateServiceAccount(context.Context, string, string) ([]byte, error) {
	return h.bytes()
}

func (h *fixtureHandler) ListUsers(context.Context, string) ([]byte, error) { return h.bytes() }
func (h *fixtureHandler) GetUser(context.Context, string) ([]byte, error)   { return h.bytes() }
func (h *fixtureHandler) InviteUser(context.Context, string, string) ([]byte, error) {
	return h.bytes()
}
func (h *fixtureHandler) UpdateUserRole(context.Context, string, string) ([]byte, error) {
	return h.bytes()
}
func (h *fixtureHandler) RemoveUser(context.Context, string) ([]byte, error) { return h.bytes() }

func (h *fixtureHandler) RunAgent(context.Context, string, string, string) (string, error) {
	return h.payload, nil
}
func (h *fixtureHandler) RunJob(context.Context, string, string) (string, error) {
	return h.payload, nil
}
func (h *fixtureHandler) RunModel(context.Context, string, string, string, string) (string, error) {
	return h.payload, nil
}
func (h *fixtureHandler) RunSandbox(context.Context, string, string, string, string) (string, error) {
	return h.payload, nil
}

// The schemas must still reject results that break their contract, or the
// validation above would prove nothing.
func TestOutputSchemasRejectMismatchedResults(t *testing.T) {
	s := newOutputSchemaTestServer(&fixtureHandler{})

	for _, tc := range []struct{ tool, structured string }{
		{"delete_agent", `{"message":"missing success"}`},
		{"get_agent", `{"status":42}`},
		{"list_workspace_users", `{"users":"not a list"}`},
		{"run_agent", `{"body":"missing format"}`},
		{"run_model", `{"format":"xml","body":""}`},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			schema := compileOutputSchema(t, s.GetTool(tc.tool).Tool)
			value, err := jsonschema.UnmarshalJSON(strings.NewReader(tc.structured))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if err := schema.Validate(value); err == nil {
				t.Fatalf("schema accepted %s", tc.structured)
			}
		})
	}
}
