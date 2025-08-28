package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/formatter"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"
	"github.com/blaxel-ai/toolkit/sdk"
)

// SDKAgentHandler implements AgentHandler using the SDK client
type SDKAgentHandler struct {
	sdkClient *sdk.ClientWithResponses
	readOnly  bool
}

// NewSDKAgentHandler creates a new SDK-based agent handler
func NewSDKAgentHandler(sdkClient *sdk.ClientWithResponses, readOnly bool) AgentHandler {
	return &SDKAgentHandler{
		sdkClient: sdkClient,
		readOnly:  readOnly,
	}
}

// ListAgents implements AgentHandler.ListAgents
func (h *SDKAgentHandler) ListAgents(ctx context.Context, filter string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := h.sdkClient.ListAgentsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("list agents failed with status %d", resp.StatusCode())
	}

	agents := []sdk.Agent{}
	if resp.JSON200 != nil {
		agents = *resp.JSON200
	}

	// Apply optional filter
	if filter != "" {
		var filtered []sdk.Agent
		for _, agent := range agents {
			if agent.Metadata != nil && agent.Metadata.Name != nil &&
				tools.ContainsString(*agent.Metadata.Name, filter) {
				filtered = append(filtered, agent)
			}
		}
		agents = filtered
	}

	// Format the agents using the formatter
	formatted := formatter.FormatAgents(agents)
	return []byte(formatted), nil
}

// GetAgent implements AgentHandler.GetAgent
func (h *SDKAgentHandler) GetAgent(ctx context.Context, name string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := h.sdkClient.GetAgentWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get agent failed with status %d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("agent not found")
	}

	// Convert to JSON for better formatting
	jsonData, err := json.MarshalIndent(resp.JSON200, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format agent data: %w", err)
	}

	return jsonData, nil
}

// DeleteAgent implements AgentHandler.DeleteAgent
func (h *SDKAgentHandler) DeleteAgent(ctx context.Context, name string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := h.sdkClient.DeleteAgentWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete agent: %w", err)
	}

	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return nil, fmt.Errorf("delete agent failed with status %d", resp.StatusCode())
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Agent '%s' deleted successfully", name),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return jsonData, nil
}

// IsReadOnly implements AgentHandlerWithReadOnly.IsReadOnly
func (h *SDKAgentHandler) IsReadOnly() bool {
	return h.readOnly
}
