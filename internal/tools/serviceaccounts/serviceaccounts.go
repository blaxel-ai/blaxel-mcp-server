package serviceaccounts

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/client"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/tools"
	"github.com/blaxel-ai/toolkit/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Request/Response types for type safety
type ListServiceAccountsRequest struct {
	Filter string `json:"filter,omitempty"`
}

type ListServiceAccountsResponse json.RawMessage

type GetServiceAccountRequest struct {
	ClientID string `json:"clientId"`
}

type GetServiceAccountResponse json.RawMessage

type CreateServiceAccountRequest struct {
	Name string `json:"name"`
}

type CreateServiceAccountResponse struct {
	Success        bool                   `json:"success"`
	Message        string                 `json:"message"`
	ServiceAccount map[string]interface{} `json:"service_account"`
}

type DeleteServiceAccountRequest struct {
	ClientID string `json:"clientId"`
}

type DeleteServiceAccountResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type UpdateServiceAccountRequest struct {
	ClientID string `json:"clientId"`
	Name     string `json:"name"`
}

type UpdateServiceAccountResponse struct {
	Success        bool                   `json:"success"`
	Message        string                 `json:"message"`
	ServiceAccount map[string]interface{} `json:"service_account"`
}

// RegisterTools registers all service account-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		// Just log the error or handle it gracefully
		fmt.Printf("Failed to initialize SDK client: %v\n", err)
		return
	}

	// List service accounts tool
	listServiceAccountsTool := mcp.NewTool("list_service_accounts",
		mcp.WithDescription("List all service accounts in the workspace"),
		mcp.WithString("filter",
			mcp.Description("Optional filter to match service account names"),
		),
	)

	s.AddTool(listServiceAccountsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req ListServiceAccountsRequest
		if err := request.BindArguments(&req); err != nil {
			// If binding fails, try to get filter directly for backward compatibility
			req.Filter = request.GetString("filter", "")
		}

		result, err := listServiceAccountsHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(*result)), nil
	})

	// Get service account tool
	getServiceAccountTool := mcp.NewTool("get_service_account",
		mcp.WithDescription("Get details of a service account by client ID"),
		mcp.WithString("clientId",
			mcp.Required(),
			mcp.Description("Client ID of the service account to retrieve"),
		),
	)

	s.AddTool(getServiceAccountTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req GetServiceAccountRequest
		if err := request.BindArguments(&req); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		if req.ClientID == "" {
			return mcp.NewToolResultError("client ID is required"), nil
		}

		result, err := getServiceAccountHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(*result)), nil
	})

	// Only register write operations if not in read-only mode
	if !cfg.ReadOnly {
		// Create service account tool
		createServiceAccountTool := mcp.NewTool("create_service_account",
			mcp.WithDescription("Create a new service account"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Display name for the service account"),
			),
		)

		s.AddTool(createServiceAccountTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req CreateServiceAccountRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Name == "" {
				return mcp.NewToolResultError("service account name is required"), nil
			}

			result, err := createServiceAccountHandler(ctx, sdkClient, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(jsonResult)), nil
		})

		// Delete service account tool
		deleteServiceAccountTool := mcp.NewTool("delete_service_account",
			mcp.WithDescription("Delete a service account by client ID"),
			mcp.WithString("clientId",
				mcp.Required(),
				mcp.Description("Client ID of the service account to delete"),
			),
		)

		s.AddTool(deleteServiceAccountTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req DeleteServiceAccountRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.ClientID == "" {
				return mcp.NewToolResultError("client ID is required"), nil
			}

			result, err := deleteServiceAccountHandler(ctx, sdkClient, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(jsonResult)), nil
		})

		// Update service account tool
		updateServiceAccountTool := mcp.NewTool("update_service_account",
			mcp.WithDescription("Update a service account's name"),
			mcp.WithString("clientId",
				mcp.Required(),
				mcp.Description("Client ID of the service account to update"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("New name for the service account"),
			),
		)

		s.AddTool(updateServiceAccountTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req UpdateServiceAccountRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.ClientID == "" {
				return mcp.NewToolResultError("clientId is required"), nil
			}
			if req.Name == "" {
				return mcp.NewToolResultError("name is required"), nil
			}

			result, err := updateServiceAccountHandler(ctx, sdkClient, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(jsonResult)), nil
		})
	}
}

func listServiceAccountsHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req ListServiceAccountsRequest) (*ListServiceAccountsResponse, error) {
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

	// Service accounts use a different type structure, fall back to manual filtering
	var result interface{} = serviceAccounts.JSON200
	if req.Filter != "" {
		var filtered []interface{}
		for _, account := range *serviceAccounts.JSON200 {
			name := ""
			if account.Name != nil {
				name = *account.Name
			}
			if name != "" && tools.ContainsString(name, req.Filter) {
				filtered = append(filtered, account)
			}
		}
		result = filtered
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")

	response := ListServiceAccountsResponse(jsonData)
	return &response, nil
}

func getServiceAccountHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req GetServiceAccountRequest) (*GetServiceAccountResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// List all service accounts and find the one with matching client ID
	serviceAccounts, err := sdkClient.GetWorkspaceServiceAccountsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get service account: %w", err)
	}

	if serviceAccounts.JSON200 == nil {
		return nil, fmt.Errorf("no service accounts found")
	}

	for _, account := range *serviceAccounts.JSON200 {
		if account.ClientId != nil && *account.ClientId == req.ClientID {
			jsonData, _ := json.MarshalIndent(account, "", "  ")
			response := GetServiceAccountResponse(jsonData)
			return &response, nil
		}
	}

	return nil, fmt.Errorf("service account with client ID '%s' not found", req.ClientID)
}

func createServiceAccountHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req CreateServiceAccountRequest) (*CreateServiceAccountResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	serviceAccountData := sdk.CreateWorkspaceServiceAccountJSONRequestBody{
		Name: req.Name,
	}

	account, err := sdkClient.CreateWorkspaceServiceAccountWithResponse(ctx, serviceAccountData)
	if err != nil {
		return nil, fmt.Errorf("failed to create service account: %w", err)
	}

	if account.JSON200 == nil {
		return nil, fmt.Errorf("no service account created")
	}

	result := &CreateServiceAccountResponse{
		Success: true,
		Message: fmt.Sprintf("Service account '%s' created successfully", req.Name),
		ServiceAccount: map[string]interface{}{
			"name":      req.Name,
			"client_id": "",
		},
	}

	if account.JSON200 != nil && account.JSON200.ClientId != nil {
		result.ServiceAccount["client_id"] = *account.JSON200.ClientId
		if account.JSON200.ClientSecret != nil {
			result.ServiceAccount["client_secret"] = *account.JSON200.ClientSecret
			result.Message = fmt.Sprintf("Service account '%s' created successfully. Save the client_secret as it won't be shown again.", req.Name)
		}
	}

	return result, nil
}

func deleteServiceAccountHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req DeleteServiceAccountRequest) (*DeleteServiceAccountResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	_, err := sdkClient.DeleteWorkspaceServiceAccountWithResponse(ctx, req.ClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete service account: %w", err)
	}

	return &DeleteServiceAccountResponse{
		Success: true,
		Message: fmt.Sprintf("Service account with client ID '%s' deleted successfully", req.ClientID),
	}, nil
}

func updateServiceAccountHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req UpdateServiceAccountRequest) (*UpdateServiceAccountResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Build update request
	updateData := sdk.UpdateWorkspaceServiceAccountJSONRequestBody{
		Name: &req.Name,
	}

	// Update the service account
	resp, err := sdkClient.UpdateWorkspaceServiceAccountWithResponse(ctx, req.ClientID, updateData)
	if err != nil {
		return nil, fmt.Errorf("failed to update service account: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to update service account with status %d", resp.StatusCode())
	}

	// Get updated details
	serviceAccounts, err := sdkClient.GetWorkspaceServiceAccountsWithResponse(ctx)
	if err != nil {
		return &UpdateServiceAccountResponse{
			Success: true,
			Message: fmt.Sprintf("Service account '%s' updated successfully", req.ClientID),
			ServiceAccount: map[string]interface{}{
				"clientId": req.ClientID,
				"name":     req.Name,
			},
		}, nil
	}

	// Find the updated account
	for _, account := range *serviceAccounts.JSON200 {
		if account.ClientId != nil && *account.ClientId == req.ClientID {
			return &UpdateServiceAccountResponse{
				Success: true,
				Message: fmt.Sprintf("Service account '%s' updated successfully", req.ClientID),
				ServiceAccount: map[string]interface{}{
					"clientId": req.ClientID,
					"name":     req.Name,
				},
			}, nil
		}
	}

	return &UpdateServiceAccountResponse{
		Success: true,
		Message: fmt.Sprintf("Service account '%s' updated successfully", req.ClientID),
		ServiceAccount: map[string]interface{}{
			"clientId": req.ClientID,
			"name":     req.Name,
		},
	}, nil
}
