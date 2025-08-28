package mcpservers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/client"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/formatter"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/logger"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/utils"
	"github.com/blaxel-ai/toolkit/sdk"
)

// SDKHandler implements MCPServerHandler using the SDK client
type SDKHandler struct {
	sdkClient *sdk.ClientWithResponses
	readOnly  bool
}

// NewSDKHandler creates a new SDK-based MCP server handler
func NewSDKHandler(cfg *config.Config) (MCPServerHandler, error) {
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SDK client: %w", err)
	}

	return &SDKHandler{
		sdkClient: sdkClient,
		readOnly:  cfg.ReadOnly,
	}, nil
}

// ListMCPServers implements MCPServerHandler.ListMCPServers
func (h *SDKHandler) ListMCPServers(ctx context.Context, filter string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := h.sdkClient.ListFunctionsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP servers: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("list MCP servers failed with status %d", resp.StatusCode())
	}

	functions := []sdk.Function{}
	if resp.JSON200 != nil {
		functions = *resp.JSON200
	}

	// Apply filter if requested
	if filter != "" {
		var filtered []sdk.Function
		for _, fn := range functions {
			if fn.Metadata != nil && fn.Metadata.Name != nil &&
				tools.ContainsString(*fn.Metadata.Name, filter) {
				filtered = append(filtered, fn)
			}
		}
		functions = filtered
	}

	// Format the functions using the formatter
	formatted := formatter.FormatFunctions(functions)
	return []byte(formatted), nil
}

// GetMCPServer implements MCPServerHandler.GetMCPServer
func (h *SDKHandler) GetMCPServer(ctx context.Context, name string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	server, err := h.sdkClient.GetFunctionWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server: %w", err)
	}

	if server.JSON200 == nil {
		return nil, fmt.Errorf("no MCP server found")
	}

	// Convert to JSON for better formatting
	jsonData, err := json.MarshalIndent(*server.JSON200, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format MCP server data: %w", err)
	}

	return jsonData, nil
}

// CreateMCPServer implements MCPServerHandler.CreateMCPServer
func (h *SDKHandler) CreateMCPServer(ctx context.Context, name, integrationConnectionName, integrationType, waitForCompletion string, secret, config map[string]string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Check for integration parameters
	hasExisting := integrationConnectionName != ""
	hasNewType := integrationType != ""

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
			Name: &name,
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
		if integrationConnectionName == "" {
			return nil, fmt.Errorf("integrationConnectionName cannot be empty")
		}
		integrationName = integrationConnectionName
	} else if hasNewType {
		// Create inline integration for the MCP server
		if integrationType == "" {
			return nil, fmt.Errorf("integrationType cannot be empty")
		}

		// Generate a unique name for the integration
		integrationName = fmt.Sprintf("%s-%s-integration", name, integrationType)

		// Create the integration
		integrationData := sdk.CreateIntegrationConnectionJSONRequestBody{
			Metadata: &sdk.Metadata{
				Name: &integrationName,
			},
			Spec: &sdk.IntegrationConnectionSpec{
				Integration: &integrationType,
			},
		}

		// Add secrets if provided
		if len(secret) > 0 {
			integrationData.Spec.Secret = &secret
		}

		// Add config if provided
		if len(config) > 0 {
			integrationData.Spec.Config = &config
		}

		// Create the integration
		integrationResp, err := h.sdkClient.CreateIntegrationConnectionWithResponse(ctx, integrationData)
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
	function, err := h.sdkClient.CreateFunctionWithResponse(ctx, functionData)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	if function.JSON200 == nil {
		if function.StatusCode() == 409 {
			return nil, fmt.Errorf("MCP server with name '%s' already exists", name)
		}
		return nil, fmt.Errorf("failed to create MCP server with status %d", function.StatusCode())
	}

	// Check if we should wait for completion
	waitForCompletionBool := true // default to true
	if waitForCompletion != "" {
		waitForCompletionBool = waitForCompletion == "true"
	}

	// Wait for the MCP server to reach a final status if requested
	if waitForCompletionBool {
		logger.Printf("Waiting for MCP server '%s' to deploy...", name)
		checker := NewMCPServerStatusChecker(h.sdkClient)
		err = utils.WaitForResourceStatus(ctx, name, checker)
		if err != nil {
			// Even if status waiting fails, we still created the MCP server
			// Return a warning but don't fail the entire operation
			logger.Printf("Warning: MCP server created but status check failed: %v", err)
			result := map[string]interface{}{
				"success": true,
				"message": fmt.Sprintf("MCP server '%s' created successfully (status check failed: %v)", name, err),
				"mcp_server": map[string]interface{}{
					"name": name,
				},
			}

			// Add integration details to result
			if integrationName != "" {
				result["mcp_server"].(map[string]interface{})["integrationConnection"] = integrationName
				if hasNewType {
					result["message"] = fmt.Sprintf("MCP server '%s' created successfully with inline integration '%s' (status check failed: %v)", name, integrationName, err)
					result["mcp_server"].(map[string]interface{})["integrationType"] = integrationType
				}
			}

			jsonData, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to format response: %w", err)
			}

			return jsonData, nil
		}
	} else {
		logger.Printf("Skipping status wait for MCP server '%s'", name)
	}

	// MCP server successfully created (and deployed if we waited)
	deploymentStatus := "created"
	if waitForCompletionBool {
		deploymentStatus = "created and deployed"
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("MCP server '%s' %s successfully", name, deploymentStatus),
		"mcp_server": map[string]interface{}{
			"name": name,
		},
	}

	// Add integration details to result
	if integrationName != "" {
		result["mcp_server"].(map[string]interface{})["integrationConnection"] = integrationName
		if hasNewType {
			result["message"] = fmt.Sprintf("MCP server '%s' %s successfully with inline integration '%s'", name, deploymentStatus, integrationName)
			result["mcp_server"].(map[string]interface{})["integrationType"] = integrationType
		}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return jsonData, nil
}

