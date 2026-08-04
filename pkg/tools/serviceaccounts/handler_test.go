package serviceaccounts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/toolkit/sdk"
)

func TestUpdateServiceAccountRenamesByClientID(t *testing.T) {
	const clientID = "test-client-id"
	const newName = "Renamed service account"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT request, got %s", r.Method)
		}
		if r.URL.Path != "/service_accounts/"+clientID {
			t.Fatalf("expected service account path, got %s", r.URL.Path)
		}

		var requestBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if want := map[string]any{"name": newName}; !mapsEqual(requestBody, want) {
			t.Fatalf("request body = %#v, want %#v", requestBody, want)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"client_id":"test-client-id","client_secret":"must-not-leak","name":"Renamed service account"}`))
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

	result, err := handler.UpdateServiceAccount(context.Background(), clientID, newName)
	if err != nil {
		t.Fatalf("update service account failed: %v", err)
	}

	var decoded struct {
		Success        bool           `json:"success"`
		ServiceAccount map[string]any `json:"service_account"`
	}
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Fatalf("failed to decode handler response: %v", err)
	}
	if !decoded.Success {
		t.Fatal("expected success result")
	}
	wantAccount := map[string]any{"client_id": clientID, "name": newName}
	if !mapsEqual(decoded.ServiceAccount, wantAccount) {
		t.Fatalf("service account response = %#v, want %#v", decoded.ServiceAccount, wantAccount)
	}
}

func mapsEqual(got, want map[string]any) bool {
	if len(got) != len(want) {
		return false
	}
	for key, wantValue := range want {
		if got[key] != wantValue {
			return false
		}
	}
	return true
}

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

	result, err := handler.CreateServiceAccount(context.Background(), accountName, false)
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

func TestCreateServiceAccountSecretDisclosureRequiresOptIn(t *testing.T) {
	const accountName = "test-service-account"
	const clientID = "test-client-id"
	const clientSecret = "blx_live-client-secret"

	for _, test := range []struct {
		name         string
		revealSecret bool
		wantSecret   string
	}{
		{name: "redacted by default", wantSecret: "[REDACTED]"},
		{name: "revealed on explicit opt-in", revealSecret: true, wantSecret: clientSecret},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"name":"test-service-account","client_id":"test-client-id","client_secret":"blx_live-client-secret"}`))
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

			result, err := handler.CreateServiceAccount(context.Background(), accountName, test.revealSecret)
			if err != nil {
				t.Fatalf("create service account failed: %v", err)
			}

			var decoded struct {
				ServiceAccount struct {
					ClientID     string `json:"client_id"`
					ClientSecret string `json:"client_secret"`
				} `json:"service_account"`
			}
			if err := json.Unmarshal(result, &decoded); err != nil {
				t.Fatalf("failed to decode handler response: %v", err)
			}
			if decoded.ServiceAccount.ClientID != clientID {
				t.Fatalf("client ID = %q, want %q", decoded.ServiceAccount.ClientID, clientID)
			}
			if decoded.ServiceAccount.ClientSecret != test.wantSecret {
				t.Fatalf("client secret = %q, want %q", decoded.ServiceAccount.ClientSecret, test.wantSecret)
			}
			if !test.revealSecret && strings.Contains(string(result), clientSecret) {
				t.Fatal("default response contains the plaintext client secret")
			}
		})
	}
}
