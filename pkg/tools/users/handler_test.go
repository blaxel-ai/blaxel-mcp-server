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

// The invite endpoint binds and honors a role, but the generated request type
// carries only email, so sending the typed body silently dropped the requested
// role and every invitee joined as a member.
func TestInviteUserSendsTheRequestedRole(t *testing.T) {
	for _, tc := range []struct {
		name     string
		role     string
		wantRole string
	}{
		{name: "admin", role: "admin", wantRole: "admin"},
		{name: "owner", role: "owner", wantRole: "owner"},
		{name: "defaults to member", role: "", wantRole: "member"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got map[string]any
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method != http.MethodPost || r.URL.Path != "/users" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Fatalf("decode invite request: %v", err)
				}
				_, _ = w.Write([]byte(`{}`))
			}))
			t.Cleanup(api.Close)

			handler := newUsersTestHandler(t, api.URL)
			result, err := handler.InviteUser(context.Background(), "person@example.com", tc.role)
			if err != nil {
				t.Fatalf("InviteUser() error = %v", err)
			}
			if got["role"] != tc.wantRole {
				t.Fatalf("request role = %v, want %q", got["role"], tc.wantRole)
			}
			if got["email"] != "person@example.com" {
				t.Fatalf("request email = %v, want person@example.com", got["email"])
			}
			if !strings.Contains(string(result), tc.wantRole) {
				t.Fatalf("InviteUser() message = %s, want it to name the role %q", result, tc.wantRole)
			}
		})
	}
}

func TestInviteUserRejectsAnUnknownRole(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("an unknown role must be rejected before the request: %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(api.Close)

	handler := newUsersTestHandler(t, api.URL)
	_, err := handler.InviteUser(context.Background(), "person@example.com", "superadmin")
	if err == nil || !strings.Contains(err.Error(), "invite accepts member, admin or owner") {
		t.Fatalf("InviteUser() error = %v, want it to name the accepted roles", err)
	}
}

// A caller cannot grant a role above their own; the endpoint enforces that with
// a 403, which is a different fix from a duplicate invite.
func TestInviteUserExplainsARefusedRole(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(api.Close)

	handler := newUsersTestHandler(t, api.URL)
	_, err := handler.InviteUser(context.Background(), "person@example.com", "owner")
	if err == nil || !strings.Contains(err.Error(), "not privileged enough") {
		t.Fatalf("InviteUser() error = %v, want it to explain the refused role", err)
	}
}
