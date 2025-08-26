package users

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/client"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Request/Response types for type safety
type ListUsersRequest struct {
	Filter string `json:"filter,omitempty"`
}

type ListUsersResponse struct {
	Users []UserInfo `json:"users"`
	Count int        `json:"count"`
}

type UserInfo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	ID    string `json:"id"`
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
	Name     string `json:"name,omitempty"`
}

type CreateUserResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	User    map[string]interface{} `json:"user"`
}

type DeleteUserRequest struct {
	ID string `json:"id"`
}

type DeleteUserResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RegisterTools registers all user-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client (even though we won't use it yet)
	_, err := client.NewSDKClient(cfg)
	if err != nil {
		// Just log the error or handle it gracefully
		fmt.Printf("Failed to initialize SDK client: %v\n", err)
		return
	}

	// List users tool
	listUsersTool := mcp.NewTool("list_users",
		mcp.WithDescription("List all users in the workspace (not yet implemented)"),
		mcp.WithString("filter",
			mcp.Description("Optional filter to match user names or emails"),
		),
	)

	s.AddTool(listUsersTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// For now, return a placeholder since the SDK doesn't support user operations yet
		result := &ListUsersResponse{
			Users: []UserInfo{},
			Count: 0,
		}
		jsonResult, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(jsonResult)), nil
	})

	// Only register write operations if not in read-only mode
	if !cfg.ReadOnly {
		// Create user tool
		createUserTool := mcp.NewTool("create_user",
			mcp.WithDescription("Create a new user (not yet implemented)"),
			mcp.WithString("email",
				mcp.Required(),
				mcp.Description("Email address for the user"),
			),
			mcp.WithString("password",
				mcp.Description("Password for the user"),
			),
			mcp.WithString("name",
				mcp.Description("Display name for the user"),
			),
		)

		s.AddTool(createUserTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultError("user creation is not yet implemented"), nil
		})

		// Delete user tool
		deleteUserTool := mcp.NewTool("delete_user",
			mcp.WithDescription("Delete a user by ID (not yet implemented)"),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description("ID of the user to delete"),
			),
		)

		s.AddTool(deleteUserTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultError("user deletion is not yet implemented"), nil
		})
	}
}
