package sandboxes

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
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
