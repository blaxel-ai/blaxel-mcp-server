package mcpservers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/toolkit/sdk"
)

func TestMCPServerExplicitFalseIsAsyncAndReportsAcceptedLifecycle(t *testing.T) {
	const name = "async-mcp"
	var creates, deletes, gets atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			creates.Add(1)
			_, _ = w.Write([]byte(`{"metadata":{"name":"async-mcp"},"status":"DEPLOYING"}`))
		case http.MethodDelete:
			deletes.Add(1)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			gets.Add(1)
			t.Fatalf("omitted wait unexpectedly polled %s", r.URL.Path)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(api.Close)

	handler := newMCPServerTestHandler(t, api.URL)
	created, err := handler.CreateMCPServer(context.Background(), name, "", "", " FaLsE ", nil, nil)
	if err != nil {
		t.Fatalf("async create: %v", err)
	}
	if !strings.Contains(string(created), "creation accepted") || strings.Contains(string(created), "created successfully") {
		t.Fatalf("async create message is inaccurate: %s", created)
	}
	deleted, err := handler.DeleteMCPServer(context.Background(), name, " FaLsE ")
	if err != nil {
		t.Fatalf("async delete: %v", err)
	}
	if !strings.Contains(string(deleted), "deletion initiated") || strings.Contains(string(deleted), "deleted successfully") {
		t.Fatalf("async delete message is inaccurate: %s", deleted)
	}
	if creates.Load() != 1 || deletes.Load() != 1 || gets.Load() != 0 {
		t.Fatalf("request counts create/delete/get = %d/%d/%d, want 1/1/0", creates.Load(), deletes.Load(), gets.Load())
	}
}

