package mcpservers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

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

// resolveHandler returns a handler for the given workspace override.
func resolveHandler(defaultHandler MCPServerHandler, cfg *config.Config, workspace string) (MCPServerHandler, error) {
	overriddenCfg := cfg.WithWorkspace(workspace)
	if overriddenCfg == cfg {
		return defaultHandler, nil
	}
	return NewSDKHandler(overriddenCfg)
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

	// Convert SDK functions to simple models
	functionModels := make([]formatter.FunctionModel, len(functions))
	for i, function := range functions {
		functionModels[i] = convertToFunctionModel(function)
	}

	// Format the functions using the formatter
	formatted := formatter.FormatFunctions(functionModels)
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
	waitForCompletionBool, err := normalizeLifecycleWait(waitForCompletion)
	if err != nil {
		return nil, err
	}
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Check for integration parameters
	hasExisting := integrationConnectionName != ""
	hasNewType := integrationType != ""

	// Integrations are optional, but the two integration configuration modes are
	// mutually exclusive when one is requested.
	if hasExisting && hasNewType {
		return nil, fmt.Errorf("specify either integrationConnectionName or integrationType, not both")
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

	// Wait for the MCP server to reach a final status if requested
	deploymentFinalStatus := ""
	if waitForCompletionBool {
		logger.Printf("Waiting for MCP server '%s' to deploy...", name)
		checker := NewMCPServerStatusChecker(h.sdkClient)
		err = utils.WaitForResourceStatus(ctx, name, checker)
		deploymentFinalStatus = checker.LastStatus()
		if err != nil {
			return nil, fmt.Errorf("MCP server '%s' status wait failed: %w", name, err)
		}
		if deploymentFinalStatus == "FAILED" {
			return nil, fmt.Errorf("MCP server '%s' deployment reached terminal status FAILED", name)
		}
	} else {
		logger.Printf("Skipping status wait for MCP server '%s'", name)
	}

	// MCP server successfully created (and reached a terminal state if we waited).
	message := fmt.Sprintf("MCP server '%s' creation accepted", name)
	switch deploymentFinalStatus {
	case "DEPLOYED":
		message = fmt.Sprintf("MCP server '%s' created and deployed successfully", name)
	}

	result := map[string]interface{}{
		"success": true,
		"message": message,
		"mcp_server": map[string]interface{}{
			"name": name,
		},
	}
	if deploymentFinalStatus != "" {
		result["mcp_server"].(map[string]interface{})["status"] = deploymentFinalStatus
	}

	// Add integration details to result
	if integrationName != "" {
		result["mcp_server"].(map[string]interface{})["integrationConnection"] = integrationName
		if hasNewType {
			result["message"] = fmt.Sprintf("%s with inline integration '%s'", message, integrationName)
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
	waitForCompletionBool, err := normalizeLifecycleWait(waitForCompletion)
	if err != nil {
		return nil, err
	}
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Delete the MCP server
	resp, err := h.sdkClient.DeleteFunctionWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete MCP server: %w", err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("MCP server '%s' not found", name)
	}
	if resp.StatusCode() < http.StatusOK || resp.StatusCode() >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("failed to delete MCP server '%s': status %d", name, resp.StatusCode())
	}

	// Wait for the MCP server to be fully deleted if requested
	if waitForCompletionBool {
		logger.Printf("Waiting for MCP server '%s' to be fully deleted...", name)
		checker := NewMCPServerStatusChecker(h.sdkClient)
		if err = utils.WaitForResourceDeletion(ctx, name, checker); err != nil {
			return nil, fmt.Errorf("MCP server '%s' deletion wait failed: %w", name, err)
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

func normalizeLifecycleWait(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("waitForCompletion must be true or false")
	}
}

// IsReadOnly implements MCPServerHandlerWithReadOnly.IsReadOnly
func (h *SDKHandler) IsReadOnly() bool {
	return h.readOnly
}

// MCPServerStatusChecker implements StatusChecker for MCP servers
type MCPServerStatusChecker struct {
	sdkClient  *sdk.ClientWithResponses
	lastStatus string
}

// NewMCPServerStatusChecker creates a new MCP server status checker
func NewMCPServerStatusChecker(sdkClient *sdk.ClientWithResponses) *MCPServerStatusChecker {
	return &MCPServerStatusChecker{sdkClient: sdkClient}
}

// GetResource gets the MCP server resource
func (m *MCPServerStatusChecker) GetResource(ctx context.Context, name string) (interface{}, error) {
	resp, err := m.sdkClient.GetFunctionWithResponse(ctx, name)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("MCP server '%s' not found (status 404)", name)
	}
	if resp.StatusCode() < http.StatusOK || resp.StatusCode() >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("get MCP server '%s' failed with status %d", name, resp.StatusCode())
	}
	return resp, nil
}

// ExtractStatus extracts status from MCP server response
func (m *MCPServerStatusChecker) ExtractStatus(resource interface{}) string {
	// Type assertion to get the function response
	if functionResp, ok := resource.(*sdk.GetFunctionResponse); ok {
		if functionResp.JSON200 != nil {
			if functionResp.JSON200.Status == nil {
				m.lastStatus = "DEPLOYING"
				return m.lastStatus
			}
			m.lastStatus = *functionResp.JSON200.Status
			return m.lastStatus
		}
	}
	m.lastStatus = "DEPLOYING"
	return m.lastStatus // Default assumption
}

// GetResourceType returns the resource type
func (m *MCPServerStatusChecker) GetResourceType() utils.ResourceType {
	return "mcp_server"
}

// LastStatus returns the latest status observed while polling.
func (m *MCPServerStatusChecker) LastStatus() string {
	return m.lastStatus
}

// convertToFunctionModel converts an SDK function to a simple function model
func convertToFunctionModel(function sdk.Function) formatter.FunctionModel {
	model := formatter.FunctionModel{
		Name:   "",
		Status: "",
		Labels: make(map[string]string),
	}

	// Extract name
	if function.Metadata != nil && function.Metadata.Name != nil {
		model.Name = *function.Metadata.Name
	}

	// Extract status
	if function.Status != nil {
		model.Status = *function.Status
	}

	// Extract labels
	if function.Metadata != nil && function.Metadata.Labels != nil {
		model.Labels = *function.Metadata.Labels
	}

	// Extract runtime spec
	if function.Spec != nil && function.Spec.Runtime != nil {
		if function.Spec.Runtime.Image != nil {
			model.Image = function.Spec.Runtime.Image
		}
		if function.Spec.Runtime.Generation != nil {
			model.Generation = function.Spec.Runtime.Generation
		}
		if function.Spec.Runtime.Memory != nil {
			model.Memory = function.Spec.Runtime.Memory
		}
	}

	// Extract integration connections
	if function.Spec != nil && function.Spec.IntegrationConnections != nil {
		model.IntegrationConnections = *function.Spec.IntegrationConnections
	}

	// Extract creation time
	if function.Metadata != nil && function.Metadata.CreatedAt != nil {
		// Parse the time string to time.Time
		if createdAt, err := time.Parse(time.RFC3339, *function.Metadata.CreatedAt); err == nil {
			model.CreatedAt = &createdAt
		}
	}

	return model
}
