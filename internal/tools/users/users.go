package users

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/client"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/blaxel-ai/toolkit/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// RegisterTools registers all user-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		// Log error but continue - tools will return errors when called
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// List users tool
	listUsersSchema := json.RawMessage(`{
		"type": "object",
		"properties": {}
	}`)

	s.AddTool(
		mcp.NewToolWithRawSchema("list_users", "List all users in the workspace", listUsersSchema),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, ok := request.Params.Arguments.(map[string]interface{})
			if !ok {
				args = make(map[string]interface{})
			}

			result, err := listUsersHandler(ctx, sdkClient, args)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent(err.Error()),
					},
					IsError: true,
				}, nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(string(jsonResult)),
				},
			}, nil
		},
	)

	// Only register write operations if not in read-only mode
	if !cfg.ReadOnly {
		// Invite user tool
		inviteUserSchema := json.RawMessage(`{
			"type": "object",
			"properties": {
				"email": {
					"type": "string",
					"format": "email",
					"description": "Email address of the user to invite",
					"minLength": 1
				}
			},
			"required": ["email"]
		}`)

		s.AddTool(
			mcp.NewToolWithRawSchema("invite_user", "Invite a user to the workspace", inviteUserSchema),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args, ok := request.Params.Arguments.(map[string]interface{})
				if !ok {
					args = make(map[string]interface{})
				}

				result, err := inviteUserHandler(ctx, sdkClient, args)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.NewTextContent(err.Error()),
						},
						IsError: true,
					}, nil
				}

				jsonResult, _ := json.MarshalIndent(result, "", "  ")
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent(string(jsonResult)),
					},
				}, nil
			},
		)

		// Delete user tool
		deleteUserSchema := json.RawMessage(`{
			"type": "object",
			"properties": {
				"subOrEmail": {
					"type": "string",
					"description": "The user's subject (sub) or email to remove",
					"minLength": 1
				}
			},
			"required": ["subOrEmail"]
		}`)

		s.AddTool(
			mcp.NewToolWithRawSchema("delete_user", "Remove a user from the workspace (or revoke invitation)", deleteUserSchema),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args, ok := request.Params.Arguments.(map[string]interface{})
				if !ok {
					args = make(map[string]interface{})
				}

				result, err := deleteUserHandler(ctx, sdkClient, args)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.NewTextContent(err.Error()),
						},
						IsError: true,
					}, nil
				}

				jsonResult, _ := json.MarshalIndent(result, "", "  ")
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent(string(jsonResult)),
					},
				}, nil
			},
		)
	}
}

func listUsersHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	users, err := sdkClient.ListWorkspaceUsersWithResponse(ctx)
	if users.JSON200 == nil {
		return nil, fmt.Errorf("no users found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	var userList []map[string]interface{}
	for _, user := range *users.JSON200 {
		userInfo := map[string]interface{}{
			"email": "",
			"sub":   "",
		}

		if user.Email != nil {
			userInfo["email"] = *user.Email
		}
		if user.Sub != nil {
			userInfo["sub"] = *user.Sub
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
			if name != "" {
				userInfo["name"] = name
			}
		}

		userList = append(userList, userInfo)
	}

	return map[string]interface{}{
		"users": userList,
		"count": len(userList),
	}, nil
}

func inviteUserHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	email, ok := params["email"].(string)
	if !ok || email == "" {
		return nil, fmt.Errorf("email is required")
	}

	emailSent := openapi_types.Email(email)
	_, err := sdkClient.InviteWorkspaceUserWithResponse(ctx, sdk.InviteWorkspaceUserJSONRequestBody{
		Email: &emailSent,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to invite user: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("User '%s' invited successfully", email),
		"user": map[string]interface{}{
			"email": email,
		},
	}, nil
}

func deleteUserHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	subOrEmail, ok := params["subOrEmail"].(string)
	if !ok || subOrEmail == "" {
		return nil, fmt.Errorf("subOrEmail is required")
	}

	_, err := sdkClient.RemoveWorkspaceUserWithResponse(ctx, subOrEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to delete user: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("User '%s' removed successfully", subOrEmail),
	}, nil
}
