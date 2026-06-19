package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
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

type runSandboxCommandArguments struct {
	Name              string            `json:"name"`
	Command           string            `json:"command"`
	ProcessName       string            `json:"processName,omitempty"`
	WorkingDir        string            `json:"workingDir,omitempty"`
	Env               map[string]string `json:"env,omitempty"`
	WaitForCompletion bool              `json:"waitForCompletion,omitempty"`
	Timeout           int               `json:"timeout,omitempty"`
	WaitForPorts      []int             `json:"waitForPorts,omitempty"`
	RestartOnFailure  bool              `json:"restartOnFailure,omitempty"`
	MaxRestarts       int               `json:"maxRestarts,omitempty"`
	KeepAlive         bool              `json:"keepAlive,omitempty"`
	Workspace         string            `json:"workspace,omitempty"`
}

func buildSandboxCommandBody(args runSandboxCommandArguments) map[string]interface{} {
	body := map[string]interface{}{
		"command":           args.Command,
		"waitForCompletion": args.WaitForCompletion,
		"restartOnFailure":  args.RestartOnFailure,
		"keepAlive":         args.KeepAlive,
	}
	if args.ProcessName != "" {
		body["name"] = args.ProcessName
	}
	if args.WorkingDir != "" {
		body["workingDir"] = args.WorkingDir
	}
	if len(args.Env) > 0 {
		body["env"] = args.Env
	}
	if args.Timeout > 0 {
		body["timeout"] = args.Timeout
	}
	if len(args.WaitForPorts) > 0 {
		body["waitForPorts"] = args.WaitForPorts
	}
	if args.MaxRestarts > 0 {
		body["maxRestarts"] = args.MaxRestarts
	}
	return body
}

