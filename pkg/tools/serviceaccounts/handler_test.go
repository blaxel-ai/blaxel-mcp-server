package serviceaccounts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/toolkit/sdk"
)

func TestCreateServiceAccountAcceptsCreatedResponse(t *testing.T) {
	const accountName = "test-service-account"
	const clientID = "test-client-id"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/service_accounts" {
			t.Fatalf("expected /service_accounts path, got %s", r.URL.Path)
		}

		var requestBody struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if requestBody.Name != accountName {
			t.Fatalf("expected request body name %q, got %q", accountName, requestBody.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"name":"test-service-account","client_id":"test-client-id"}`))
	}))
	t.Cleanup(server.Close)

	handler, err := NewSDKHandler(&config.Config{
		APIEndpoint: server.URL,
		RunEndpoint: server.URL,
		Workspace:   "test-workspace",
		Credentials: sdk.Credentials{APIKey: "test-api-key"},
	})
	if err != nil {
		t.Fatalf("failed to create service account handler: %v", err)
	}

	result, err := handler.CreateServiceAccount(context.Background(), accountName)
	if err != nil {
		t.Fatalf("expected create service account to accept 201 response: %v", err)
	}

	var decoded struct {
		Success        bool `json:"success"`
		ServiceAccount struct {
			Name         string `json:"name"`
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret,omitempty"`
		} `json:"service_account"`
	}
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Fatalf("failed to decode handler response: %v", err)
	}
	if !decoded.Success {
		t.Fatal("expected success result")
	}
	if decoded.ServiceAccount.Name != accountName {
		t.Fatalf("expected service account name %q, got %q", accountName, decoded.ServiceAccount.Name)
	}
	if decoded.ServiceAccount.ClientID != clientID {
		t.Fatalf("expected client id %q, got %q", clientID, decoded.ServiceAccount.ClientID)
	}
	if decoded.ServiceAccount.ClientSecret != "" {
		t.Fatal("test response should not contain a client secret")
	}
}
