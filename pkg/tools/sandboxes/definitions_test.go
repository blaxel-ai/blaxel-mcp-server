package sandboxes

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestCreateSandboxToolRejectsOutOfRangeAndFractionalMemoryWithoutCallingHandler(t *testing.T) {
	for _, memory := range []any{-100, 1023, 262145, 1024.5} {
		mcpServer, handler := newCreateSandboxToolTestServer()
		result := callCreateSandboxTool(t, mcpServer, map[string]any{
			"name":   "invalid-memory",
			"memory": memory,
		})

		if !result.IsError || !strings.Contains(strings.ToLower(toolResultText(result)), "memory") {
			t.Fatalf("expected actionable memory error for %v, got error=%v text=%q", memory, result.IsError, toolResultText(result))
		}
		if len(handler.createCalls) != 0 {
			t.Fatalf("expected no create call for %v, got %d", memory, len(handler.createCalls))
		}
	}
}

func TestCreateSandboxToolRejectsUnrepresentableMemoryWithoutCallingHandler(t *testing.T) {
	mcpServer, handler := newCreateSandboxToolTestServer()

	result := callCreateSandboxTool(t, mcpServer, map[string]any{
		"name":   "invalid-memory",
		"memory": int64(999999999999),
	})

	if !result.IsError || !strings.Contains(strings.ToLower(toolResultText(result)), "memory") {
		t.Fatalf("expected actionable memory error, got error=%v text=%q", result.IsError, toolResultText(result))
	}
	if len(handler.createCalls) != 0 {
		t.Fatalf("expected no create call, got %d", len(handler.createCalls))
	}
}

func TestCreateSandboxToolRejectsNonFiniteMemoryWithoutCallingHandler(t *testing.T) {
	for _, memory := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		mcpServer, handler := newCreateSandboxToolTestServer()
		result := callCreateSandboxTool(t, mcpServer, map[string]any{
			"name":   "invalid-memory",
			"memory": memory,
		})

		if !result.IsError || !strings.Contains(strings.ToLower(toolResultText(result)), "memory") {
			t.Fatalf("expected actionable memory error for %v, got error=%v text=%q", memory, result.IsError, toolResultText(result))
		}
		if len(handler.createCalls) != 0 {
			t.Fatalf("expected no create call, got %d", len(handler.createCalls))
		}
	}
}

func TestCreateSandboxToolRejectsInvalidPortsWithoutCallingHandler(t *testing.T) {
	for _, ports := range []string{"not-a-port", "0", "65536"} {
		t.Run(ports, func(t *testing.T) {
			mcpServer, handler := newCreateSandboxToolTestServer()
			result := callCreateSandboxTool(t, mcpServer, map[string]any{
				"name":  "invalid-port",
				"ports": ports,
			})

			if !result.IsError || !strings.Contains(strings.ToLower(toolResultText(result)), "port") {
				t.Fatalf("expected actionable port error, got error=%v text=%q", result.IsError, toolResultText(result))
			}
			if len(handler.createCalls) != 0 {
				t.Fatalf("expected no create call, got %d", len(handler.createCalls))
			}
		})
	}
}

func TestCreateSandboxToolRejectsZeroAndNonnumericMemoryWithoutCallingHandler(t *testing.T) {
	for _, memory := range []any{0, "not-memory"} {
		mcpServer, handler := newCreateSandboxToolTestServer()
		result := callCreateSandboxTool(t, mcpServer, map[string]any{
			"name":   "invalid-memory",
			"memory": memory,
		})

		if !result.IsError || !strings.Contains(strings.ToLower(toolResultText(result)), "memory") {
			t.Fatalf("expected actionable memory error for %#v, got error=%v text=%q", memory, result.IsError, toolResultText(result))
		}
		if len(handler.createCalls) != 0 {
			t.Fatalf("expected no create call, got %d", len(handler.createCalls))
		}
	}
}

func TestCreateSandboxToolAllowsOmittedMemoryAndValidValues(t *testing.T) {
	for _, test := range []struct {
		name       string
		args       map[string]any
		wantMemory float64
		wantPorts  string
	}{
		{name: "omitted memory", args: map[string]any{"name": "default-memory"}},
		{name: "minimum memory and ports", args: map[string]any{"name": "minimum", "memory": 1024, "ports": "8080, 65535"}, wantMemory: 1024, wantPorts: "8080, 65535"},
		{name: "maximum memory", args: map[string]any{"name": "maximum", "memory": 262144}, wantMemory: 262144},
	} {
		t.Run(test.name, func(t *testing.T) {
			mcpServer, handler := newCreateSandboxToolTestServer()
			result := callCreateSandboxTool(t, mcpServer, test.args)
			if result.IsError {
				t.Fatalf("expected create_sandbox success, got %q", toolResultText(result))
			}
			if len(handler.createCalls) != 1 {
				t.Fatalf("expected one create call, got %d", len(handler.createCalls))
			}
			got := handler.createCalls[0]
			if got.memory != test.wantMemory {
				t.Fatalf("create call memory = %v, want %v", got.memory, test.wantMemory)
			}
			if got.ports != test.wantPorts {
				t.Fatalf("create call ports = %q, want %q", got.ports, test.wantPorts)
			}
		})
	}
}

func TestCreateSandboxMemorySchemaMatchesUniversalPlatformContract(t *testing.T) {
	mcpServer, _ := newCreateSandboxToolTestServer()
	property := mcpServer.GetTool("create_sandbox").Tool.InputSchema.Properties["memory"].(map[string]any)
	if property["type"] != "integer" || property["minimum"] != 1024 || property["maximum"] != 262144 || property["default"] != 1024 {
		t.Fatalf("memory schema type/minimum/maximum/default = %#v/%#v/%#v/%#v, want integer/1024/262144/1024", property["type"], property["minimum"], property["maximum"], property["default"])
	}
}

type createSandboxCall struct {
	name   string
	image  string
	memory float64
	ports  string
	env    string
}

type createSandboxTestHandler struct {
	createCalls []createSandboxCall
}

func (h *createSandboxTestHandler) ListSandboxes(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (h *createSandboxTestHandler) GetSandbox(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (h *createSandboxTestHandler) CreateSandbox(_ context.Context, name, image string, memory float64, ports, env string) ([]byte, error) {
	h.createCalls = append(h.createCalls, createSandboxCall{name: name, image: image, memory: memory, ports: ports, env: env})
	return []byte(`{"success":true}`), nil
}
func (h *createSandboxTestHandler) DeleteSandbox(context.Context, string) ([]byte, error) {
	return nil, nil
}

func newCreateSandboxToolTestServer() (*server.MCPServer, *createSandboxTestHandler) {
	mcpServer := server.NewMCPServer("sandbox-test", "1.0.0")
	handler := &createSandboxTestHandler{}
	RegisterSandboxTools(mcpServer, handler, &config.Config{})
	return mcpServer, handler
}

func callCreateSandboxTool(t *testing.T, mcpServer *server.MCPServer, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	registered := mcpServer.GetTool("create_sandbox")
	if registered == nil {
		t.Fatal("expected create_sandbox tool")
	}
	result, err := registered.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: args},
	})
	if err != nil {
		t.Fatalf("create_sandbox returned protocol error: %v", err)
	}
	return result
}

func toolResultText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	text, _ := result.Content[0].(mcp.TextContent)
	return text.Text
}
