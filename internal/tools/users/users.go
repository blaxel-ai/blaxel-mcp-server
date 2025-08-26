package users

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/client"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/blaxel-ai/toolkit/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	openapi_types "github.com/oapi-codegen/runtime/types"
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
	Email         string `json:"email"`
	Sub           string `json:"sub,omitempty"`
	Name          string `json:"name,omitempty"`
	Role          string `json:"role,omitempty"`
	Accepted      bool   `json:"accepted"`
	EmailVerified bool   `json:"email_verified"`
}

type GetUserRequest struct {
	Email string `json:"email"`
}

type GetUserResponse struct {
	User UserInfo `json:"user"`
}

type InviteUserRequest struct {
	Email string `json:"email"`
	Role  string `json:"role,omitempty"` // If role assignment is supported on invite
}

type InviteUserResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type UpdateUserRoleRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type UpdateUserRoleResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type RemoveUserRequest struct {
	Email string `json:"email"`
}

type RemoveUserResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RegisterTools registers all user-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		// Just log the error or handle it gracefully
		fmt.Printf("Failed to initialize SDK client: %v\n", err)
		return
	}

	// List workspace users tool
	listUsersTool := mcp.NewTool("list_workspace_users",
		mcp.WithDescription("List all users in the workspace"),
		mcp.WithString("filter",
			mcp.Description("Optional filter to match user names or emails"),
		),
	)

	s.AddTool(listUsersTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req ListUsersRequest
		if err := request.BindArguments(&req); err != nil {
			// If binding fails, try to get filter directly for backward compatibility
			req.Filter = request.GetString("filter", "")
		}

		result, err := listUsersHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		jsonResult, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(jsonResult)), nil
	})

	// Get workspace user tool
	getUserTool := mcp.NewTool("get_workspace_user",
		mcp.WithDescription("Get details of a workspace user by email"),
		mcp.WithString("email",
			mcp.Required(),
			mcp.Description("Email address of the user to retrieve"),
		),
	)

	s.AddTool(getUserTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req GetUserRequest
		if err := request.BindArguments(&req); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		if req.Email == "" {
			return mcp.NewToolResultError("user email is required"), nil
		}

		result, err := getUserHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		jsonResult, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(jsonResult)), nil
	})

	// Only register write operations if not in read-only mode
	if !cfg.ReadOnly {
		// Invite user to workspace tool
		inviteUserTool := mcp.NewTool("invite_workspace_user",
			mcp.WithDescription("Invite a user to join the workspace"),
			mcp.WithString("email",
				mcp.Required(),
				mcp.Description("Email address of the user to invite"),
			),
		)

		s.AddTool(inviteUserTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req InviteUserRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Email == "" {
				return mcp.NewToolResultError("user email is required"), nil
			}

			result, err := inviteUserHandler(ctx, sdkClient, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(jsonResult)), nil
		})

		// Update user role tool
		updateUserRoleTool := mcp.NewTool("update_workspace_user_role",
			mcp.WithDescription("Update a workspace user's role"),
			mcp.WithString("email",
				mcp.Required(),
				mcp.Description("Email address of the user"),
			),
			mcp.WithString("role",
				mcp.Required(),
				mcp.Description("New role for the user (e.g., admin, member, viewer)"),
			),
		)

		s.AddTool(updateUserRoleTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req UpdateUserRoleRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Email == "" {
				return mcp.NewToolResultError("user email is required"), nil
			}
			if req.Role == "" {
				return mcp.NewToolResultError("role is required"), nil
			}

			result, err := updateUserRoleHandler(ctx, sdkClient, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(jsonResult)), nil
		})

		// Remove user from workspace tool
		removeUserTool := mcp.NewTool("remove_workspace_user",
			mcp.WithDescription("Remove a user from the workspace"),
			mcp.WithString("email",
				mcp.Required(),
				mcp.Description("Email address of the user to remove"),
			),
		)

		s.AddTool(removeUserTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req RemoveUserRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Email == "" {
				return mcp.NewToolResultError("user email is required"), nil
			}

			result, err := removeUserHandler(ctx, sdkClient, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(jsonResult)), nil
		})
	}
}

func listUsersHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req ListUsersRequest) (*ListUsersResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	users, err := sdkClient.ListWorkspaceUsersWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace users: %w", err)
	}

	if users.JSON200 == nil {
		return &ListUsersResponse{
			Users: []UserInfo{},
			Count: 0,
		}, nil
	}

	var userList []UserInfo
	for _, user := range *users.JSON200 {
		userInfo := UserInfo{}

		if user.Email != nil {
			userInfo.Email = *user.Email
		}
		if user.Sub != nil {
			userInfo.Sub = *user.Sub
		}
		if user.GivenName != nil || user.FamilyName != nil {
			name := ""
			if user.GivenName != nil {
				name = *user.GivenName
			}
			if user.FamilyName != nil {
				if name != "" {
					name += " "
				}
				name += *user.FamilyName
			}
			userInfo.Name = name
		}
		if user.Role != nil {
			userInfo.Role = *user.Role
		}
		if user.Accepted != nil {
			userInfo.Accepted = *user.Accepted
		}
		if user.EmailVerified != nil {
			userInfo.EmailVerified = *user.EmailVerified
		}

		// Apply filter if provided
		if req.Filter != "" {
			filterLower := strings.ToLower(req.Filter)
			emailMatch := strings.Contains(strings.ToLower(userInfo.Email), filterLower)
			nameMatch := strings.Contains(strings.ToLower(userInfo.Name), filterLower)
			if !emailMatch && !nameMatch {
				continue
			}
		}

		userList = append(userList, userInfo)
	}

	return &ListUsersResponse{
		Users: userList,
		Count: len(userList),
	}, nil
}

func getUserHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req GetUserRequest) (*GetUserResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// List all users and find the one with matching email
	users, err := sdkClient.ListWorkspaceUsersWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace users: %w", err)
	}

	if users.JSON200 == nil {
		return nil, fmt.Errorf("no users found")
	}

	for _, user := range *users.JSON200 {
		if user.Email != nil && strings.EqualFold(*user.Email, req.Email) {
			userInfo := UserInfo{}

			if user.Email != nil {
				userInfo.Email = *user.Email
			}
			if user.Sub != nil {
				userInfo.Sub = *user.Sub
			}
			if user.GivenName != nil || user.FamilyName != nil {
				name := ""
				if user.GivenName != nil {
					name = *user.GivenName
				}
				if user.FamilyName != nil {
					if name != "" {
						name += " "
					}
					name += *user.FamilyName
				}
				userInfo.Name = name
			}
			if user.Role != nil {
				userInfo.Role = *user.Role
			}
			if user.Accepted != nil {
				userInfo.Accepted = *user.Accepted
			}
			if user.EmailVerified != nil {
				userInfo.EmailVerified = *user.EmailVerified
			}

			return &GetUserResponse{
				User: userInfo,
			}, nil
		}
	}

	return nil, fmt.Errorf("user with email '%s' not found in workspace", req.Email)
}

func inviteUserHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req InviteUserRequest) (*InviteUserResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	email := openapi_types.Email(req.Email)
	inviteData := sdk.InviteWorkspaceUserJSONRequestBody{
		Email: &email,
	}

	resp, err := sdkClient.InviteWorkspaceUserWithResponse(ctx, inviteData)
	if err != nil {
		return nil, fmt.Errorf("failed to invite user: %w", err)
	}

	// Check response status
	if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
		message := fmt.Sprintf("Successfully invited user '%s' to the workspace", req.Email)

		// If role was requested but not set on invite, mention it needs to be set separately
		if req.Role != "" {
			message += fmt.Sprintf(". Role '%s' can be set using update_workspace_user_role after the user accepts the invitation", req.Role)
		}

		return &InviteUserResponse{
			Success: true,
			Message: message,
		}, nil
	}

	if resp.StatusCode() == 409 {
		return nil, fmt.Errorf("user '%s' is already in the workspace or has a pending invitation", req.Email)
	}

	return nil, fmt.Errorf("failed to invite user with status %d", resp.StatusCode())
}

func updateUserRoleHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req UpdateUserRoleRequest) (*UpdateUserRoleResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	updateData := sdk.UpdateWorkspaceUserRoleJSONRequestBody{
		Role: req.Role,
	}

	// The API expects either sub or email as the identifier
	resp, err := sdkClient.UpdateWorkspaceUserRoleWithResponse(ctx, req.Email, updateData)
	if err != nil {
		return nil, fmt.Errorf("failed to update user role: %w", err)
	}

	// Check response status
	if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
		return &UpdateUserRoleResponse{
			Success: true,
			Message: fmt.Sprintf("Successfully updated role for user '%s' to '%s'", req.Email, req.Role),
		}, nil
	}

	if resp.StatusCode() == 404 {
		return nil, fmt.Errorf("user '%s' not found in workspace", req.Email)
	}

	return nil, fmt.Errorf("failed to update user role with status %d", resp.StatusCode())
}

func removeUserHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req RemoveUserRequest) (*RemoveUserResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// The API expects either sub or email as the identifier
	resp, err := sdkClient.RemoveWorkspaceUserWithResponse(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to remove user: %w", err)
	}

	// Check response status
	if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
		return &RemoveUserResponse{
			Success: true,
			Message: fmt.Sprintf("Successfully removed user '%s' from the workspace", req.Email),
		}, nil
	}

	if resp.StatusCode() == 404 {
		return nil, fmt.Errorf("user '%s' not found in workspace", req.Email)
	}

	return nil, fmt.Errorf("failed to remove user with status %d", resp.StatusCode())
}
