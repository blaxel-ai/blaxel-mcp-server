package mcpservers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/client"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/logger"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/tools"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/tools/utils"
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
	WaitForCompletion         string            `json:"waitForCompletion,omitempty"`
}

type CreateMCPServerResponse struct {
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	MCPServer map[string]interface{} `json:"mcp_server"`
}

type DeleteMCPServerRequest struct {
	Name              string `json:"name"`
	WaitForCompletion string `json:"waitForCompletion,omitempty"`
}

type DeleteMCPServerResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Note: Status polling functionality has been moved to internal/tools/utils/status_polling.go

// RegisterTools registers all MCP server-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		logger.Warnf("Failed to initialize SDK client: %v", err)
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
			mcp.WithString("waitForCompletion",
				mcp.Description("Whether to wait for the MCP server to reach a final status (true/false, default: true)"),
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
			mcp.WithString("waitForCompletion",
				mcp.Description("Whether to wait for the MCP server to be fully deleted (true/false, default: true)"),
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

	resp, err := sdkClient.ListFunctionsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP servers: %w", err)
	}

	functions := []sdk.Function{}
	if resp.JSON200 != nil {
		functions = *resp.JSON200
	}

	// Apply filter if requested
	if req.Filter != "" {
		var filtered []sdk.Function
		for _, fn := range functions {
			if fn.Metadata != nil && fn.Metadata.Name != nil &&
				tools.ContainsString(*fn.Metadata.Name, req.Filter) {
				filtered = append(filtered, fn)
			}
		}
		functions = filtered
	}

	// Format the functions using the new formatter
	formatted := tools.FormatFunctions(functions)
	jsonData, err := json.Marshal(formatted)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal formatted functions: %w", err)
	}

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
	if !hasExisting && !hasNewType {
		return nil, fmt.Errorf("must provide either integrationConnectionName to reference an existing integration or integrationType to create a new one")
	}

	// Build MCP server request
	runtimeType := "mcp"
	functionData := sdk.CreateFunctionJSONRequestBody{
		Metadata: &sdk.Metadata{
			Name: &req.Name,
		},
		Spec: &sdk.FunctionSpec{
			Runtime: &sdk.Runtime{
				Type: &runtimeType,
			},
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
				logger.Printf("Integration '%s' already exists, will attempt to use it", integrationName)
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

	// Check if we should wait for completion
	waitForCompletion := true // default to true
	if req.WaitForCompletion != "" {
		waitForCompletion = req.WaitForCompletion == "true"
	}

	// Wait for the MCP server to reach a final status if requested
	if waitForCompletion {
		logger.Printf("Waiting for MCP server '%s' to deploy...", req.Name)
		checker := utils.NewMCPServerStatusChecker(sdkClient)
		err = utils.WaitForResourceStatus(ctx, req.Name, checker)
		if err != nil {
			// Even if status waiting fails, we still created the MCP server
			// Return a warning but don't fail the entire operation
			logger.Printf("Warning: MCP server created but status check failed: %v", err)
			result := &CreateMCPServerResponse{
				Success: true,
				Message: fmt.Sprintf("MCP server '%s' created successfully (status check failed: %v)", req.Name, err),
				MCPServer: map[string]interface{}{
					"name": req.Name,
				},
			}

			// Add integration details to result
			if integrationName != "" {
				result.MCPServer["integrationConnection"] = integrationName
				if hasNewType {
					result.Message = fmt.Sprintf("MCP server '%s' created successfully with inline integration '%s' (status check failed: %v)", req.Name, integrationName, err)
					result.MCPServer["integrationType"] = req.IntegrationType
				}
			}

			return result, nil
		}
	} else {
		logger.Printf("Skipping status wait for MCP server '%s'", req.Name)
	}

	// MCP server successfully created (and deployed if we waited)
	deploymentStatus := "created"
	if waitForCompletion {
		deploymentStatus = "created and deployed"
	}

	result := &CreateMCPServerResponse{
		Success: true,
		Message: fmt.Sprintf("MCP server '%s' %s successfully", req.Name, deploymentStatus),
		MCPServer: map[string]interface{}{
			"name": req.Name,
		},
	}

	// Add integration details to result
	if integrationName != "" {
		result.MCPServer["integrationConnection"] = integrationName
		if hasNewType {
			result.Message = fmt.Sprintf("MCP server '%s' %s successfully with inline integration '%s'", req.Name, deploymentStatus, integrationName)
			result.MCPServer["integrationType"] = req.IntegrationType
		}
	}

	return result, nil
}

func deleteMCPServerHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req DeleteMCPServerRequest) (*DeleteMCPServerResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Delete the MCP server
	_, err := sdkClient.DeleteFunctionWithResponse(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete MCP server: %w", err)
	}

	// Check if we should wait for completion
	waitForCompletion := true // default to true
	if req.WaitForCompletion != "" {
		waitForCompletion = req.WaitForCompletion == "true"
	}

	// Wait for the MCP server to be fully deleted if requested
	if waitForCompletion {
		logger.Printf("Waiting for MCP server '%s' to be fully deleted...", req.Name)
		checker := utils.NewMCPServerStatusChecker(sdkClient)
		err = utils.WaitForResourceDeletion(ctx, req.Name, checker)
		if err != nil {
			// Even if deletion polling fails, we still initiated the deletion
			// Return a warning but don't fail the entire operation
			logger.Printf("Warning: MCP server deletion initiated but status check failed: %v", err)
			return &DeleteMCPServerResponse{
				Success: true,
				Message: fmt.Sprintf("MCP server '%s' deletion initiated (status check failed: %v)", req.Name, err),
			}, nil
		}
	} else {
		logger.Printf("Skipping deletion wait for MCP server '%s'", req.Name)
	}

	// MCP server successfully deleted (or deletion initiated)
	deletionStatus := "deleted"
	if !waitForCompletion {
		deletionStatus = "deletion initiated"
	}

	return &DeleteMCPServerResponse{
		Success: true,
		Message: fmt.Sprintf("MCP server '%s' %s successfully", req.Name, deletionStatus),
	}, nil
}
