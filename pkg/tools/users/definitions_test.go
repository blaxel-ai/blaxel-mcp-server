package users

import (
	"context"
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type inviteValidationHandler struct{ inviteCalls int }

func (*inviteValidationHandler) ListUsers(context.Context, string) ([]byte, error) { return nil, nil }
func (*inviteValidationHandler) GetUser(context.Context, string) ([]byte, error)   { return nil, nil }
func (h *inviteValidationHandler) InviteUser(context.Context, string, string) ([]byte, error) {
	h.inviteCalls++
	return []byte(`{"success":true}`), nil
}
func (*inviteValidationHandler) UpdateUserRole(context.Context, string, string) ([]byte, error) {
	return nil, nil
}
func (*inviteValidationHandler) RemoveUser(context.Context, string) ([]byte, error) { return nil, nil }

func TestInviteWorkspaceUserRejectsMalformedEmailBeforeHandler(t *testing.T) {
	handler := &inviteValidationHandler{}
	mcpServer := server.NewMCPServer("users-test", "1")
	RegisterUserTools(mcpServer, handler, &config.Config{})
	tool := mcpServer.GetTool("invite_workspace_user")
	if tool == nil {
		t.Fatal("invite_workspace_user was not registered")
	}

	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: map[string]any{"email": "not-an-email", "role": "member"}},
	})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if !result.IsError || len(result.Content) == 0 {
		t.Fatalf("malformed email result = %#v, want tool error", result)
	}
	text, _ := result.Content[0].(mcp.TextContent)
	if !strings.Contains(strings.ToLower(text.Text), "valid") {
		t.Fatalf("malformed email error = %q, want actionable validation", text.Text)
	}
	if handler.inviteCalls != 0 {
		t.Fatalf("InviteUser called %d times, want zero", handler.inviteCalls)
	}
}
