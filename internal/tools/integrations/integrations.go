package integrations

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
type ListIntegrationsRequest struct {
	Filter string `json:"filter,omitempty"`
}

type ListIntegrationsResponse json.RawMessage

type GetIntegrationRequest struct {
	Name string `json:"name"`
}

type GetIntegrationResponse json.RawMessage

type CreateIntegrationRequest struct {
	Name            string            `json:"name"`
	IntegrationType string            `json:"integrationType"`
	Secret          map[string]string `json:"secret,omitempty"`
	Config          map[string]string `json:"config,omitempty"`
}

type CreateIntegrationResponse struct {
	Success     bool                   `json:"success"`
	Message     string                 `json:"message"`
	Integration map[string]interface{} `json:"integration"`
}

type DeleteIntegrationRequest struct {
	Name string `json:"name"`
}

type DeleteIntegrationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RegisterTools registers all integration-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// List integrations
	listIntegrationsTool := mcp.NewTool("list_integrations",
		mcp.WithDescription("List all integration connections in the workspace"),
		mcp.WithString("filter",
			mcp.Description("Optional filter string"),
		),
	)

	s.AddTool(listIntegrationsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req ListIntegrationsRequest
		if err := request.BindArguments(&req); err != nil {
			// If binding fails, try to get filter directly for backward compatibility
			req.Filter = request.GetString("filter", "")
		}

		result, err := listIntegrationsHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(*result)), nil
	})

	// Get integration
	getIntegrationTool := mcp.NewTool("get_integration",
		mcp.WithDescription("Get details of a specific integration connection"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the integration"),
		),
	)

	s.AddTool(getIntegrationTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req GetIntegrationRequest
		if err := request.BindArguments(&req); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		if req.Name == "" {
			return mcp.NewToolResultError("integration name is required"), nil
		}

		result, err := getIntegrationHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(*result)), nil
	})

	if !cfg.ReadOnly {
		// Create integration
		createIntegrationTool := mcp.NewTool("create_integration",
			mcp.WithDescription("Create a new integration connection"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name for the integration connection"),
			),
			mcp.WithString("integrationType",
				mcp.Required(),
				mcp.Description("Type of integration (e.g., github, slack, etc.)"),
			),
			mcp.WithObject("secret",
				mcp.Description("Secret credentials for the integration"),
			),
			mcp.WithObject("config",
				mcp.Description("Configuration parameters for the integration"),
			),
		)

		s.AddTool(createIntegrationTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req CreateIntegrationRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Name == "" {
				return mcp.NewToolResultError("integration name is required"), nil
			}
			if req.IntegrationType == "" {
				return mcp.NewToolResultError("integrationType is required"), nil
			}

			result, err := createIntegrationHandler(ctx, sdkClient, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(jsonResult)), nil
		})

		// Delete integration
		deleteIntegrationTool := mcp.NewTool("delete_integration",
			mcp.WithDescription("Delete an integration connection by name"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the integration to delete"),
			),
		)

		s.AddTool(deleteIntegrationTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req DeleteIntegrationRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Name == "" {
				return mcp.NewToolResultError("integration name is required"), nil
			}

			result, err := deleteIntegrationHandler(ctx, sdkClient, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(jsonResult)), nil
		})
	}
}

func listIntegrationsHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req ListIntegrationsRequest) (*ListIntegrationsResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	integrations, err := sdkClient.ListIntegrationConnectionsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list integrations: %w", err)
	}
	if integrations.JSON200 == nil {
		return nil, fmt.Errorf("no integrations found")
	}

	// Use generic filter and marshal function
	jsonData, _ := tools.FilterAndMarshal(integrations.JSON200, req.Filter, func(integration sdk.IntegrationConnection) string {
		if integration.Metadata != nil && integration.Metadata.Name != nil {
			return *integration.Metadata.Name
		}
		return ""
	})

	response := ListIntegrationsResponse(jsonData)
	return &response, nil
}

func getIntegrationHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req GetIntegrationRequest) (*GetIntegrationResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	integration, err := sdkClient.GetIntegrationConnectionWithResponse(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get integration: %w", err)
	}
	if integration.JSON200 == nil {
		return nil, fmt.Errorf("no integration found")
	}

	jsonData, _ := json.MarshalIndent(*integration.JSON200, "", "  ")
	response := GetIntegrationResponse(jsonData)
	return &response, nil
}

func createIntegrationHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req CreateIntegrationRequest) (*CreateIntegrationResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Build integration request
	integrationData := sdk.CreateIntegrationConnectionJSONRequestBody{
		Metadata: &sdk.Metadata{
			Name: &req.Name,
		},
		Spec: &sdk.IntegrationConnectionSpec{
			Integration: &req.IntegrationType,
		},
	}

	// Add secret if provided
	if len(req.Secret) > 0 {
		integrationData.Spec.Secret = &req.Secret
	}

	// Add config if provided
	if len(req.Config) > 0 {
		integrationData.Spec.Config = &req.Config
	}

	integration, err := sdkClient.CreateIntegrationConnectionWithResponse(ctx, integrationData)
	if err != nil {
		return nil, fmt.Errorf("failed to create integration: %w", err)
	}

	if integration.JSON200 == nil {
		if integration.StatusCode() == 409 {
			return nil, fmt.Errorf("integration with name '%s' already exists", req.Name)
		}
		return nil, fmt.Errorf("failed to create integration with status %d", integration.StatusCode())
	}

	return &CreateIntegrationResponse{
		Success: true,
		Message: fmt.Sprintf("Integration '%s' created successfully", req.Name),
		Integration: map[string]interface{}{
			"name": req.Name,
			"type": req.IntegrationType,
		},
	}, nil
}

func deleteIntegrationHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req DeleteIntegrationRequest) (*DeleteIntegrationResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	_, err := sdkClient.DeleteIntegrationConnectionWithResponse(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete integration: %w", err)
	}

	return &DeleteIntegrationResponse{
		Success: true,
		Message: fmt.Sprintf("Integration '%s' deleted successfully", req.Name),
	}, nil
}
