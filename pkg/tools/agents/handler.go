package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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

	// Convert SDK agents to simple models
	agentModels := make([]formatter.AgentModel, len(agents))
	for i, agent := range agents {
		agentModels[i] = convertToAgentModel(agent)
	}

	// Format the agents using the formatter
	formatted := formatter.FormatAgents(agentModels)
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

// convertToAgentModel converts an SDK agent to a simple agent model
func convertToAgentModel(agent sdk.Agent) formatter.AgentModel {
	model := formatter.AgentModel{
		Name:   "",
		Status: "",
		Labels: make(map[string]string),
	}

	// Extract name
	if agent.Metadata != nil && agent.Metadata.Name != nil {
		model.Name = *agent.Metadata.Name
	}

	// Extract status
	if agent.Status != nil {
		model.Status = *agent.Status
	}

	// Extract labels
	if agent.Metadata != nil && agent.Metadata.Labels != nil {
		model.Labels = *agent.Metadata.Labels
	}

	// Extract runtime spec
	if agent.Spec != nil && agent.Spec.Runtime != nil {
		if agent.Spec.Runtime.Image != nil {
			model.Image = agent.Spec.Runtime.Image
		}
		if agent.Spec.Runtime.Generation != nil {
			model.Generation = agent.Spec.Runtime.Generation
		}
		if agent.Spec.Runtime.Memory != nil {
			model.Memory = agent.Spec.Runtime.Memory
		}
		if agent.Spec.Runtime.MaxConcurrentTasks != nil {
			model.MaxTasks = agent.Spec.Runtime.MaxConcurrentTasks
		}
	}

	// Extract creation time
	if agent.Metadata != nil && agent.Metadata.CreatedAt != nil {
		// Parse the time string to time.Time
		if createdAt, err := time.Parse(time.RFC3339, *agent.Metadata.CreatedAt); err == nil {
			model.CreatedAt = &createdAt
		}
	}

	return model
}
