package serviceaccounts

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/client"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/blaxel-ai/toolkit/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all service account-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		// Log error but continue - tools will return errors when called
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// List service accounts tool
	listServiceAccountsSchema := json.RawMessage(`{
		"type": "object",
		"properties": {}
	}`)

	s.AddTool(
		mcp.NewToolWithRawSchema("list_service_accounts", "List all service accounts in the workspace", listServiceAccountsSchema),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, ok := request.Params.Arguments.(map[string]interface{})
			if !ok {
				args = make(map[string]interface{})
			}

			result, err := listServiceAccountsHandler(ctx, sdkClient, args)
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
		// Create service account tool
		createServiceAccountSchema := json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {
					"type": "string",
					"description": "Display name for the service account",
					"minLength": 1
				}
			},
			"required": ["name"]
		}`)

		s.AddTool(
			mcp.NewToolWithRawSchema("create_service_account", "Create a new service account", createServiceAccountSchema),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args, ok := request.Params.Arguments.(map[string]interface{})
				if !ok {
					args = make(map[string]interface{})
				}

				result, err := createServiceAccountHandler(ctx, sdkClient, args)
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

		// Delete service account tool
		deleteServiceAccountSchema := json.RawMessage(`{
			"type": "object",
			"properties": {
				"clientId": {
					"type": "string",
					"description": "Client ID of the service account to delete",
					"minLength": 1
				}
			},
			"required": ["clientId"]
		}`)

		s.AddTool(
			mcp.NewToolWithRawSchema("delete_service_account", "Delete a service account by client_id", deleteServiceAccountSchema),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args, ok := request.Params.Arguments.(map[string]interface{})
				if !ok {
					args = make(map[string]interface{})
				}

				result, err := deleteServiceAccountHandler(ctx, sdkClient, args)
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

func listServiceAccountsHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	serviceAccounts, err := sdkClient.GetWorkspaceServiceAccountsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list service accounts: %w", err)
	}

	if serviceAccounts.JSON200 == nil {
		return nil, fmt.Errorf("no service accounts found")
	}

	var accountList []map[string]interface{}
	for _, account := range *serviceAccounts.JSON200 {
		accountInfo := map[string]interface{}{
			"name":      "",
			"client_id": "",
		}

		if account.Name != nil {
			accountInfo["name"] = *account.Name
		}
		if account.ClientId != nil {
			accountInfo["client_id"] = *account.ClientId
		}

		accountList = append(accountList, accountInfo)
	}

	return map[string]interface{}{
		"service_accounts": accountList,
		"count":            len(accountList),
	}, nil
}

func createServiceAccountHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}

	serviceAccountData := sdk.CreateWorkspaceServiceAccountJSONRequestBody{
		Name: name,
	}

	account, err := sdkClient.CreateWorkspaceServiceAccountWithResponse(ctx, serviceAccountData)
	if err != nil {
		return nil, fmt.Errorf("failed to create service account: %w", err)
	}

	if account.JSON200 == nil {
		return nil, fmt.Errorf("no service account created")
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Service account '%s' created successfully", name),
		"service_account": map[string]interface{}{
			"name":      name,
			"client_id": "",
		},
	}

	if account.JSON200 != nil && account.JSON200.ClientId != nil {
		result["service_account"].(map[string]interface{})["client_id"] = *account.JSON200.ClientId
		if account.JSON200.ClientSecret != nil {
			result["service_account"].(map[string]interface{})["client_secret"] = *account.JSON200.ClientSecret
			result["message"] = fmt.Sprintf("Service account '%s' created successfully. Save the client_secret as it won't be shown again.", name)
		}
	}

	return result, nil
}

func deleteServiceAccountHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	clientID, ok := params["clientId"].(string)
	if !ok || clientID == "" {
		return nil, fmt.Errorf("clientId is required")
	}

	_, err := sdkClient.DeleteWorkspaceServiceAccountWithResponse(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete service account: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Service account with client_id '%s' deleted successfully", clientID),
	}, nil
}
