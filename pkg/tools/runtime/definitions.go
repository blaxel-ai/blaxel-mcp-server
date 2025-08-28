package runtime

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RuntimeHandler defines the interface for runtime operations
type RuntimeHandler interface {
	RunAgent(ctx context.Context, name, message, context string) (string, error)
	RunJob(ctx context.Context, name, parameters string) (string, error)
	RunModel(ctx context.Context, name, body, path, method string) (string, error)
	RunSandbox(ctx context.Context, name, body, method, path string) (string, error)
}

// RuntimeHandlerWithReadOnly extends RuntimeHandler with readonly capability
type RuntimeHandlerWithReadOnly interface {
	RuntimeHandler
	IsReadOnly() bool
}

// RegisterRuntimeTools registers runtime tools with the given handler
func RegisterRuntimeTools(s *server.MCPServer, handler RuntimeHandler) {

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
		name := request.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("agent name is required"), nil
		}

		message := request.GetString("message", "")
		if message == "" {
			return mcp.NewToolResultError("message is required"), nil
		}

		context := request.GetString("context", "")

		result, err := handler.RunAgent(ctx, name, message, context)
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
		name := request.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("job name is required"), nil
		}

		parameters := request.GetString("parameters", "")

		result, err := handler.RunJob(ctx, name, parameters)
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
		name := request.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("model name is required"), nil
		}

		body := request.GetString("body", "")
		if body == "" {
			return mcp.NewToolResultError("body is required"), nil
		}

		path := request.GetString("path", "")
		if path == "" {
			path = "/v1/chat/completions"
		}

		method := request.GetString("method", "")
		if method == "" {
			method = "POST"
		}

		result, err := handler.RunModel(ctx, name, body, path, method)
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
		name := request.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("sandbox name is required"), nil
		}

		body := request.GetString("body", "{}")
		method := request.GetString("method", "POST")
		path := request.GetString("path", "/process")

		result, err := handler.RunSandbox(ctx, name, body, method, path)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	})
}
