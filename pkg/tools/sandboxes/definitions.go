package sandboxes

import (
	"context"
	"fmt"
	"strings"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// SandboxHandler defines the interface for sandbox operations
type SandboxHandler interface {
	ListSandboxes(ctx context.Context, filter string) ([]byte, error)
	GetSandbox(ctx context.Context, name string) ([]byte, error)
	CreateSandbox(ctx context.Context, name, image string, memory float64, ports, env string) ([]byte, error)
	DeleteSandbox(ctx context.Context, name string) ([]byte, error)
}

// RegionalSandboxHandler adds explicit placement without breaking legacy handlers.
type RegionalSandboxHandler interface {
	CreateSandboxInRegion(ctx context.Context, name, image string, memory float64, ports, env, region string) ([]byte, error)
}

// ParseRegionArgument rejects malformed explicit placement instead of using a default.
func ParseRegionArgument(args map[string]any) (string, error) {
	raw, exists := args["region"]
	if !exists {
		return "", nil
	}
	region, ok := raw.(string)
	if !ok || region == "" || strings.TrimSpace(region) != region {
		return "", fmt.Errorf("region must be a non-empty region identifier")
	}
	return region, nil
}

// SandboxHandlerWithReadOnly extends SandboxHandler with readonly capability
type SandboxHandlerWithReadOnly interface {
	SandboxHandler
	IsReadOnly() bool
}

// RegisterSandboxTools registers sandbox tools with the given handler
func RegisterSandboxTools(s *server.MCPServer, handler SandboxHandler, cfg *config.Config) {
	// Check if handler supports readonly mode
	readOnlyHandler, hasReadOnly := handler.(SandboxHandlerWithReadOnly)
	isReadOnly := hasReadOnly && readOnlyHandler.IsReadOnly()

	// List sandboxes tool
	listSandboxesTool := mcp.NewTool("list_sandboxes",
		tools.OutputSchema(ListSandboxesOutputSchema()),
		mcp.WithToolTitle("List Sandboxes"),
		mcp.WithTitleAnnotation("List Sandboxes"),
		mcp.WithDescription("List the Blaxel sandboxes in the workspace with each sandbox's status, image and resources. Pass filter to keep only sandboxes whose name contains a case-insensitive substring. Use this to find the sandbox name the run_sandbox_command and process tools need. See https://docs.blaxel.ai/Sandboxes/Overview."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithString("filter",
			mcp.Description("Optional filter string"),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(listSandboxesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
		}
		filter := request.GetString("filter", "")

		result, err := activeHandler.ListSandboxes(ctx, filter)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return tools.StructuredResult(result), nil
	})

	// Get sandbox tool
	getSandboxTool := mcp.NewTool("get_sandbox",
		tools.OutputSchema(getSandboxOutputSchema()),
		mcp.WithToolTitle("Get Sandbox"),
		mcp.WithTitleAnnotation("Get Sandbox"),
		mcp.WithDescription("Get one Blaxel sandbox by name, returning its full definition: image, resources, exposed ports, volumes, expiration and current status. Call this to confirm a sandbox is running before executing commands in it. See https://docs.blaxel.ai/Sandboxes/Overview."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the sandbox to retrieve"),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(getSandboxTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
		}
		name := request.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("sandbox name is required"), nil
		}

		result, err := activeHandler.GetSandbox(ctx, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return tools.StructuredResult(result), nil
	})

	// Only register write operations if not in read-only mode
	if !isReadOnly {
		// Create sandbox tool
		createSandboxTool := mcp.NewTool("create_sandbox", createSandboxToolOptions()...)

		s.AddTool(createSandboxTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			opts, err := ParseCreateSandboxArguments(request.GetArguments())
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
			}

			result, err := createSandbox(ctx, activeHandler, opts)
			if err != nil {
				return mcp.NewToolResultError(withCreateSandboxErrorHint(err.Error(), opts)), nil
			}

			return tools.StructuredResult(result), nil
		})

		// Delete sandbox tool
		deleteSandboxTool := mcp.NewTool("delete_sandbox",
			tools.OutputSchema(deleteSandboxOutputSchema()),
			mcp.WithToolTitle("Delete Sandbox"),
			mcp.WithTitleAnnotation("Delete Sandbox"),
			mcp.WithDescription("Permanently delete a Blaxel sandbox by name, discarding its filesystem and any processes still running in it. See https://docs.blaxel.ai/Sandboxes/Overview."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the sandbox to delete"),
			),
			mcp.WithString("workspace",
				mcp.Description("Optional workspace name to override the default workspace"),
			),
		)

		s.AddTool(deleteSandboxTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
			}
			name := request.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("sandbox name is required"), nil
			}

			result, err := activeHandler.DeleteSandbox(ctx, name)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return tools.StructuredResult(result), nil
		})
	}
}

const createSandboxDescription = "Create a Blaxel sandbox, an isolated VM you run commands in with run_sandbox_command. " +
	"An idle sandbox goes to standby (scale to zero) and resumes on the next request. " +
	"Without ttl or expirationPolicies it is kept until you call delete_sandbox (some plan tiers cap its lifetime), so set one for temporary work. " +
	"ttl and ttl-max-age must be at least 5m and ttl-idle at least 60s; shorter values fail with HTTP 400. To remove a sandbox sooner, delete it. " +
	"Set createIfNotExist to get the live sandbox with this name back instead of a conflict error. " +
	"See https://docs.blaxel.ai/Sandboxes/Overview and https://docs.blaxel.ai/Sandboxes/Expiration."

