package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	blaxel "github.com/blaxel-ai/sdk-go"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/client"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
)

// SDKHandler implements RuntimeHandler using the SDK client
type SDKHandler struct {
	blaxelClient *blaxel.Client
	cfg          *config.Config
	readOnly     bool
}

// NewSDKHandler creates a new SDK-based runtime handler
func NewSDKHandler(cfg *config.Config) (RuntimeHandler, error) {
	blaxelClient, err := client.NewBlaxelClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize blaxel client: %w", err)
	}

	return &SDKHandler{
		blaxelClient: blaxelClient,
		cfg:          cfg,
		readOnly:     cfg.ReadOnly,
	}, nil
}

// RunAgent implements RuntimeHandler.RunAgent
func (h *SDKHandler) RunAgent(ctx context.Context, name, message, agentContext string) (string, error) {
	if h.blaxelClient == nil {
		return "", fmt.Errorf("blaxel client not initialized")
	}

	// Prepare the request body for the agent
	requestBody := map[string]interface{}{
		"inputs": message,
	}
	if agentContext != "" {
		var contextData interface{}
		if err := json.Unmarshal([]byte(agentContext), &contextData); err == nil {
			requestBody["context"] = contextData
		}
	}

	resp, err := h.blaxelClient.RunWithMetadata(
		ctx,
		h.cfg.Workspace,
		"agent",
		name,
		"POST",
		"",
		requestBody,
	)
	if err != nil {
		return "", fmt.Errorf("failed to run agent: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("agent invocation failed with status %d: %s", resp.StatusCode, string(body))
	}

	return formatJSONResponse(body), nil
}

// RunJob implements RuntimeHandler.RunJob
func (h *SDKHandler) RunJob(ctx context.Context, name, parameters string) (string, error) {
	if h.blaxelClient == nil {
		return "", fmt.Errorf("blaxel client not initialized")
	}

	var bodyData interface{}
	if parameters != "" {
		if err := json.Unmarshal([]byte(parameters), &bodyData); err != nil {
			bodyData = map[string]interface{}{}
		}
	} else {
		bodyData = map[string]interface{}{}
	}

	resp, err := h.blaxelClient.RunWithMetadata(
		ctx,
		h.cfg.Workspace,
		"job",
		name,
		"POST",
		"",
		bodyData,
	)
	if err != nil {
		return "", fmt.Errorf("failed to run job: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("job trigger failed with status %d: %s", resp.StatusCode, string(body))
	}

	return fmt.Sprintf("Job triggered successfully:\n%s", formatJSONResponse(body)), nil
}

// RunModel implements RuntimeHandler.RunModel
func (h *SDKHandler) RunModel(ctx context.Context, name, bodyStr, path, method string) (string, error) {
	if h.blaxelClient == nil {
		return "", fmt.Errorf("blaxel client not initialized")
	}

	var bodyData interface{}
	if err := json.Unmarshal([]byte(bodyStr), &bodyData); err != nil {
		bodyData = json.RawMessage(bodyStr)
	}

	resp, err := h.blaxelClient.RunWithMetadata(
		ctx,
		h.cfg.Workspace,
		"model",
		name,
		method,
		path,
		bodyData,
	)
	if err != nil {
		return "", fmt.Errorf("failed to run model: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("model invocation failed with status %d: %s", resp.StatusCode, string(body))
	}

	return formatJSONResponse(body), nil
}

// RunSandbox implements RuntimeHandler.RunSandbox
// Uses sandbox.metadata.url via RunWithMetadata to route to the correct sandbox endpoint.
func (h *SDKHandler) RunSandbox(ctx context.Context, name, bodyStr, method, path string) (string, error) {
	if h.blaxelClient == nil {
		return "", fmt.Errorf("blaxel client not initialized")
	}

	var bodyData interface{}
	if err := json.Unmarshal([]byte(bodyStr), &bodyData); err != nil {
		bodyData = json.RawMessage(bodyStr)
	}

	// Use RunWithMetadata which fetches sandbox.metadata.url and uses it as the base URL.
	// This ensures the request goes directly to the sandbox's assigned endpoint.
	resp, err := h.blaxelClient.RunWithMetadata(
		ctx,
		h.cfg.Workspace,
		"sandbox",
		name,
		method,
		path,
		bodyData,
	)
	if err != nil {
		return "", fmt.Errorf("failed to execute in sandbox: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("sandbox execution failed with status %d: %s", resp.StatusCode, string(body))
	}

	return formatJSONResponse(body), nil
}

// IsReadOnly implements RuntimeHandlerWithReadOnly.IsReadOnly
func (h *SDKHandler) IsReadOnly() bool {
	return h.readOnly
}

// formatJSONResponse tries to pretty-print JSON, falls back to raw string
func formatJSONResponse(data []byte) string {
	var result interface{}
	if err := json.Unmarshal(data, &result); err == nil {
		formatted, _ := json.MarshalIndent(result, "", "  ")
		return string(formatted)
	}
	return string(data)
}
