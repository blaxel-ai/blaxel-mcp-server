package modelapis

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

func TestModelAPIExplicitFalseIsAsyncAndReportsAcceptedLifecycle(t *testing.T) {
	const name = "async-model"
	var creates, deletes, statusGets atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/integrations/connections/existing-openai":
			_, _ = w.Write([]byte(`{"metadata":{"name":"existing-openai"},"spec":{"integration":"openai"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/models":
			creates.Add(1)
			_, _ = w.Write([]byte(`{"metadata":{"name":"async-model"},"status":"DEPLOYING"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/models/"+name:
			deletes.Add(1)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/models/"+name:
			statusGets.Add(1)
			t.Fatalf("omitted wait unexpectedly polled %s", r.URL.Path)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(api.Close)

	handler := newModelAPITestHandler(t, api.URL)
	created, err := handler.CreateModelAPI(context.Background(), name, "gpt-4o", "", "existing-openai", "", "", " FaLsE ", nil)
	if err != nil {
		t.Fatalf("async create: %v", err)
	}
	if !strings.Contains(string(created), "creation accepted") || strings.Contains(string(created), "created successfully") {
		t.Fatalf("async create message is inaccurate: %s", created)
	}
	deleted, err := handler.DeleteModelAPI(context.Background(), name, " FaLsE ")
	if err != nil {
		t.Fatalf("async delete: %v", err)
	}
	if !strings.Contains(string(deleted), "deletion initiated") || strings.Contains(string(deleted), "deleted successfully") {
		t.Fatalf("async delete message is inaccurate: %s", deleted)
	}
	if creates.Load() != 1 || deletes.Load() != 1 || statusGets.Load() != 0 {
		t.Fatalf("request counts create/delete/status-get = %d/%d/%d, want 1/1/0", creates.Load(), deletes.Load(), statusGets.Load())
	}
}

func TestCreateModelAPIWaitsForTerminalStatus(t *testing.T) {
	const name = "waiting-model"
	var getCount atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/integrations/connections/existing-openai":
			_, _ = w.Write([]byte(`{"metadata":{"name":"existing-openai"},"spec":{"integration":"openai"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/models":
			_, _ = w.Write([]byte(`{"metadata":{"name":"waiting-model"},"status":"DEPLOYING"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/models/"+name:
			status := "DEPLOYING"
			if getCount.Add(1) > 1 {
				status = "DEPLOYED"
			}
			_, _ = w.Write([]byte(`{"metadata":{"name":"waiting-model"},"status":"` + status + `"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(api.Close)

	handler := newModelAPITestHandler(t, api.URL)
	result, err := handler.CreateModelAPI(context.Background(), name, "gpt-4o", "", "existing-openai", "", "", "", nil)
	if err != nil {
		t.Fatalf("create with omitted wait: %v", err)
	}
	if !strings.Contains(string(result), `"status": "DEPLOYED"`) {
		t.Fatalf("create result does not report terminal status: %s", result)
	}
	if got := getCount.Load(); got != 2 {
		t.Fatalf("status GET count = %d, want 2 (DEPLOYING then DEPLOYED)", got)
	}
}

func TestCreateModelAPIWaitReturnsErrorAfterFailedStatus(t *testing.T) {
	const name = "failed-model"
	var getCount atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/integrations/connections/existing-openai":
			_, _ = w.Write([]byte(`{"metadata":{"name":"existing-openai"},"spec":{"integration":"openai"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/models":
			_, _ = w.Write([]byte(`{"metadata":{"name":"failed-model"},"status":"DEPLOYING"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/models/"+name:
			getCount.Add(1)
			_, _ = w.Write([]byte(`{"metadata":{"name":"failed-model"},"status":"FAILED"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(api.Close)

	handler := newModelAPITestHandler(t, api.URL)
	result, err := handler.CreateModelAPI(context.Background(), name, "gpt-4o", "", "existing-openai", "", "", "true", nil)
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

func TestModelAPIStatusCheckerRejectsPollingHTTPFailure(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/models/broken-model" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		http.Error(w, `{"message":"unavailable"}`, http.StatusServiceUnavailable)
	}))
	t.Cleanup(api.Close)

	handler := newModelAPITestHandler(t, api.URL).(*SDKHandler)
	checker := NewModelAPIStatusChecker(handler.sdkClient)
	if _, err := checker.GetResource(context.Background(), "broken-model"); err == nil || !strings.Contains(err.Error(), "status 503") {
		t.Fatalf("polling error = %v, want actionable status 503 error", err)
	}
	if checker.LastStatus() != "" {
		t.Fatalf("polling failure recorded terminal status %q", checker.LastStatus())
	}
}

func TestDeleteModelAPIWaitsForNotFound(t *testing.T) {
	const name = "deleting-model"
	var getCount atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/models/"+name:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/models/"+name:
			if getCount.Add(1) == 1 {
				_, _ = w.Write([]byte(`{"metadata":{"name":"deleting-model"},"status":"DELETING"}`))
				return
			}
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(api.Close)

	handler := newModelAPITestHandler(t, api.URL)
	if _, err := handler.DeleteModelAPI(context.Background(), name, "true"); err != nil {
		t.Fatalf("delete with wait: %v", err)
	}
	if got := getCount.Load(); got != 2 {
		t.Fatalf("status GET count = %d, want 2 (DELETING then not-found)", got)
	}
}

func TestModelAPIHandlerRejectsInvalidWaitBeforeMutation(t *testing.T) {
	var requests atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	t.Cleanup(api.Close)
	handler := newModelAPITestHandler(t, api.URL)

	if _, err := handler.CreateModelAPI(context.Background(), "invalid-wait", "gpt-4o", "", "existing-openai", "", "", "eventually", nil); err == nil || !strings.Contains(err.Error(), "true or false") {
		t.Fatalf("invalid create wait error = %v", err)
	}
	if _, err := handler.DeleteModelAPI(context.Background(), "invalid-wait", "eventually"); err == nil || !strings.Contains(err.Error(), "true or false") {
		t.Fatalf("invalid delete wait error = %v", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("invalid waits made %d API requests, want 0", requests.Load())
	}
}

func TestCreateModelAPIWaitFalsePreservesInlineProviderBehavior(t *testing.T) {
	const name = "inline-model"
	var integrationCalls, modelGets atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/integrations/connections":
			integrationCalls.Add(1)
			var body struct {
				Spec struct {
					Integration *string            `json:"integration"`
					Secret      *map[string]string `json:"secret"`
					Config      *map[string]string `json:"config"`
				} `json:"spec"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode integration request: %v", err)
			}
			if body.Spec.Integration == nil || *body.Spec.Integration != "openai" || body.Spec.Secret == nil || (*body.Spec.Secret)["apiKey"] != "secret" || body.Spec.Config == nil || (*body.Spec.Config)["baseURL"] != "https://api.example.test/v1" {
				t.Fatalf("unexpected inline integration request: %+v", body.Spec)
			}
			_, _ = w.Write([]byte(`{"metadata":{"name":"inline-model-openai-integration"},"spec":{"integration":"openai"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/models":
			var body struct {
				Spec struct {
					IntegrationConnections *[]string `json:"integrationConnections"`
					Runtime                struct {
						EndpointName *string `json:"endpointName"`
						Type         *string `json:"type"`
					} `json:"runtime"`
				} `json:"spec"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode model request: %v", err)
			}
			if body.Spec.Runtime.EndpointName == nil || *body.Spec.Runtime.EndpointName != "provider-endpoint" || body.Spec.Runtime.Type == nil || *body.Spec.Runtime.Type != "openai" || body.Spec.IntegrationConnections == nil || len(*body.Spec.IntegrationConnections) != 1 || (*body.Spec.IntegrationConnections)[0] != "inline-model-openai-integration" {
				t.Fatalf("unexpected inline model request: %+v", body.Spec)
			}
			_, _ = w.Write([]byte(`{"metadata":{"name":"inline-model"},"status":"DEPLOYING"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/models/"+name:
			modelGets.Add(1)
			_, _ = w.Write([]byte(`{"metadata":{"name":"inline-model"},"status":"DEPLOYED"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(api.Close)

	handler := newModelAPITestHandler(t, api.URL)
	result, err := handler.CreateModelAPI(context.Background(), name, "gpt-4o", "provider-endpoint", "", "openai", "secret", "false", map[string]interface{}{"baseURL": "https://api.example.test/v1"})
	if err != nil {
		t.Fatalf("create without wait: %v", err)
	}
	if integrationCalls.Load() != 1 {
		t.Fatalf("integration calls = %d, want 1", integrationCalls.Load())
	}
	if modelGets.Load() != 0 {
		t.Fatalf("wait=false made %d status GETs, want 0", modelGets.Load())
	}
	if !strings.Contains(string(result), `"provider": "openai"`) {
		t.Fatalf("create result does not retain provider: %s", result)
	}
}

func TestCreateModelAPIRejectsAmbiguousIntegrationModesBeforeMutation(t *testing.T) {
	var requests atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	t.Cleanup(api.Close)
	handler := newModelAPITestHandler(t, api.URL)

	for _, test := range []struct {
		name, existing, provider, apiKey string
		config                           map[string]interface{}
	}{
		{name: "mixed modes", existing: "existing", provider: "openai", apiKey: "secret"},
		{name: "provider missing key", provider: "openai"},
		{name: "key missing provider", apiKey: "secret"},
		{name: "orphan config", config: map[string]interface{}{"baseURL": "https://api.example.test"}},
		{name: "config with existing", existing: "existing", config: map[string]interface{}{"baseURL": "https://api.example.test"}},
		{name: "non-string config", provider: "openai", apiKey: "secret", config: map[string]interface{}{"timeout": float64(30)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := handler.CreateModelAPI(context.Background(), "model-one", "gpt-4o", "", test.existing, test.provider, test.apiKey, "false", test.config); err == nil {
				t.Fatal("CreateModelAPI() error = nil, want invalid integration-mode error")
			}
		})
	}
	if requests.Load() != 0 {
		t.Fatalf("invalid integration modes made %d remote requests, want 0", requests.Load())
	}
}

func TestCreateModelAPIRollsBackOwnedInlineIntegrationWhenModelCreationFails(t *testing.T) {
	var deletes atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/integrations/connections":
			_, _ = w.Write([]byte(`{"metadata":{"name":"model-one-openai-integration"},"spec":{"integration":"openai"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/models":
			http.Error(w, `{"message":"model rejected"}`, http.StatusBadRequest)
		case r.Method == http.MethodDelete && r.URL.Path == "/integrations/connections/model-one-openai-integration":
			deletes.Add(1)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(api.Close)

	_, err := newModelAPITestHandler(t, api.URL).CreateModelAPI(context.Background(), "model-one", "gpt-4o", "", "", "openai", "secret", "false", nil)
	if err == nil || !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("CreateModelAPI() error = %v, want model status 400", err)
	}
	if deletes.Load() != 1 {
		t.Fatalf("inline integration delete calls = %d, want 1", deletes.Load())
	}
}

func newModelAPITestHandler(t *testing.T, endpoint string) ModelAPIHandler {
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
