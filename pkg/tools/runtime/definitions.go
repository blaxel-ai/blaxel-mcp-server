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
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
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
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
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
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
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
		mcp.WithDescription(`Execute commands and manage processes in a Blaxel sandbox environment.

Endpoints:
- POST /process - Execute a command. Body fields: "command" (required), "name", "workingDir", "env" (object), "waitForCompletion" (bool), "timeout" (seconds), "waitForPorts" (int array), "restartOnFailure" (bool), "maxRestarts" (int), "keepAlive" (bool).
- GET /process - List all processes.
- GET /process/{identifier} - Get process info by PID or name.
- GET /process/{identifier}/logs - Get process stdout/stderr logs.
- DELETE /process/{identifier} - Gracefully stop a process.
- DELETE /process/{identifier}/kill - Force kill a process.

Response includes: pid, name, command, status (running/completed/failed/killed/stopped), exitCode, stdout, stderr, logs.

Examples:
- Run a command: method="POST", path="/process", body="{\"command\": \"ls -la\", \"waitForCompletion\": true}"
- Start a server: method="POST", path="/process", body="{\"command\": \"npm start\", \"name\": \"web\", \"waitForPorts\": [3000]}"
- Get logs: method="GET", path="/process/web/logs", body="{}"
- Stop process: method="DELETE", path="/process/web", body="{}"`),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the sandbox to use"),
		),
		mcp.WithString("body",
			mcp.Description("JSON string body for the request. For process execution, must include at least a \"command\" field. See tool description for full schema and examples."),
			mcp.DefaultString("{}"),
		),
		mcp.WithString("method",
			mcp.Description("HTTP method: GET to list/read processes and logs, POST to execute commands, DELETE to stop/kill processes"),
			mcp.DefaultString("POST"),
		),
		mcp.WithString("path",
			mcp.Description("API path. Common paths: /process (execute or list), /process/{identifier} (get or stop), /process/{identifier}/logs (get logs), /process/{identifier}/kill (force kill)"),
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
