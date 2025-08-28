package sandboxes

import (
	"context"
	"strconv"

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

// RegisterSandboxTools registers sandbox tools with the given handler
func RegisterSandboxTools(s *server.MCPServer, handler SandboxHandler) {
	// Check if handler supports readonly mode
	readOnlyHandler, hasReadOnly := handler.(SandboxHandlerWithReadOnly)
	isReadOnly := hasReadOnly && readOnlyHandler.IsReadOnly()

	// List sandboxes tool
	listSandboxesTool := mcp.NewTool("list_sandboxes",
		mcp.WithDescription("List all sandboxes in the workspace"),
		mcp.WithString("filter",
			mcp.Description("Optional filter string"),
		),
	)

	s.AddTool(listSandboxesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filter := request.GetString("filter", "")

		result, err := handler.ListSandboxes(ctx, filter)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Get sandbox tool
	getSandboxTool := mcp.NewTool("get_sandbox",
		mcp.WithDescription("Get details of a specific sandbox"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the sandbox to retrieve"),
		),
	)

	s.AddTool(getSandboxTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := request.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("sandbox name is required"), nil
		}

		result, err := handler.GetSandbox(ctx, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Only register write operations if not in read-only mode
	if !isReadOnly {
		// Create sandbox tool
		createSandboxTool := mcp.NewTool("create_sandbox",
			mcp.WithDescription("Create a new sandbox"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name for the sandbox"),
			),
			mcp.WithString("image",
				mcp.Description("Docker image to use for the sandbox"),
			),
			mcp.WithNumber("memory",
				mcp.Description("Memory in MB (default: 512)"),
			),
			mcp.WithString("ports", mcp.Description("Ports to expose from the sandbox, separated by commas (eg. 8080,8081)")),
			mcp.WithString("env", mcp.Description("Environment variables to set in the sandbox, separated by commas (eg. FOO=bar,BAR=baz)")),
		)

		s.AddTool(createSandboxTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Extract parameters from request
			name := request.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("sandbox name is required"), nil
			}

			image := request.GetString("image", "")
			ports := request.GetString("ports", "")
			env := request.GetString("env", "")

			// Handle memory parameter - try to get it as a number, default to 0
			memory := 0.0
			if memoryStr := request.GetString("memory", ""); memoryStr != "" {
				if mem, err := strconv.ParseFloat(memoryStr, 64); err == nil {
					memory = mem
				}
			}

			result, err := handler.CreateSandbox(ctx, name, image, memory, ports, env)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})

		// Delete sandbox tool
		deleteSandboxTool := mcp.NewTool("delete_sandbox",
			mcp.WithDescription("Delete a sandbox by name"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the sandbox to delete"),
			),
		)

		s.AddTool(deleteSandboxTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := request.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("sandbox name is required"), nil
			}

			result, err := handler.DeleteSandbox(ctx, name)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})
	}
}
