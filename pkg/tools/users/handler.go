package users

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/client"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/toolkit/sdk"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// SDKHandler implements UserHandler using the SDK client
type SDKHandler struct {
	sdkClient *sdk.ClientWithResponses
	readOnly  bool
}

// NewSDKHandler creates a new SDK-based user handler
func NewSDKHandler(cfg *config.Config) (UserHandler, error) {
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SDK client: %w", err)
	}

	return &SDKHandler{
		sdkClient: sdkClient,
		readOnly:  cfg.ReadOnly,
	}, nil
}

// ListUsers implements UserHandler.ListUsers
func (h *SDKHandler) ListUsers(ctx context.Context, filter string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	users, err := h.sdkClient.ListWorkspaceUsersWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace users: %w", err)
	}

	if users.JSON200 == nil {
		result := map[string]interface{}{
			"users": []interface{}{},
			"count": 0,
		}
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to format response: %w", err)
		}
		return jsonData, nil
	}

	var userList []map[string]interface{}
	for _, user := range *users.JSON200 {
		userInfo := make(map[string]interface{})

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
			userInfo["name"] = name
		}
		if user.Role != nil {
			userInfo["role"] = *user.Role
		}
		if user.Accepted != nil {
			userInfo["accepted"] = *user.Accepted
		}
		if user.EmailVerified != nil {
			userInfo["email_verified"] = *user.EmailVerified
		}

		// Apply filter if provided
		if filter != "" {
			filterLower := strings.ToLower(filter)
			email := ""
			if user.Email != nil {
				email = *user.Email
			}
			name := ""
			if user.GivenName != nil || user.FamilyName != nil {
				if user.GivenName != nil {
					name = *user.GivenName
				}
				if user.FamilyName != nil {
					if name != "" {
						name += " "
					}
					name += *user.FamilyName
				}
			}
			emailMatch := strings.Contains(strings.ToLower(email), filterLower)
			nameMatch := strings.Contains(strings.ToLower(name), filterLower)
			if !emailMatch && !nameMatch {
				continue
			}
		}

		userList = append(userList, userInfo)
	}

	result := map[string]interface{}{
		"users": userList,
		"count": len(userList),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return jsonData, nil
}

// GetUser implements UserHandler.GetUser
func (h *SDKHandler) GetUser(ctx context.Context, email string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// List all users and find the one with matching email
	users, err := h.sdkClient.ListWorkspaceUsersWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace users: %w", err)
	}

	if users.JSON200 == nil {
		return nil, fmt.Errorf("no users found")
	}

	for _, user := range *users.JSON200 {
		if user.Email != nil && strings.EqualFold(*user.Email, email) {
			userInfo := make(map[string]interface{})

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
				userInfo["name"] = name
			}
			if user.Role != nil {
				userInfo["role"] = *user.Role
			}
			if user.Accepted != nil {
				userInfo["accepted"] = *user.Accepted
			}
			if user.EmailVerified != nil {
				userInfo["email_verified"] = *user.EmailVerified
			}

			result := map[string]interface{}{
				"user": userInfo,
			}

			jsonData, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to format response: %w", err)
			}

			return jsonData, nil
		}
	}

	return nil, fmt.Errorf("user with email '%s' not found in workspace", email)
}

// InviteUser implements UserHandler.InviteUser
func (h *SDKHandler) InviteUser(ctx context.Context, email, role string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	emailType := openapi_types.Email(email)
	inviteData := sdk.InviteWorkspaceUserJSONRequestBody{
		Email: &emailType,
	}

	resp, err := h.sdkClient.InviteWorkspaceUserWithResponse(ctx, inviteData)
	if err != nil {
		return nil, fmt.Errorf("failed to invite user: %w", err)
	}

	// Check response status
	if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
		message := fmt.Sprintf("Successfully invited user '%s' to the workspace", email)

		// If role was requested but not set on invite, mention it needs to be set separately
		if role != "" {
			message += fmt.Sprintf(". Role '%s' can be set using update_workspace_user_role after the user accepts the invitation", role)
		}

		result := map[string]interface{}{
			"success": true,
			"message": message,
		}

		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to format response: %w", err)
		}

		return jsonData, nil
	}

	if resp.StatusCode() == 409 {
		return nil, fmt.Errorf("user '%s' is already in the workspace or has a pending invitation", email)
	}

	return nil, fmt.Errorf("failed to invite user with status %d", resp.StatusCode())
}

// UpdateUserRole implements UserHandler.UpdateUserRole
func (h *SDKHandler) UpdateUserRole(ctx context.Context, email, role string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	updateData := sdk.UpdateWorkspaceUserRoleJSONRequestBody{
		Role: role,
	}

	// The API expects either sub or email as the identifier
	resp, err := h.sdkClient.UpdateWorkspaceUserRoleWithResponse(ctx, email, updateData)
	if err != nil {
		return nil, fmt.Errorf("failed to update user role: %w", err)
	}

	// Check response status
	if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
		result := map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Successfully updated role for user '%s' to '%s'", email, role),
		}

		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to format response: %w", err)
		}

		return jsonData, nil
	}

	if resp.StatusCode() == 404 {
		return nil, fmt.Errorf("user '%s' not found in workspace", email)
	}

	return nil, fmt.Errorf("failed to update user role with status %d", resp.StatusCode())
}

// RemoveUser implements UserHandler.RemoveUser
func (h *SDKHandler) RemoveUser(ctx context.Context, email string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// The API expects either sub or email as the identifier
	resp, err := h.sdkClient.RemoveWorkspaceUserWithResponse(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to remove user: %w", err)
	}

	// Check response status
	if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
		result := map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Successfully removed user '%s' from the workspace", email),
		}

		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to format response: %w", err)
		}

		return jsonData, nil
	}

	if resp.StatusCode() == 404 {
		return nil, fmt.Errorf("user '%s' not found in workspace", email)
	}

	return nil, fmt.Errorf("failed to remove user with status %d", resp.StatusCode())
}

// IsReadOnly implements UserHandlerWithReadOnly.IsReadOnly
func (h *SDKHandler) IsReadOnly() bool {
	return h.readOnly
}
