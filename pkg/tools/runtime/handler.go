package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/client"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/toolkit/sdk"
)

// SDKHandler implements RuntimeHandler using the SDK client
type SDKHandler struct {
	sdkClient *sdk.ClientWithResponses
	cfg       *config.Config
	readOnly  bool
}

// NewSDKHandler creates a new SDK-based runtime handler
func NewSDKHandler(cfg *config.Config) (RuntimeHandler, error) {
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SDK client: %w", err)
	}

	return &SDKHandler{
		sdkClient: sdkClient,
		cfg:       cfg,
		readOnly:  cfg.ReadOnly,
	}, nil
}

// RunAgent implements RuntimeHandler.RunAgent
func (h *SDKHandler) RunAgent(ctx context.Context, name, message, context string) (string, error) {
	if h.sdkClient == nil {
		return "", fmt.Errorf("SDK client not initialized")
	}

	// Prepare the request body for the agent
	requestBody := map[string]interface{}{
		"inputs": message,
	}
	if context != "" {
		// Parse context JSON string into interface{}
		var contextData interface{}
		if err := json.Unmarshal([]byte(context), &contextData); err == nil {
			requestBody["context"] = contextData
		}
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use the SDK Run method to invoke the agent
	resp, err := h.sdkClient.Run(
		ctx,
		h.cfg.Workspace,
		"agent",
		name,
		"POST",
		"", // Path will be constructed by the Run method
		map[string]string{"Content-Type": "application/json"},
		nil, // No query params
		string(bodyBytes),
		false, // debug
		false, // local
	)
	if err != nil {
		return "", fmt.Errorf("failed to run agent: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("agent invocation failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Try to format as JSON for better readability
	var result interface{}
	if err := json.Unmarshal(body, &result); err == nil {
		formatted, _ := json.MarshalIndent(result, "", "  ")
		return string(formatted), nil
	}

	return string(body), nil
}

// RunJob implements RuntimeHandler.RunJob
func (h *SDKHandler) RunJob(ctx context.Context, name, parameters string) (string, error) {
	if h.sdkClient == nil {
		return "", fmt.Errorf("SDK client not initialized")
	}

	// Prepare the request body for the job
	var bodyBytes []byte
	if parameters != "" {
		// Use the parameters JSON string directly
		bodyBytes = []byte(parameters)
	} else {
		bodyBytes = []byte("{}")
	}

	// Use the SDK Run method to trigger the job
	resp, err := h.sdkClient.Run(
		ctx,
		h.cfg.Workspace,
		"job",
		name,
		"POST",
		"", // Path will be constructed by the Run method
		map[string]string{"Content-Type": "application/json"},
		nil, // No query params
		string(bodyBytes),
		false, // debug
		false, // local
	)
	if err != nil {
		return "", fmt.Errorf("failed to run job: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("job trigger failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Try to format as JSON for better readability
	var result interface{}
	if err := json.Unmarshal(body, &result); err == nil {
		formatted, _ := json.MarshalIndent(result, "", "  ")
		return fmt.Sprintf("Job triggered successfully:\n%s", string(formatted)), nil
	}

	return fmt.Sprintf("Job triggered successfully:\n%s", string(body)), nil
}

// RunModel implements RuntimeHandler.RunModel
func (h *SDKHandler) RunModel(ctx context.Context, name, body, path, method string) (string, error) {
	if h.sdkClient == nil {
		return "", fmt.Errorf("SDK client not initialized")
	}

	// Prepare the request body for the model
	// Use the body JSON string directly
	bodyBytes := []byte(body)

	// Use the SDK Run method to invoke the model
	resp, err := h.sdkClient.Run(
		ctx,
		h.cfg.Workspace,
		"model",
		name,
		method,
		path,
		map[string]string{"Content-Type": "application/json"},
		nil, // No query params
		string(bodyBytes),
		false, // debug
		false, // local
	)
	if err != nil {
		return "", fmt.Errorf("failed to run model: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("model invocation failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Try to format as JSON for better readability
	var result interface{}
	if err := json.Unmarshal(bodyBytes, &result); err == nil {
		formatted, _ := json.MarshalIndent(result, "", "  ")
		return string(formatted), nil
	}

	return string(bodyBytes), nil
}

// RunSandbox implements RuntimeHandler.RunSandbox
func (h *SDKHandler) RunSandbox(ctx context.Context, name, body, method, path string) (string, error) {
	if h.sdkClient == nil {
		return "", fmt.Errorf("SDK client not initialized")
	}

	// Prepare the request body for the sandbox
	// Use the body JSON string directly
	bodyBytes := []byte(body)

	// First, ensure the sandbox is started
	startResp, err := h.sdkClient.StartSandboxWithResponse(ctx, name)
	if err != nil {
		return "", fmt.Errorf("failed to start sandbox: %w", err)
	}
	if startResp.StatusCode() != http.StatusOK && startResp.StatusCode() != http.StatusConflict {
		return "", fmt.Errorf("failed to start sandbox with status %d", startResp.StatusCode())
	}

	// Use the SDK Run method to execute code in the sandbox
	resp, err := h.sdkClient.Run(
		ctx,
		h.cfg.Workspace,
		"sandbox",
		name,
		method,
		path,
		map[string]string{"Content-Type": "application/json"},
		nil, // No query params
		string(bodyBytes),
		false, // debug
		false, // local
	)
	if err != nil {
		return "", fmt.Errorf("failed to execute in sandbox: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("sandbox execution failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Try to format as JSON for better readability
	var result interface{}
	if err := json.Unmarshal(bodyBytes, &result); err == nil {
		formatted, _ := json.MarshalIndent(result, "", "  ")
		return string(formatted), nil
	}

	return string(bodyBytes), nil
}

// IsReadOnly implements RuntimeHandlerWithReadOnly.IsReadOnly
func (h *SDKHandler) IsReadOnly() bool {
	return h.readOnly
}