// RegisterRuntimeTools registers runtime tools with the given handler
func RegisterRuntimeTools(s *server.MCPServer, handler RuntimeHandler, cfg *config.Config) {

	// Run/Chat with Agent
	runAgentTool := mcp.NewTool("run_agent",
		mcp.WithTitleAnnotation("Run Agent"),
		mcp.WithDescription("Chat with or invoke an agent"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
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
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(runAgentTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
		}
		name := request.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("agent name is required"), nil
		}

		message := request.GetString("message", "")
		if message == "" {
			return mcp.NewToolResultError("message is required"), nil
		}

		agentContext := request.GetString("context", "")

		result, err := activeHandler.RunAgent(ctx, name, message, agentContext)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	})

	// Trigger/Run Job
	runJobTool := mcp.NewTool("run_job",
		mcp.WithTitleAnnotation("Run Job"),
		mcp.WithDescription("Trigger or run a job"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the job to run"),
		),
		mcp.WithString("parameters",
			mcp.Description("Optional parameters for the job (JSON string)"),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(runJobTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
		}
		name := request.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("job name is required"), nil
		}

		parameters := request.GetString("parameters", "")

		result, err := activeHandler.RunJob(ctx, name, parameters)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	})

	// Invoke/Run Model
	runModelTool := mcp.NewTool("run_model",
		mcp.WithTitleAnnotation("Run Model"),
		mcp.WithDescription("Invoke a Blaxel Model API with a POST request. The request body should follow the target model endpoint schema; see https://docs.blaxel.ai/Models/Query-a-model."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the model API to invoke"),
		),
		mcp.WithString("body",
			mcp.Required(),
			mcp.Description("JSON request body for the Blaxel Model API endpoint; see https://docs.blaxel.ai/Models/Query-a-model."),
		),
		mcp.WithString("path",
			mcp.Description("Blaxel Model API path to invoke, such as /v1/chat/completions; see https://docs.blaxel.ai/Models/Query-a-model."),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(runModelTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
		}
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

		result, err := activeHandler.RunModel(ctx, name, body, path, "POST")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	})

	// Execute a command in a Sandbox process.
	runSandboxCommandTool := mcp.NewTool("run_sandbox_command",
		mcp.WithTitleAnnotation("Run Sandbox Command"),
		mcp.WithDescription("Execute a command in a Blaxel sandbox by creating a sandbox process with POST /process. See https://docs.blaxel.ai/Sandboxes/Processes."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the sandbox to use"),
		),
		mcp.WithString("command",
			mcp.Required(),
			mcp.Description("Command to execute in the sandbox process"),
		),
		mcp.WithString("processName",
			mcp.Description("Optional name to assign to the sandbox process"),
		),
		mcp.WithString("workingDir",
			mcp.Description("Optional working directory for the command"),
		),
		mcp.WithObject("env",
			mcp.Description("Optional environment variables as string key-value pairs"),
			mcp.AdditionalProperties(map[string]any{"type": "string"}),
		),
		mcp.WithBoolean("waitForCompletion",
			mcp.Description("Whether to wait for the process to complete before returning"),
			mcp.DefaultBool(false),
		),
		mcp.WithNumber("timeout",
			mcp.Description("Optional timeout in seconds when waiting for completion"),
		),
		mcp.WithArray("waitForPorts",
			mcp.Description("Optional ports to wait for before returning"),
			mcp.WithNumberItems(),
		),
		mcp.WithBoolean("restartOnFailure",
			mcp.Description("Whether the process should restart if it exits unsuccessfully"),
			mcp.DefaultBool(false),
		),
		mcp.WithNumber("maxRestarts",
			mcp.Description("Optional maximum restart count when restartOnFailure is true"),
		),
		mcp.WithBoolean("keepAlive",
			mcp.Description("Whether to keep the process alive after the initial command completes"),
			mcp.DefaultBool(false),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(runSandboxCommandTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args runSandboxCommandArguments
		if err := request.BindArguments(&args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid sandbox command arguments: %v", err)), nil
		}
		if args.Name == "" {
			return mcp.NewToolResultError("sandbox name is required"), nil
		}
		if args.Command == "" {
			return mcp.NewToolResultError("command is required"), nil
		}

		body := buildSandboxCommandBody(args)

		bodyData, err := json.Marshal(body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to encode sandbox command body: %v", err)), nil
		}

		return callSandbox(ctx, handler, cfg, args.Workspace, args.Name, string(bodyData), "POST", "/process")
	})

	listSandboxProcessesTool := mcp.NewTool("list_sandbox_processes",
		mcp.WithTitleAnnotation("List Sandbox Processes"),
		mcp.WithDescription("List running and completed processes in a Blaxel sandbox with GET /process. See https://docs.blaxel.ai/Sandboxes/Processes."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the sandbox to inspect"),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(listSandboxProcessesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := request.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("sandbox name is required"), nil
		}
		return callSandbox(ctx, handler, cfg, request.GetString("workspace", ""), name, "", "GET", "/process")
	})

	getSandboxProcessTool := mcp.NewTool("get_sandbox_process",
		mcp.WithTitleAnnotation("Get Sandbox Process"),
		mcp.WithDescription("Get process status and metadata from a Blaxel sandbox with GET /process/{identifier}. See https://docs.blaxel.ai/Sandboxes/Processes."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the sandbox to inspect"),
		),
		mcp.WithString("identifier",
			mcp.Required(),
			mcp.Description("Process PID or process name"),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(getSandboxProcessTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, identifier, errResult := sandboxProcessLookupArgs(request)
		if errResult != nil {
			return errResult, nil
		}
		return callSandbox(ctx, handler, cfg, request.GetString("workspace", ""), name, "", "GET", "/process/"+url.PathEscape(identifier))
	})

	getSandboxProcessLogsTool := mcp.NewTool("get_sandbox_process_logs",
		mcp.WithTitleAnnotation("Get Sandbox Process Logs"),
		mcp.WithDescription("Get stdout and stderr logs for a Blaxel sandbox process with GET /process/{identifier}/logs. See https://docs.blaxel.ai/Sandboxes/Processes."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the sandbox to inspect"),
		),
		mcp.WithString("identifier",
			mcp.Required(),
			mcp.Description("Process PID or process name"),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(getSandboxProcessLogsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, identifier, errResult := sandboxProcessLookupArgs(request)
		if errResult != nil {
			return errResult, nil
		}
		return callSandbox(ctx, handler, cfg, request.GetString("workspace", ""), name, "", "GET", "/process/"+url.PathEscape(identifier)+"/logs")
	})

	stopSandboxProcessTool := mcp.NewTool("stop_sandbox_process",
		mcp.WithTitleAnnotation("Stop Sandbox Process"),
		mcp.WithDescription("Gracefully stop a process in a Blaxel sandbox with DELETE /process/{identifier}. See https://docs.blaxel.ai/Sandboxes/Processes."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the sandbox containing the process"),
		),
		mcp.WithString("identifier",
			mcp.Required(),
			mcp.Description("Process PID or process name"),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(stopSandboxProcessTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, identifier, errResult := sandboxProcessLookupArgs(request)
		if errResult != nil {
			return errResult, nil
		}
		return callSandbox(ctx, handler, cfg, request.GetString("workspace", ""), name, "{}", "DELETE", "/process/"+url.PathEscape(identifier))
	})

	killSandboxProcessTool := mcp.NewTool("kill_sandbox_process",
		mcp.WithTitleAnnotation("Kill Sandbox Process"),
		mcp.WithDescription("Force-kill a process in a Blaxel sandbox with DELETE /process/{identifier}/kill. See https://docs.blaxel.ai/Sandboxes/Processes."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the sandbox containing the process"),
		),
		mcp.WithString("identifier",
			mcp.Required(),
			mcp.Description("Process PID or process name"),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(killSandboxProcessTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, identifier, errResult := sandboxProcessLookupArgs(request)
		if errResult != nil {
			return errResult, nil
		}
		return callSandbox(ctx, handler, cfg, request.GetString("workspace", ""), name, "{}", "DELETE", "/process/"+url.PathEscape(identifier)+"/kill")
	})
}

func sandboxProcessLookupArgs(request mcp.CallToolRequest) (string, string, *mcp.CallToolResult) {
	name := request.GetString("name", "")
	if name == "" {
		return "", "", mcp.NewToolResultError("sandbox name is required")
	}
	identifier := request.GetString("identifier", "")
	if identifier == "" {
		return "", "", mcp.NewToolResultError("process identifier is required")
	}
	return name, identifier, nil
}

func callSandbox(ctx context.Context, handler RuntimeHandler, cfg *config.Config, workspace, name, body, method, path string) (*mcp.CallToolResult, error) {
	activeHandler, err := resolveHandler(handler, cfg, workspace)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
	}

	result, err := activeHandler.RunSandbox(ctx, name, body, method, path)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(result), nil
}
