package integrations

import (
	"context"
	"fmt"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
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
func RegisterIntegrationTools(s *server.MCPServer, handler IntegrationHandler, cfg *config.Config) {
	// Check if handler supports readonly mode
	readOnlyHandler, hasReadOnly := handler.(IntegrationHandlerWithReadOnly)
	isReadOnly := hasReadOnly && readOnlyHandler.IsReadOnly()

	// List integrations tool
	listIntegrationsTool := mcp.NewTool("list_integrations",
		mcp.WithDescription("List all integration connections in the workspace"),
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

	s.AddTool(listIntegrationsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
		}
		filter := request.GetString("filter", "")

		result, err := activeHandler.ListIntegrations(ctx, filter)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Get integration tool
	getIntegrationTool := mcp.NewTool("get_integration",
		mcp.WithDescription("Get details of a specific integration connection"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the integration"),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(getIntegrationTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
		}
		name := request.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("integration name is required"), nil
		}

		result, err := activeHandler.GetIntegration(ctx, name)
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
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
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
			mcp.WithString("workspace",
				mcp.Description("Optional workspace name to override the default workspace"),
			),
		)

		s.AddTool(createIntegrationTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
			}
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

			result, err := activeHandler.CreateIntegration(ctx, args.Name, args.IntegrationType, secret, config)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})

		// Delete integration tool
		deleteIntegrationTool := mcp.NewTool("delete_integration",
			mcp.WithDescription("Delete an integration connection by name"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the integration to delete"),
			),
			mcp.WithString("workspace",
				mcp.Description("Optional workspace name to override the default workspace"),
			),
		)

		s.AddTool(deleteIntegrationTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
			}
			name := request.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("integration name is required"), nil
			}

			result, err := activeHandler.DeleteIntegration(ctx, name)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})
	}
}
