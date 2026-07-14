package sandboxes

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/toolkit/sdk"
)

func TestSDKHandlerRejectsInvalidMemoryBeforeAPIRequest(t *testing.T) {
	// Explicit zero is rejected by the MCP binder; zero at this compatibility
	// interface means the optional field was omitted.
	for _, memory := range []float64{-100, 999999999999, math.NaN(), math.Inf(1), math.Inf(-1)} {
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

func TestSDKHandlerAllowsValidMemoryAndPorts(t *testing.T) {
	handler, requestCount := newCreateSandboxHandlerTestServer(t)

	memory := 1024.0
	if _, err := handler.CreateSandbox(context.Background(), "valid", "", memory, "8080,65535", ""); err != nil {
		t.Fatalf("expected valid sandbox input to reach API: %v", err)
	}
	if got := requestCount.Load(); got != 1 {
		t.Fatalf("expected one API request, got %d", got)
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
