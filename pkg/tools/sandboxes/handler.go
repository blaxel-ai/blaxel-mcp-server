package sandboxes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/client"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/formatter"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"
	"github.com/blaxel-ai/toolkit/sdk"
)

// SDKHandler implements SandboxHandler using the SDK client
type SDKHandler struct {
	sdkClient *sdk.ClientWithResponses
	readOnly  bool
}

// NewSDKHandler creates a new SDK-based sandbox handler
func NewSDKHandler(cfg *config.Config) (SandboxHandler, error) {
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SDK client: %w", err)
	}

	return &SDKHandler{
		sdkClient: sdkClient,
		readOnly:  cfg.ReadOnly,
	}, nil
}

// ListSandboxes implements SandboxHandler.ListSandboxes
func (h *SDKHandler) ListSandboxes(ctx context.Context, filter string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := h.sdkClient.ListSandboxesWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sandboxes: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("list sandboxes failed with status %d", resp.StatusCode())
	}

	sandboxes := []sdk.Sandbox{}
	if resp.JSON200 != nil {
		sandboxes = *resp.JSON200
	}

	// Apply filter if requested
	if filter != "" {
		var filtered []sdk.Sandbox
		for _, sandbox := range sandboxes {
			if sandbox.Metadata != nil && sandbox.Metadata.Name != nil &&
				tools.ContainsString(*sandbox.Metadata.Name, filter) {
				filtered = append(filtered, sandbox)
			}
		}
		sandboxes = filtered
	}

	// Convert SDK sandboxes to simple models
	sandboxModels := make([]formatter.SandboxModel, len(sandboxes))
	for i, sandbox := range sandboxes {
		sandboxModels[i] = convertToSandboxModel(sandbox)
	}

	// Format the sandboxes using the formatter
	formatted := formatter.FormatSandboxes(sandboxModels)
	return []byte(formatted), nil
}

// GetSandbox implements SandboxHandler.GetSandbox
func (h *SDKHandler) GetSandbox(ctx context.Context, name string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	sandbox, err := h.sdkClient.GetSandboxWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox: %w", err)
	}

	if sandbox.JSON200 == nil {
		return nil, fmt.Errorf("no sandbox found")
	}

	// Convert to JSON for better formatting
	jsonData, err := json.MarshalIndent(*sandbox.JSON200, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format sandbox data: %w", err)
	}

	return jsonData, nil
}

// CreateSandbox implements SandboxHandler.CreateSandbox
func (h *SDKHandler) CreateSandbox(ctx context.Context, name, image string, memory float64, ports, env string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Build sandbox request
	sandboxData := sdk.CreateSandboxJSONRequestBody{
		Metadata: &sdk.Metadata{
			Name: &name,
		},
		Spec: &sdk.SandboxSpec{
			Runtime: &sdk.Runtime{},
		},
	}

	// Add optional image
	if image != "" {
		sandboxData.Spec.Runtime.Image = &image
	}

	// Add optional memory
	if memory > 0 {
		mem := int(memory)
		sandboxData.Spec.Runtime.Memory = &mem
	}

	// Add optional ports
	if ports != "" {
		defaultProtocol := "TCP"
		portStrings := strings.Split(ports, ",")
		portsList := make([]sdk.Port, 0, len(portStrings))
		for _, portStr := range portStrings {
			portStr = strings.TrimSpace(portStr)
			if portStr == "" {
				continue
			}
			intPort, err := strconv.Atoi(portStr)
			if err != nil {
				return nil, fmt.Errorf("invalid port '%s': %w", portStr, err)
			}
			portData := sdk.Port{
				Target:   &intPort,
				Protocol: &defaultProtocol,
			}
			portsList = append(portsList, portData)
		}
		if len(portsList) > 0 {
			sandboxData.Spec.Runtime.Ports = &portsList
		}
	}

	// Add optional environment variables
	if env != "" {
		sandboxData.Spec.Runtime.Envs = tools.SetRuntimeEnv(env)
	}

	// Create sandbox
	sandbox, err := h.sdkClient.CreateSandboxWithResponse(ctx, sandboxData)
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox: %w", err)
	}

	if sandbox.JSON200 == nil {
		if sandbox.StatusCode() == 409 {
			return nil, fmt.Errorf("sandbox with name '%s' already exists", name)
		}
		return nil, fmt.Errorf("failed to create sandbox with status %d", sandbox.StatusCode())
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Sandbox '%s' created successfully", name),
		"sandbox": map[string]interface{}{
			"name": name,
		},
	}

	if sandbox.JSON200 != nil && sandbox.JSON200.Status != nil {
		result["sandbox"].(map[string]interface{})["status"] = *sandbox.JSON200.Status
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return jsonData, nil
}

// DeleteSandbox implements SandboxHandler.DeleteSandbox
func (h *SDKHandler) DeleteSandbox(ctx context.Context, name string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	_, err := h.sdkClient.DeleteSandboxWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete sandbox: %w", err)
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Sandbox '%s' deleted successfully", name),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return jsonData, nil
}

// IsReadOnly implements SandboxHandlerWithReadOnly.IsReadOnly
func (h *SDKHandler) IsReadOnly() bool {
	return h.readOnly
}

// convertToSandboxModel converts an SDK sandbox to a simple sandbox model
func convertToSandboxModel(sandbox sdk.Sandbox) formatter.SandboxModel {
	model := formatter.SandboxModel{
		Name:   "",
		Status: "",
		Labels: make(map[string]string),
	}

	// Extract name
	if sandbox.Metadata != nil && sandbox.Metadata.Name != nil {
		model.Name = *sandbox.Metadata.Name
	}

	// Extract status
	if sandbox.Status != nil {
		model.Status = *sandbox.Status
	}

	// Extract labels
	if sandbox.Metadata != nil && sandbox.Metadata.Labels != nil {
		model.Labels = *sandbox.Metadata.Labels
	}

	// Extract runtime spec
	if sandbox.Spec != nil && sandbox.Spec.Runtime != nil {
		if sandbox.Spec.Runtime.Image != nil {
			model.Image = sandbox.Spec.Runtime.Image
		}
		if sandbox.Spec.Runtime.Generation != nil {
			model.Generation = sandbox.Spec.Runtime.Generation
		}
		if sandbox.Spec.Runtime.Memory != nil {
			model.Memory = sandbox.Spec.Runtime.Memory
		}
		if sandbox.Spec.Runtime.Ttl != nil {
			model.TTL = sandbox.Spec.Runtime.Ttl
		}
		if sandbox.Spec.Runtime.Expires != nil {
			// Parse the time string to time.Time
			if expires, err := time.Parse(time.RFC3339, *sandbox.Spec.Runtime.Expires); err == nil {
				model.Expires = &expires
			}
		}
		// Note: Ports conversion skipped due to SDK type complexity
		// if sandbox.Spec.Runtime.Ports != nil {
		// 	model.Ports = convertPorts(*sandbox.Spec.Runtime.Ports)
		// }
	}

	// Extract creation time
	if sandbox.Metadata != nil && sandbox.Metadata.CreatedAt != nil {
		// Parse the time string to time.Time
		if createdAt, err := time.Parse(time.RFC3339, *sandbox.Metadata.CreatedAt); err == nil {
			model.CreatedAt = &createdAt
		}
	}

	return model
}
