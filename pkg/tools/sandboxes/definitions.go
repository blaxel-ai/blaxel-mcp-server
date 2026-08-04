package sandboxes

import (
	"context"
	"fmt"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
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

// SandboxHandlerWithReadOnly extends SandboxHandler with readonly capability
type SandboxHandlerWithReadOnly interface {
	SandboxHandler
	IsReadOnly() bool
}

type createSandboxArguments struct {
	Name      string   `json:"name"`
	Image     string   `json:"image"`
	Memory    *float64 `json:"memory"`
	Ports     string   `json:"ports"`
	Env       string   `json:"env"`
	Workspace string   `json:"workspace"`
}

// RegisterSandboxTools registers sandbox tools with the given handler
func RegisterSandboxTools(s *server.MCPServer, handler SandboxHandler, cfg *config.Config) {
	// Check if handler supports readonly mode
	readOnlyHandler, hasReadOnly := handler.(SandboxHandlerWithReadOnly)
	isReadOnly := hasReadOnly && readOnlyHandler.IsReadOnly()

	// List sandboxes tool
	listSandboxesTool := mcp.NewTool("list_sandboxes",
		mcp.WithToolTitle("List Sandboxes"),
		mcp.WithTitleAnnotation("List Sandboxes"),
		mcp.WithDescription("List all sandboxes in the workspace"),
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

		return mcp.NewToolResultText(string(result)), nil
	})

	// Get sandbox tool
	getSandboxTool := mcp.NewTool("get_sandbox",
		mcp.WithToolTitle("Get Sandbox"),
		mcp.WithTitleAnnotation("Get Sandbox"),
		mcp.WithDescription("Get details of a specific sandbox"),
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

		return mcp.NewToolResultText(string(result)), nil
	})

	// Only register write operations if not in read-only mode
	if !isReadOnly {
		// Create sandbox tool
	createSandboxTool := mcp.NewTool("create_sandbox",
		mcp.WithToolTitle("Create Sandbox"),
		mcp.WithTitleAnnotation("Create Sandbox"),
		mcp.WithDescription("Create a new sandbox"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
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
			mcp.WithString("ports", mcp.Description("Ports to expose from the sandbox, separated by commas (eg. 8080,8081)")),
			mcp.WithString("env", mcp.Description("Environment variables to set in the sandbox, separated by commas (eg. FOO=bar,BAR=baz)")),
			mcp.WithString("workspace",
				mcp.Description("Optional workspace name to override the default workspace"),
			),
		)

		s.AddTool(createSandboxTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// BindArguments serializes map arguments through JSON, which rejects
			// non-finite floats before it can identify the offending field.
			if rawMemory, ok := request.GetArguments()["memory"].(float64); ok {
				if err := validateSandboxMemory(rawMemory); err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
			}

			var args createSandboxArguments
			if err := request.BindArguments(&args); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid create_sandbox arguments: %v", err)), nil
			}
			if args.Name == "" {
				return mcp.NewToolResultError("sandbox name is required"), nil
			}
			if args.Memory != nil {
				if err := validateSandboxMemory(*args.Memory); err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
			}
			if _, err := parseSandboxPorts(args.Ports); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			activeHandler, err := resolveHandler(handler, cfg, args.Workspace)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
			}
			memory := float64(0)
			if args.Memory != nil {
				memory = *args.Memory
			}
			result, err := activeHandler.CreateSandbox(ctx, args.Name, args.Image, memory, args.Ports, args.Env)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})

		// Delete sandbox tool
		deleteSandboxTool := mcp.NewTool("delete_sandbox",
			mcp.WithToolTitle("Delete Sandbox"),
			mcp.WithTitleAnnotation("Delete Sandbox"),
			mcp.WithDescription("Delete a sandbox by name"),
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

			return mcp.NewToolResultText(string(result)), nil
		})
	}
}
