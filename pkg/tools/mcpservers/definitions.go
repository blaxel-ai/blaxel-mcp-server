package mcpservers

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MCPServerHandler defines the interface for MCP server operations
type MCPServerHandler interface {
	ListMCPServers(ctx context.Context, filter string) ([]byte, error)
	GetMCPServer(ctx context.Context, name string) ([]byte, error)
	CreateMCPServer(ctx context.Context, name, integrationConnectionName, integrationType, waitForCompletion string, secret, config map[string]string) ([]byte, error)
	DeleteMCPServer(ctx context.Context, name, waitForCompletion string) ([]byte, error)
}

// MCPServerHandlerWithReadOnly extends MCPServerHandler with readonly capability
type MCPServerHandlerWithReadOnly interface {
	MCPServerHandler
	IsReadOnly() bool
}

// RegisterMCPServerTools registers MCP server tools with the given handler
func RegisterMCPServerTools(s *server.MCPServer, handler MCPServerHandler) {
	// Check if handler supports readonly mode
	readOnlyHandler, hasReadOnly := handler.(MCPServerHandlerWithReadOnly)
	isReadOnly := hasReadOnly && readOnlyHandler.IsReadOnly()

	// List MCP servers tool
	listMCPServersTool := mcp.NewTool("list_mcp_servers",
		mcp.WithDescription("List all MCP servers (functions) in the workspace"),
		mcp.WithString("filter",
			mcp.Description("Optional filter string"),
		),
	)

	s.AddTool(listMCPServersTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filter := request.GetString("filter", "")

		result, err := handler.ListMCPServers(ctx, filter)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Get MCP server tool
	getMCPServerTool := mcp.NewTool("get_mcp_server",
		mcp.WithDescription("Get details of a specific MCP server (function)"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the MCP server"),
		),
	)

	s.AddTool(getMCPServerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := request.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("MCP server name is required"), nil
		}

		result, err := handler.GetMCPServer(ctx, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Only register write operations if not in read-only mode
	if !isReadOnly {
		// Create MCP server tool
		createMCPServerTool := mcp.NewTool("create_mcp_server",
			mcp.WithDescription("Create an MCP server (function) with flexible integration options"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name for the MCP server"),
			),
			mcp.WithString("integrationConnectionName",
				mcp.Description("Existing integration to use"),
			),
			mcp.WithString("integrationType",
				mcp.Description("Type for new integration (e.g., github)"),
			),
			mcp.WithObject("secret",
				mcp.Description("Secrets for new integration"),
			),
			mcp.WithObject("config",
				mcp.Description("Config for new integration"),
			),
			mcp.WithString("waitForCompletion",
				mcp.Description("Whether to wait for the MCP server to reach a final status (true/false, default: true)"),
			),
		)

		s.AddTool(createMCPServerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Use the original approach of binding to a struct for complex parameters
			type CreateMCPServerArgs struct {
				Name                      string                 `json:"name"`
				IntegrationConnectionName string                 `json:"integrationConnectionName,omitempty"`
				IntegrationType           string                 `json:"integrationType,omitempty"`
				Secret                    map[string]interface{} `json:"secret,omitempty"`
				Config                    map[string]interface{} `json:"config,omitempty"`
				WaitForCompletion         string                 `json:"waitForCompletion,omitempty"`
			}

			var args CreateMCPServerArgs
			if err := request.BindArguments(&args); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if args.Name == "" {
				return mcp.NewToolResultError("MCP server name is required"), nil
			}

			// Convert interface{} maps to string maps
			secret := make(map[string]string)
			if args.Secret != nil {
				for k, v := range args.Secret {
					if strVal, ok := v.(string); ok {
						secret[k] = strVal
					}
				}
			}

			config := make(map[string]string)
			if args.Config != nil {
				for k, v := range args.Config {
					if strVal, ok := v.(string); ok {
						config[k] = strVal
					}
				}
			}

			result, err := handler.CreateMCPServer(ctx, args.Name, args.IntegrationConnectionName, args.IntegrationType, args.WaitForCompletion, secret, config)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})

		// Delete MCP server tool
		deleteMCPServerTool := mcp.NewTool("delete_mcp_server",
			mcp.WithDescription("Delete an MCP server (function) by name"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the MCP server to delete"),
			),
			mcp.WithString("waitForCompletion",
				mcp.Description("Whether to wait for the MCP server to be fully deleted (true/false, default: true)"),
			),
		)

		s.AddTool(deleteMCPServerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := request.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("MCP server name is required"), nil
			}

			waitForCompletion := request.GetString("waitForCompletion", "true")

			result, err := handler.DeleteMCPServer(ctx, name, waitForCompletion)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})
	}
}
