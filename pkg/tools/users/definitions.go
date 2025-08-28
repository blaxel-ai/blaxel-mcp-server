package users

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// UserHandler defines the interface for user operations
type UserHandler interface {
	ListUsers(ctx context.Context, filter string) ([]byte, error)
	GetUser(ctx context.Context, email string) ([]byte, error)
	InviteUser(ctx context.Context, email, role string) ([]byte, error)
	UpdateUserRole(ctx context.Context, email, role string) ([]byte, error)
	RemoveUser(ctx context.Context, email string) ([]byte, error)
}

// UserHandlerWithReadOnly extends UserHandler with readonly capability
type UserHandlerWithReadOnly interface {
	UserHandler
	IsReadOnly() bool
}

// RegisterUserTools registers user tools with the given handler
func RegisterUserTools(s *server.MCPServer, handler UserHandler) {
	// Check if handler supports readonly mode
	readOnlyHandler, hasReadOnly := handler.(UserHandlerWithReadOnly)
	isReadOnly := hasReadOnly && readOnlyHandler.IsReadOnly()

	// List workspace users tool
	listUsersTool := mcp.NewTool("list_workspace_users",
		mcp.WithDescription("List all users in the workspace"),
		mcp.WithString("filter",
			mcp.Description("Optional filter to match user names or emails"),
		),
	)

	s.AddTool(listUsersTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filter := request.GetString("filter", "")

		result, err := handler.ListUsers(ctx, filter)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Get user tool
	getUserTool := mcp.NewTool("get_workspace_user",
		mcp.WithDescription("Get details of a specific user in the workspace"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Email of the user to retrieve"),
		),
	)

	s.AddTool(getUserTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		email := request.GetString("name", "")
		if email == "" {
			return mcp.NewToolResultError("name is required"), nil
		}

		result, err := handler.GetUser(ctx, email)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Only register write operations if not in read-only mode
	if !isReadOnly {
		// Invite user tool
		inviteUserTool := mcp.NewTool("invite_workspace_user",
			mcp.WithDescription("Invite a user to the workspace"),
			mcp.WithString("email",
				mcp.Required(),
				mcp.Description("Email of the user to invite"),
			),
			mcp.WithString("role",
				mcp.Description("Role to assign to the user (optional)"),
			),
		)

		s.AddTool(inviteUserTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			email := request.GetString("email", "")
			if email == "" {
				return mcp.NewToolResultError("email is required"), nil
			}

			role := request.GetString("role", "")

			result, err := handler.InviteUser(ctx, email, role)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})

		// Update user role tool
		updateUserRoleTool := mcp.NewTool("update_workspace_user_role",
			mcp.WithDescription("Update a user's role in the workspace"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Email of the user to update"),
			),
			mcp.WithString("role",
				mcp.Required(),
				mcp.Description("New role for the user"),
			),
		)

		s.AddTool(updateUserRoleTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			email := request.GetString("name", "")
			if email == "" {
				return mcp.NewToolResultError("name is required"), nil
			}

			role := request.GetString("role", "")
			if role == "" {
				return mcp.NewToolResultError("role is required"), nil
			}

			result, err := handler.UpdateUserRole(ctx, email, role)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})

		// Remove user tool
		removeUserTool := mcp.NewTool("remove_workspace_user",
			mcp.WithDescription("Remove a user from the workspace"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Email of the user to remove"),
			),
		)

		s.AddTool(removeUserTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			email := request.GetString("name", "")
			if email == "" {
				return mcp.NewToolResultError("name is required"), nil
			}

			result, err := handler.RemoveUser(ctx, email)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})
	}
}