// DeleteMCPServer implements MCPServerHandler.DeleteMCPServer
func (h *SDKHandler) DeleteMCPServer(ctx context.Context, name, waitForCompletion string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Delete the MCP server
	_, err := h.sdkClient.DeleteFunctionWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete MCP server: %w", err)
	}

	// Check if we should wait for completion
	waitForCompletionBool := true // default to true
	if waitForCompletion != "" {
		waitForCompletionBool = waitForCompletion == "true"
	}

	// Wait for the MCP server to be fully deleted if requested
	if waitForCompletionBool {
		logger.Printf("Waiting for MCP server '%s' to be fully deleted...", name)
		checker := NewMCPServerStatusChecker(h.sdkClient)
		err = utils.WaitForResourceDeletion(ctx, name, checker)
		if err != nil {
			// Even if deletion polling fails, we still initiated the deletion
			// Return a warning but don't fail the entire operation
			logger.Printf("Warning: MCP server deletion initiated but status check failed: %v", err)
			result := map[string]interface{}{
				"success": true,
				"message": fmt.Sprintf("MCP server '%s' deletion initiated (status check failed: %v)", name, err),
			}

			jsonData, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to format response: %w", err)
			}

			return jsonData, nil
		}
	} else {
		logger.Printf("Skipping deletion wait for MCP server '%s'", name)
	}

	// MCP server successfully deleted (or deletion initiated)
	deletionStatus := "deleted"
	if !waitForCompletionBool {
		deletionStatus = "deletion initiated"
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("MCP server '%s' %s successfully", name, deletionStatus),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return jsonData, nil
}

// IsReadOnly implements MCPServerHandlerWithReadOnly.IsReadOnly
func (h *SDKHandler) IsReadOnly() bool {
	return h.readOnly
}

// MCPServerStatusChecker implements StatusChecker for MCP servers
type MCPServerStatusChecker struct {
	sdkClient *sdk.ClientWithResponses
}

// NewMCPServerStatusChecker creates a new MCP server status checker
func NewMCPServerStatusChecker(sdkClient *sdk.ClientWithResponses) *MCPServerStatusChecker {
	return &MCPServerStatusChecker{sdkClient: sdkClient}
}

// GetResource gets the MCP server resource
func (m *MCPServerStatusChecker) GetResource(ctx context.Context, name string) (interface{}, error) {
	return m.sdkClient.GetFunctionWithResponse(ctx, name)
}

// ExtractStatus extracts status from MCP server response
func (m *MCPServerStatusChecker) ExtractStatus(resource interface{}) string {
	// Type assertion to get the function response
	if functionResp, ok := resource.(*sdk.GetFunctionResponse); ok {
		if functionResp.JSON200 != nil {
			if functionResp.JSON200.Status == nil {
				return "DEPLOYING"
			}
			return *functionResp.JSON200.Status
		}
	}
	return "DEPLOYING" // Default assumption
}

// GetResourceType returns the resource type
func (m *MCPServerStatusChecker) GetResourceType() utils.ResourceType {
	return "mcp_server"
}
