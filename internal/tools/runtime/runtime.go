package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/client"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/logger"
	"github.com/blaxel-ai/toolkit/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Request/Response types for runtime execution

// RunAgentRequest defines the request for running/chatting with an agent
type RunAgentRequest struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Context string `json:"context,omitempty"`
}

// RunJobRequest defines the request for triggering a job
type RunJobRequest struct {
	Name       string `json:"name"`
	Parameters string `json:"parameters,omitempty"`
}

// RunModelRequest defines the request for invoking a model
type RunModelRequest struct {
	Name   string `json:"name"`
	Body   string `json:"body"`
	Path   string `json:"path,omitempty"`
	Method string `json:"method,omitempty"`
}

// RunSandboxRequest defines the request for executing code in a sandbox
type RunSandboxRequest struct {
	Name   string `json:"name"`
	Body   string `json:"body"`
	Method string `json:"method,omitempty"`
	Path   string `json:"path,omitempty"`
}

// RegisterTools registers all runtime execution tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		logger.Warnf("Failed to initialize SDK client: %v", err)
	}

	// Run/Chat with Agent
	runAgentTool := mcp.NewTool("run_agent",
		mcp.WithDescription("Chat with or invoke an agent"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the agent to run"),
		),
		mcp.WithString("message",
			mcp.Required(),
			mcp.Description("Message or prompt to send to the agent"),
		),
		mcp.WithString("context",
			mcp.Description("Optional context data for the agent (JSON string)"),
		),
	)

	s.AddTool(runAgentTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req RunAgentRequest
		if err := request.BindArguments(&req); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		if req.Name == "" {
			return mcp.NewToolResultError("agent name is required"), nil
		}
		if req.Message == "" {
			return mcp.NewToolResultError("message is required"), nil
		}

		result, err := runAgentHandler(ctx, sdkClient, cfg, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	})

	// Trigger/Run Job
	runJobTool := mcp.NewTool("run_job",
		mcp.WithDescription("Trigger or run a job"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the job to run"),
		),
		mcp.WithString("parameters",
			mcp.Description("Optional parameters for the job (JSON string)"),
		),
	)

	s.AddTool(runJobTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req RunJobRequest
		if err := request.BindArguments(&req); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		if req.Name == "" {
			return mcp.NewToolResultError("job name is required"), nil
		}

		result, err := runJobHandler(ctx, sdkClient, cfg, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	})

	// Invoke/Run Model
	runModelTool := mcp.NewTool("run_model",
		mcp.WithDescription("Invoke a model API"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the model API to invoke"),
		),
		mcp.WithString("body",
			mcp.Required(),
			mcp.Description("Body data for the model (JSON string)"),
		),
		mcp.WithString("path",
			mcp.Description("Path of the model API to invoke"),
		),
	)

	s.AddTool(runModelTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req RunModelRequest
		if err := request.BindArguments(&req); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		if req.Name == "" {
			return mcp.NewToolResultError("model name is required"), nil
		}
		if req.Body == "" {
			return mcp.NewToolResultError("body is required"), nil
		}
		if req.Path == "" {
			req.Path = "/v1/chat/completions"
		}
		if req.Method == "" {
			req.Method = "POST"
		}

		result, err := runModelHandler(ctx, sdkClient, cfg, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	})

	// Execute code in Sandbox
	runSandboxTool := mcp.NewTool("run_sandbox",
		mcp.WithDescription("Execute code in a sandbox environment"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the sandbox to use"),
		),
		mcp.WithString("body",
			mcp.Description("Body to use for the request (JSON string)"),
			mcp.DefaultString("{}"),
		),
		mcp.WithString("method",
			mcp.Description("HTTP method to use"),
			mcp.DefaultString("POST"),
		),
		mcp.WithString("path",
			mcp.Description("Path to use"),
			mcp.DefaultString("/process"),
		),
	)

	s.AddTool(runSandboxTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req RunSandboxRequest
		if err := request.BindArguments(&req); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		if req.Name == "" {
			return mcp.NewToolResultError("sandbox name is required"), nil
		}

		result, err := runSandboxHandler(ctx, sdkClient, cfg, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	})
}

// Handler implementations

func runAgentHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, cfg *config.Config, req RunAgentRequest) (string, error) {
	if sdkClient == nil {
		return "", fmt.Errorf("SDK client not initialized")
	}

	// Prepare the request body for the agent
	requestBody := map[string]interface{}{
		"inputs": req.Message,
	}
	if req.Context != "" {
		// Parse context JSON string into interface{}
		var contextData interface{}
		if err := json.Unmarshal([]byte(req.Context), &contextData); err == nil {
			requestBody["context"] = contextData
		}
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use the SDK Run method to invoke the agent
	resp, err := sdkClient.Run(
		ctx,
		cfg.Workspace,
		"agent",
		req.Name,
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

func runJobHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, cfg *config.Config, req RunJobRequest) (string, error) {
	if sdkClient == nil {
		return "", fmt.Errorf("SDK client not initialized")
	}

	// Prepare the request body for the job
	var bodyBytes []byte
	if req.Parameters != "" {
		// Use the parameters JSON string directly
		bodyBytes = []byte(req.Parameters)
	} else {
		bodyBytes = []byte("{}")
	}

	// Use the SDK Run method to trigger the job
	resp, err := sdkClient.Run(
		ctx,
		cfg.Workspace,
		"job",
		req.Name,
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

func runModelHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, cfg *config.Config, req RunModelRequest) (string, error) {
	if sdkClient == nil {
		return "", fmt.Errorf("SDK client not initialized")
	}

	// Prepare the request body for the model
	// Use the body JSON string directly
	bodyBytes := []byte(req.Body)

	// Use the SDK Run method to invoke the model
	resp, err := sdkClient.Run(
		ctx,
		cfg.Workspace,
		"model",
		req.Name,
		req.Method,
		req.Path,
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("model invocation failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Try to format as JSON for better readability
	var result interface{}
	if err := json.Unmarshal(body, &result); err == nil {
		formatted, _ := json.MarshalIndent(result, "", "  ")
		return string(formatted), nil
	}

	return string(body), nil
}

func runSandboxHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, cfg *config.Config, req RunSandboxRequest) (string, error) {
	if sdkClient == nil {
		return "", fmt.Errorf("SDK client not initialized")
	}

	// Prepare the request body for the sandbox
	// Use the body JSON string directly
	bodyBytes := []byte(req.Body)

	// First, ensure the sandbox is started
	startResp, err := sdkClient.StartSandboxWithResponse(ctx, req.Name)
	if err != nil {
		return "", fmt.Errorf("failed to start sandbox: %w", err)
	}
	if startResp.StatusCode() != http.StatusOK && startResp.StatusCode() != http.StatusConflict {
		return "", fmt.Errorf("failed to start sandbox with status %d", startResp.StatusCode())
	}

	// Use the SDK Run method to execute code in the sandbox
	resp, err := sdkClient.Run(
		ctx,
		cfg.Workspace,
		"sandbox",
		req.Name,
		req.Method,
		req.Path,
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("sandbox execution failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Try to format as JSON for better readability
	var result interface{}
	if err := json.Unmarshal(body, &result); err == nil {
		formatted, _ := json.MarshalIndent(result, "", "  ")
		return string(formatted), nil
	}

	return string(body), nil
}
