package users

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

func TestUpdateUserRoleUsesAcceptedUserSub(t *testing.T) {
	const (
		email = "accepted@example.com"
		sub   = "auth0|accepted-user"
	)

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/users":
			_, _ = w.Write([]byte(`[{"email":"accepted@example.com","sub":"auth0|accepted-user","accepted":true}]`))
		case r.Method == http.MethodPut && r.URL.Path == "/users/"+sub:
			var body struct {
				Role string `json:"role"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update request: %v", err)
			}
			if body.Role != "admin" {
				t.Fatalf("role = %q, want admin", body.Role)
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(api.Close)

	handler := newUsersTestHandler(t, api.URL)
	if _, err := handler.UpdateUserRole(context.Background(), email, "admin"); err != nil {
		t.Fatalf("update accepted user role: %v", err)
	}
}

func TestUpdateUserRoleUsesEmailForPendingUserWithProvisionalSub(t *testing.T) {
	const (
		email     = "pending+alias@example.com"
		requested = "PENDING+ALIAS@example.com"
	)

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/users":
			_, _ = w.Write([]byte(`[{"email":"pending+alias@example.com","accepted":false,"sub":"provisional-sub"}]`))
		case r.Method == http.MethodPut && r.URL.Path == "/users/pending%2Balias@example.com":
			if r.RequestURI != "/users/pending%252Balias@example.com" {
				t.Fatalf("wire request URI = %q, want double-escaped plus for the route's second decode", r.RequestURI)
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(api.Close)

	handler := newUsersTestHandler(t, api.URL)
	if _, err := handler.UpdateUserRole(context.Background(), requested, "member"); err != nil {
		t.Fatalf("update pending user role: %v", err)
	}
}

func TestRemoveUserUsesAcceptedUserSub(t *testing.T) {
	const (
		email = "accepted@example.com"
		sub   = "auth0|accepted-user"
	)

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/users":
			_, _ = w.Write([]byte(`[{"email":"accepted@example.com","sub":"auth0|accepted-user","accepted":true}]`))
		case r.Method == http.MethodDelete && r.URL.Path == "/users/"+sub:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(api.Close)

	handler := newUsersTestHandler(t, api.URL)
	if _, err := handler.RemoveUser(context.Background(), email); err != nil {
		t.Fatalf("remove accepted user: %v", err)
	}
}

func TestRemoveUserUsesEmailForPendingUserWithProvisionalSub(t *testing.T) {
	const email = "pending+alias@example.com"

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/users":
			_, _ = w.Write([]byte(`[{"email":"pending+alias@example.com","accepted":false,"sub":"provisional-sub"}]`))
		case r.Method == http.MethodDelete && r.URL.Path == "/users/pending%2Balias@example.com":
			if r.RequestURI != "/users/pending%252Balias@example.com" {
				t.Fatalf("wire request URI = %q, want double-escaped plus for the route's second decode", r.RequestURI)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(api.Close)

	handler := newUsersTestHandler(t, api.URL)
	if _, err := handler.RemoveUser(context.Background(), email); err != nil {
		t.Fatalf("remove pending user: %v", err)
	}
}

func TestRemoveUserReturnsActionableNotFoundWhenEmailDoesNotMatch(t *testing.T) {
	const email = "missing@example.com"

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/users" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"email":"someone-else@example.com","sub":"auth0|other","accepted":true}]`))
	}))
	t.Cleanup(api.Close)

	handler := newUsersTestHandler(t, api.URL)
	_, err := handler.RemoveUser(context.Background(), email)
	if err == nil || !strings.Contains(err.Error(), "user with email 'missing@example.com' not found in workspace") {
		t.Fatalf("remove missing user error = %v, want actionable not-found error", err)
	}
}

func newUsersTestHandler(t *testing.T, endpoint string) UserHandler {
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
