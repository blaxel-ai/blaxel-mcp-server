package serviceaccounts

import (
	"context"
	"fmt"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ServiceAccountHandler defines the interface for service account operations
type ServiceAccountHandler interface {
	ListServiceAccounts(ctx context.Context, filter string) ([]byte, error)
	GetServiceAccount(ctx context.Context, clientID string) ([]byte, error)
	CreateServiceAccount(ctx context.Context, name string) ([]byte, error)
	DeleteServiceAccount(ctx context.Context, clientID string) ([]byte, error)
	UpdateServiceAccount(ctx context.Context, clientID, name string) ([]byte, error)
}

// ServiceAccountHandlerWithReadOnly extends ServiceAccountHandler with readonly capability
type ServiceAccountHandlerWithReadOnly interface {
	ServiceAccountHandler
	IsReadOnly() bool
}

// RegisterServiceAccountTools registers service account tools with the given handler
func RegisterServiceAccountTools(s *server.MCPServer, handler ServiceAccountHandler, cfg *config.Config) {
	// Check if handler supports readonly mode
	readOnlyHandler, hasReadOnly := handler.(ServiceAccountHandlerWithReadOnly)
	isReadOnly := hasReadOnly && readOnlyHandler.IsReadOnly()

	// List service accounts tool
	listServiceAccountsTool := mcp.NewTool("list_service_accounts",
		mcp.WithToolTitle("List Service Accounts"),
		mcp.WithTitleAnnotation("List Service Accounts"),
		mcp.WithDescription("List the service accounts in the workspace with each account's client ID, name and creation time. Client secrets are never returned by this tool; a secret is shown only once, when create_service_account mints it. Requires a workspace admin or owner. See https://docs.blaxel.ai/Security/Service-accounts."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithString("filter",
			mcp.Description("Optional filter to match service account names"),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(listServiceAccountsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
		}
		filter := request.GetString("filter", "")

		result, err := activeHandler.ListServiceAccounts(ctx, filter)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Get service account tool
	getServiceAccountTool := mcp.NewTool("get_service_account",
		mcp.WithToolTitle("Get Service Account"),
		mcp.WithTitleAnnotation("Get Service Account"),
		mcp.WithDescription("Get one service account by its client ID, returning its name, description and creation time. The client secret is not returned and cannot be read back after creation. Requires a workspace admin or owner. See https://docs.blaxel.ai/Security/Service-accounts."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Client ID of the service account to retrieve"),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(getServiceAccountTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
		}
		clientID := request.GetString("name", "")
		if clientID == "" {
			return mcp.NewToolResultError("name is required"), nil
		}

		result, err := activeHandler.GetServiceAccount(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Only register write operations if not in read-only mode
	if !isReadOnly {
		// Create service account tool
		createServiceAccountTool := mcp.NewTool("create_service_account",
			mcp.WithToolTitle("Create Service Account"),
			mcp.WithTitleAnnotation("Create Service Account"),
			mcp.WithDescription("Create a service account for machine access to the workspace and return its client ID plus a one-time client secret. The secret appears only in this response and cannot be retrieved again, so record it now if it is needed. Requires a workspace admin or owner. See https://docs.blaxel.ai/Security/Service-accounts."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Display name for the service account"),
			),
			mcp.WithString("workspace",
				mcp.Description("Optional workspace name to override the default workspace"),
			),
		)

		s.AddTool(createServiceAccountTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
			}
			name := request.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("service account name is required"), nil
			}

			result, err := activeHandler.CreateServiceAccount(ctx, name)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})

		// Delete service account tool
		deleteServiceAccountTool := mcp.NewTool("delete_service_account",
			mcp.WithToolTitle("Delete Service Account"),
			mcp.WithTitleAnnotation("Delete Service Account"),
			mcp.WithDescription("Permanently delete a service account, identified by its client ID rather than its display name. Anything authenticating with its credentials immediately loses workspace access. Requires a workspace admin or owner. See https://docs.blaxel.ai/Security/Service-accounts."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Client ID of the service account to delete"),
			),
			mcp.WithString("workspace",
				mcp.Description("Optional workspace name to override the default workspace"),
			),
		)

		s.AddTool(deleteServiceAccountTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
			}
			clientID := request.GetString("name", "")
			if clientID == "" {
				return mcp.NewToolResultError("name is required"), nil
			}

			result, err := activeHandler.DeleteServiceAccount(ctx, clientID)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})

		// Update service account tool
		updateServiceAccountTool := mcp.NewTool("update_service_account",
			mcp.WithToolTitle("Update Service Account"),
			mcp.WithTitleAnnotation("Update Service Account"),
			mcp.WithDescription("Rename an existing service account, identified by its client ID. Its credentials and workspace access are unchanged. Requires a workspace admin or owner. See https://docs.blaxel.ai/Security/Service-accounts."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithString("client_id",
				mcp.Required(),
				mcp.Description("Client ID of the service account to update"),
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("New name for the service account"),
			),
			mcp.WithString("workspace",
				mcp.Description("Optional workspace name to override the default workspace"),
			),
		)

		s.AddTool(updateServiceAccountTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
			}
			clientID := request.GetString("client_id", "")
			if clientID == "" {
				return mcp.NewToolResultError("client_id is required"), nil
			}
			name := request.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("name is required"), nil
			}

			result, err := activeHandler.UpdateServiceAccount(ctx, clientID, name)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})
	}
}
