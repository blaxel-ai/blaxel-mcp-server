package mcpservers

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
type ListMCPServersRequest struct {
	Filter string `json:"filter,omitempty"`
}

type ListMCPServersResponse json.RawMessage

type GetMCPServerRequest struct {
	Name string `json:"name"`
}

type GetMCPServerResponse json.RawMessage

type CreateMCPServerRequest struct {
	Name                      string            `json:"name"`
	IntegrationConnectionName string            `json:"integrationConnectionName,omitempty"`
	IntegrationType           string            `json:"integrationType,omitempty"`
	Secret                    map[string]string `json:"secret,omitempty"`
	Config                    map[string]string `json:"config,omitempty"`
}

type CreateMCPServerResponse struct {
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	MCPServer map[string]interface{} `json:"mcp_server"`
}

type DeleteMCPServerRequest struct {
	Name string `json:"name"`
}

type DeleteMCPServerResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RegisterTools registers all MCP server-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// List MCP servers
	listMCPServersTool := mcp.NewTool("list_mcp_servers",
		mcp.WithDescription("List all MCP servers (functions) in the workspace"),
		mcp.WithString("filter",
			mcp.Description("Optional filter string"),
		),
	)

	s.AddTool(listMCPServersTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req ListMCPServersRequest
		if err := request.BindArguments(&req); err != nil {
			// If binding fails, try to get filter directly for backward compatibility
			req.Filter = request.GetString("filter", "")
		}

		result, err := listMCPServersHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(*result)), nil
	})

	// Get MCP server
	getMCPServerTool := mcp.NewTool("get_mcp_server",
		mcp.WithDescription("Get details of a specific MCP server (function)"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the MCP server"),
		),
	)

	s.AddTool(getMCPServerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req GetMCPServerRequest
		if err := request.BindArguments(&req); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		if req.Name == "" {
			return mcp.NewToolResultError("MCP server name is required"), nil
		}

		result, err := getMCPServerHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(*result)), nil
	})

	if !cfg.ReadOnly {
		// Create MCP server
		createMCPServerTool := mcp.NewTool("create_mcp_server",
			mcp.WithDescription("Create an MCP server (function) with flexible integration options"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name for the MCP server"),
			),
			mcp.WithString("integrationConnectionName",
				mcp.Description("Existing integration to use"),
			),
			mcp.WithString("integrationType",
				mcp.Description("Type for new integration (e.g., github)"),
			),
			mcp.WithObject("secret",
				mcp.Description("Secrets for new integration"),
			),
			mcp.WithObject("config",
				mcp.Description("Config for new integration"),
			),
		)

		s.AddTool(createMCPServerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req CreateMCPServerRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Name == "" {
				return mcp.NewToolResultError("MCP server name is required"), nil
			}

			result, err := createMCPServerHandler(ctx, sdkClient, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(jsonResult)), nil
		})

		// Delete MCP server
		deleteMCPServerTool := mcp.NewTool("delete_mcp_server",
			mcp.WithDescription("Delete an MCP server (function) by name"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the MCP server to delete"),
			),
		)

		s.AddTool(deleteMCPServerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req DeleteMCPServerRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Name == "" {
				return mcp.NewToolResultError("MCP server name is required"), nil
			}

			result, err := deleteMCPServerHandler(ctx, sdkClient, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(jsonResult)), nil
		})
	}
}

func listMCPServersHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req ListMCPServersRequest) (*ListMCPServersResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	servers, err := sdkClient.ListFunctionsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP servers: %w", err)
	}
	if servers.JSON200 == nil {
		return nil, fmt.Errorf("no MCP servers found")
	}

	// Use generic filter and marshal function
	jsonData, _ := tools.FilterAndMarshal(servers.JSON200, req.Filter, func(server sdk.Function) string {
		if server.Metadata != nil && server.Metadata.Name != nil {
			return *server.Metadata.Name
		}
		return ""
	})

	response := ListMCPServersResponse(jsonData)
	return &response, nil
}

func getMCPServerHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req GetMCPServerRequest) (*GetMCPServerResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	server, err := sdkClient.GetFunctionWithResponse(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server: %w", err)
	}
	if server.JSON200 == nil {
		return nil, fmt.Errorf("no MCP server found")
	}

	jsonData, _ := json.MarshalIndent(*server.JSON200, "", "  ")
	response := GetMCPServerResponse(jsonData)
	return &response, nil
}

func createMCPServerHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req CreateMCPServerRequest) (*CreateMCPServerResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Check for integration parameters
	hasExisting := req.IntegrationConnectionName != ""
	hasNewType := req.IntegrationType != ""

	// Validate integration parameters
	if hasExisting && hasNewType {
		return nil, fmt.Errorf("specify either integrationConnectionName or integrationType, not both")
	}

	// Build MCP server request
	functionData := sdk.CreateFunctionJSONRequestBody{
		Metadata: &sdk.Metadata{
			Name: &req.Name,
		},
		Spec: &sdk.FunctionSpec{
			Runtime: &sdk.Runtime{},
		},
	}

	// Handle integration configuration
	var integrationName string
	if hasExisting {
		// Use existing integration connection
		if req.IntegrationConnectionName == "" {
			return nil, fmt.Errorf("integrationConnectionName cannot be empty")
		}
		integrationName = req.IntegrationConnectionName
	} else if hasNewType {
		// Create inline integration for the MCP server
		if req.IntegrationType == "" {
			return nil, fmt.Errorf("integrationType cannot be empty")
		}

		// Generate a unique name for the integration
		integrationName = fmt.Sprintf("%s-%s-integration", req.Name, req.IntegrationType)

		// Create the integration
		integrationData := sdk.CreateIntegrationConnectionJSONRequestBody{
			Metadata: &sdk.Metadata{
				Name: &integrationName,
			},
			Spec: &sdk.IntegrationConnectionSpec{
				Integration: &req.IntegrationType,
			},
		}

		// Add secrets if provided
		if len(req.Secret) > 0 {
			integrationData.Spec.Secret = &req.Secret
		}

		// Add config if provided
		if len(req.Config) > 0 {
			integrationData.Spec.Config = &req.Config
		}

		// Create the integration
		integrationResp, err := sdkClient.CreateIntegrationConnectionWithResponse(ctx, integrationData)
		if err != nil {
			return nil, fmt.Errorf("failed to create inline integration: %w", err)
		}

		if integrationResp.StatusCode() >= 400 {
			if integrationResp.StatusCode() == 409 {
				// Integration might already exist, try to use it
				fmt.Printf("Integration '%s' already exists, will attempt to use it\n", integrationName)
			} else {
				return nil, fmt.Errorf("failed to create integration with status %d", integrationResp.StatusCode())
			}
		}
	}

	// Set the integration connection on the MCP server
	if integrationName != "" {
		connections := sdk.IntegrationConnectionsList{integrationName}
		functionData.Spec.IntegrationConnections = &connections
	}

	// Create the MCP server
	function, err := sdkClient.CreateFunctionWithResponse(ctx, functionData)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	if function.JSON200 == nil {
		if function.StatusCode() == 409 {
			return nil, fmt.Errorf("MCP server with name '%s' already exists", req.Name)
		}
		return nil, fmt.Errorf("failed to create MCP server with status %d", function.StatusCode())
	}

	result := &CreateMCPServerResponse{
		Success: true,
		Message: fmt.Sprintf("MCP server '%s' created successfully", req.Name),
		MCPServer: map[string]interface{}{
			"name": req.Name,
		},
	}

	// Add integration details to result
	if integrationName != "" {
		result.MCPServer["integrationConnection"] = integrationName
		if hasNewType {
			result.Message = fmt.Sprintf("MCP server '%s' created successfully with inline integration '%s'", req.Name, integrationName)
			result.MCPServer["integrationType"] = req.IntegrationType
		}
	}

	return result, nil
}

func deleteMCPServerHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req DeleteMCPServerRequest) (*DeleteMCPServerResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	_, err := sdkClient.DeleteFunctionWithResponse(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete MCP server: %w", err)
	}

	return &DeleteMCPServerResponse{
		Success: true,
		Message: fmt.Sprintf("MCP server '%s' deleted successfully", req.Name),
	}, nil
}