// createSandboxToolOptions declares create_sandbox. Parameter names follow the
// create-sandbox API fields they set; TestCreateSandboxCoversAPIFields keeps
// them in step with the API.
func createSandboxToolOptions() []mcp.ToolOption {
	expirationPolicy := map[string]any{
		"type":     "object",
		"required": []string{"type", "value"},
		"properties": map[string]any{
			"type": map[string]any{
				"type":        "string",
				"enum":        []string{ExpirationPolicyTTLIdle, ExpirationPolicyTTLMaxAge, ExpirationPolicyDate},
				"description": "ttl-idle: delete after value without activity (minimum 60s). ttl-max-age: delete value after creation (minimum 5m). date: delete at value.",
			},
			"value": map[string]any{
				"type":        "string",
				"description": "Duration for ttl-idle and ttl-max-age (units s, m, h, d or w, for example 30m, 24h or 7d), or a future ISO 8601 time for date, for example 2030-01-01T00:00:00Z",
			},
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{ExpirationActionDelete},
				"default":     ExpirationActionDelete,
				"description": "What happens when the rule is reached. Only delete is supported.",
			},
		},
		"additionalProperties": false,
	}
	volume := map[string]any{
		"type":     "object",
		"required": []string{"name", "mountPath"},
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Name of an existing volume in the same workspace and region, or a name for a new ephemeral volume",
			},
			"mountPath": map[string]any{
				"type":        "string",
				"description": "Absolute path to mount the volume at, for example /data. It replaces what the image has at that path.",
			},
			"readOnly": map[string]any{
				"type":        "boolean",
				"description": "Mount the volume read-only",
			},
			"type": map[string]any{
				"type":        "string",
				"enum":        []string{VolumeTypePersistent, VolumeTypeEphemeral},
				"default":     VolumeTypePersistent,
				"description": "persistent attaches an existing volume whose data outlives the sandbox. ephemeral creates scratch disk that is deleted with the sandbox.",
			},
			"sizeMb": map[string]any{
				"type":        "integer",
				"minimum":     1,
				"description": "Size in MB. Required for ephemeral volumes, ignored for persistent ones.",
			},
		},
		"additionalProperties": false,
	}

	return []mcp.ToolOption{
		tools.OutputSchema(createSandboxOutputSchema()),
		mcp.WithToolTitle("Create Sandbox"),
		mcp.WithTitleAnnotation("Create Sandbox"),
		mcp.WithDescription(createSandboxDescription),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name for the sandbox"),
		),
		mcp.WithString("image",
			mcp.Description("Docker image to use for the sandbox"),
		),
		mcp.WithInteger("memory",
			mcp.Description("Memory in MB (minimum: 1024, maximum: 262144, default: 1024)"),
			mcp.Min(1024),
			mcp.Max(262144),
			mcp.DefaultNumber(1024),
		),
		mcp.WithString("region", mcp.Description("Optional deployment region, for example us-was-1. Omit to use the workspace default.")),
		mcp.WithString("ports", mcp.Description("Ports to expose from the sandbox, separated by commas (eg. 8080,8081)")),
		mcp.WithString("env", mcp.Description("Environment variables to set in the sandbox, separated by commas (eg. FOO=bar,BAR=baz)")),
		mcp.WithObject("labels",
			mcp.Description("Labels to set on the sandbox as string key-value pairs, for example {\"task\": \"build\"}"),
			mcp.AdditionalProperties(map[string]any{"type": "string"}),
		),
		mcp.WithString("ttl",
			mcp.Description("Delete the sandbox this long after creation, even while in use. Units s, m, h, d or w, for example 30m, 24h or 7d; minimum 5m. To delete after inactivity instead, use expirationPolicies with type ttl-idle."),
		),
		mcp.WithArray("expirationPolicies",
			mcp.Description("Rules that delete the sandbox automatically; the first one reached applies. Use each type at most once."),
			mcp.Items(expirationPolicy),
		),
		mcp.WithString("terminatedRetention",
			mcp.Description("How long to keep the record of the sandbox after it is deleted, for reading its logs, for example 1h or 24h. Default 5m."),
		),
		mcp.WithArray("volumes",
			mcp.Description("Volumes to mount, set at creation only. A sandbox takes at most one persistent volume, and a persistent volume can be attached to one sandbox at a time. Ephemeral volumes have no such limit."),
			mcp.Items(volume),
		),
		mcp.WithInteger("standbyGracePeriodSeconds",
			mcp.Description("Seconds the sandbox gets after a SIGUSR2 signal before it goes to standby, 0 to 300. Default 0: no signal."),
			mcp.Min(0),
		),
		mcp.WithInteger("terminationGracePeriodSeconds",
			mcp.Description("Seconds the sandbox gets after a SIGTERM signal before it is deleted, 0 to 300. Default 0: no signal."),
			mcp.Min(0),
		),
		mcp.WithBoolean("createIfNotExist",
			mcp.Description("If a live sandbox with this name exists, return it unchanged instead of failing with a conflict. The settings in this call are then not applied."),
			mcp.DefaultBool(false),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	}
}