func TestCreateMCPServerNameOnlyWaitsForTerminalStatus(t *testing.T) {
	const name = "name-only-mcp"
	var getCount atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/functions":
			var body struct {
				Spec struct {
					IntegrationConnections *[]string `json:"integrationConnections"`
				} `json:"spec"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create request: %v", err)
			}
			if body.Spec.IntegrationConnections != nil {
				t.Fatalf("name-only create sent integration connections: %v", *body.Spec.IntegrationConnections)
			}
			_, _ = w.Write([]byte(`{"metadata":{"name":"name-only-mcp"},"status":"DEPLOYING"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/functions/"+name:
			status := "DELETING"
			if getCount.Add(1) > 1 {
				status = "DEPLOYED"
			}
			_, _ = w.Write([]byte(`{"metadata":{"name":"name-only-mcp"},"status":"` + status + `"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(api.Close)

	handler := newMCPServerTestHandler(t, api.URL)
	result, err := handler.CreateMCPServer(context.Background(), name, "", "", "", nil, nil)
	if err != nil {
		t.Fatalf("name-only create with omitted wait: %v", err)
	}
	if !strings.Contains(string(result), name) {
		t.Fatalf("create result does not identify server: %s", result)
	}
	if got := getCount.Load(); got != 2 {
		t.Fatalf("status GET count = %d, want 2 (DELETING then DEPLOYED)", got)
	}
}

func TestCreateMCPServerWaitReturnsErrorAfterFailedStatus(t *testing.T) {
	const name = "failed-mcp"
	var getCount atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			_, _ = w.Write([]byte(`{"metadata":{"name":"failed-mcp"},"status":"DEPLOYING"}`))
		case http.MethodGet:
			getCount.Add(1)
			_, _ = w.Write([]byte(`{"metadata":{"name":"failed-mcp"},"status":"FAILED"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(api.Close)

	handler := newMCPServerTestHandler(t, api.URL)
	result, err := handler.CreateMCPServer(context.Background(), name, "", "", "true", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "FAILED") {
		t.Fatalf("create ending in FAILED error = %v, want FAILED deployment error", err)
	}
	if result != nil {
		t.Fatalf("create ending in FAILED result = %s, want nil", result)
	}
	if got := getCount.Load(); got != 1 {
		t.Fatalf("status GET count = %d, want 1", got)
	}
}

func TestDeleteMCPServerWaitsForNotFound(t *testing.T) {
	const name = "deleting-mcp"
	var getCount atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			if getCount.Add(1) == 1 {
				_, _ = w.Write([]byte(`{"metadata":{"name":"deleting-mcp"},"status":"DELETING"}`))
				return
			}
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(api.Close)

	handler := newMCPServerTestHandler(t, api.URL)
	if _, err := handler.DeleteMCPServer(context.Background(), name, "true"); err != nil {
		t.Fatalf("delete with wait: %v", err)
	}
	if got := getCount.Load(); got != 2 {
		t.Fatalf("status GET count = %d, want 2 (DELETING then not-found)", got)
	}
}

func TestMCPServerHandlerRejectsInvalidWaitBeforeMutation(t *testing.T) {
	var requests atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	t.Cleanup(api.Close)
	handler := newMCPServerTestHandler(t, api.URL)

	if _, err := handler.CreateMCPServer(context.Background(), "invalid-wait", "", "", "eventually", nil, nil); err == nil || !strings.Contains(err.Error(), "true or false") {
		t.Fatalf("invalid create wait error = %v", err)
	}
	if _, err := handler.DeleteMCPServer(context.Background(), "invalid-wait", "eventually"); err == nil || !strings.Contains(err.Error(), "true or false") {
		t.Fatalf("invalid delete wait error = %v", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("invalid waits made %d API requests, want 0", requests.Load())
	}
}

func TestCreateMCPServerIntegrationModesStillWork(t *testing.T) {
	tests := []struct {
		name               string
		existing           string
		integrationType    string
		wantConnection     string
		wantIntegrationAPI bool
	}{
		{name: "existing connection", existing: "existing-github", wantConnection: "existing-github"},
		{name: "inline integration", integrationType: "github", wantConnection: "integration-mcp-github-integration", wantIntegrationAPI: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var integrationCalls atomic.Int32
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/integrations/connections":
					integrationCalls.Add(1)
					_, _ = w.Write([]byte(`{"metadata":{"name":"integration-mcp-github-integration"},"spec":{"integration":"github"}}`))
				case "/functions":
					var body struct {
						Spec struct {
							IntegrationConnections *[]string `json:"integrationConnections"`
						} `json:"spec"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatalf("decode create request: %v", err)
					}
					if body.Spec.IntegrationConnections == nil || len(*body.Spec.IntegrationConnections) != 1 || (*body.Spec.IntegrationConnections)[0] != tt.wantConnection {
						t.Fatalf("integration connections = %v, want [%s]", body.Spec.IntegrationConnections, tt.wantConnection)
					}
					_, _ = w.Write([]byte(`{"metadata":{"name":"integration-mcp"},"status":"DEPLOYING"}`))
				default:
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
			}))
			t.Cleanup(api.Close)

			handler := newMCPServerTestHandler(t, api.URL)
			if _, err := handler.CreateMCPServer(context.Background(), "integration-mcp", tt.existing, tt.integrationType, "false", nil, nil); err != nil {
				t.Fatalf("create with integration: %v", err)
			}
			wantCalls := int32(0)
			if tt.wantIntegrationAPI {
				wantCalls = 1
			}
			if got := integrationCalls.Load(); got != wantCalls {
				t.Fatalf("integration API calls = %d, want %d", got, wantCalls)
			}
		})
	}
}

func TestMCPServerStatusCheckerFallbackUpdatesLastStatus(t *testing.T) {
	checker := &MCPServerStatusChecker{lastStatus: "FAILED"}
	if got := checker.ExtractStatus(struct{}{}); got != "DEPLOYING" {
		t.Fatalf("ExtractStatus() = %q, want DEPLOYING", got)
	}
	if got := checker.LastStatus(); got != "DEPLOYING" {
		t.Fatalf("LastStatus() = %q, want fallback status DEPLOYING", got)
	}
}

func newMCPServerTestHandler(t *testing.T, endpoint string) MCPServerHandler {
	t.Helper()
	handler, err := NewSDKHandler(&config.Config{
		APIEndpoint: endpoint,
		RunEndpoint: endpoint,
		Workspace:   "test-workspace",
		Credentials: sdk.Credentials{APIKey: "test-api-key"},
	})
	if err != nil {
		t.Fatalf("create SDK handler: %v", err)
	}
	return handler
}
