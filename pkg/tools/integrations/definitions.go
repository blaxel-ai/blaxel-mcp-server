package integrations

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// IntegrationHandler defines the interface for integration operations
type IntegrationHandler interface {
	ListIntegrations(ctx context.Context, filter string) ([]byte, error)
	GetIntegration(ctx context.Context, name string) ([]byte, error)
	CreateIntegration(ctx context.Context, name, integrationType string, secret, config map[string]string) ([]byte, error)
	DeleteIntegration(ctx context.Context, name string) ([]byte, error)
}

// IntegrationHandlerWithReadOnly extends IntegrationHandler with readonly capability
type IntegrationHandlerWithReadOnly interface {
	IntegrationHandler
	IsReadOnly() bool
}

// RegisterIntegrationTools registers integration tools with the given handler
func RegisterIntegrationTools(s *server.MCPServer, handler IntegrationHandler) {
	// Check if handler supports readonly mode
	readOnlyHandler, hasReadOnly := handler.(IntegrationHandlerWithReadOnly)
	isReadOnly := hasReadOnly && readOnlyHandler.IsReadOnly()

	// List integrations tool
	listIntegrationsTool := mcp.NewTool("list_integrations",
		mcp.WithDescription("List all integration connections in the workspace"),
		mcp.WithString("filter",
			mcp.Description("Optional filter string"),
		),
	)

	s.AddTool(listIntegrationsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filter := request.GetString("filter", "")

		result, err := handler.ListIntegrations(ctx, filter)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Get integration tool
	getIntegrationTool := mcp.NewTool("get_integration",
		mcp.WithDescription("Get details of a specific integration connection"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the integration"),
		),
	)

	s.AddTool(getIntegrationTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := request.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("integration name is required"), nil
		}

		result, err := handler.GetIntegration(ctx, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Only register write operations if not in read-only mode
	if !isReadOnly {
		// Create integration tool
		createIntegrationTool := mcp.NewTool("create_integration",
			mcp.WithDescription("Create a new integration connection"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name for the integration connection"),
			),
			mcp.WithString("integrationType",
				mcp.Required(),
				mcp.Description("Type of integration (e.g., github, slack, etc.)"),
			),
			mcp.WithObject("secret",
				mcp.Description("Secret credentials for the integration"),
			),
			mcp.WithObject("config",
				mcp.Description("Configuration parameters for the integration"),
			),
		)

		s.AddTool(createIntegrationTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Use the original approach of binding to a struct for complex parameters
			type CreateIntegrationArgs struct {
				Name            string                 `json:"name"`
				IntegrationType string                 `json:"integrationType"`
				Secret          map[string]interface{} `json:"secret,omitempty"`
				Config          map[string]interface{} `json:"config,omitempty"`
			}

			var args CreateIntegrationArgs
			if err := request.BindArguments(&args); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if args.Name == "" {
				return mcp.NewToolResultError("integration name is required"), nil
			}

			if args.IntegrationType == "" {
				return mcp.NewToolResultError("integrationType is required"), nil
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

			result, err := handler.CreateIntegration(ctx, args.Name, args.IntegrationType, secret, config)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})

		// Delete integration tool
		deleteIntegrationTool := mcp.NewTool("delete_integration",
			mcp.WithDescription("Delete an integration connection by name"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the integration to delete"),
			),
		)

		s.AddTool(deleteIntegrationTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := request.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("integration name is required"), nil
			}

			result, err := handler.DeleteIntegration(ctx, name)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})
	}
}
