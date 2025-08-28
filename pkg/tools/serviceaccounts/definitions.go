package serviceaccounts

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ServiceAccountHandler defines the interface for service account operations
type ServiceAccountHandler interface {
	ListServiceAccounts(ctx context.Context, filter string) ([]byte, error)
	GetServiceAccount(ctx context.Context, clientID string) ([]byte, error)
	CreateServiceAccount(ctx context.Context, name string) ([]byte, error)
	DeleteServiceAccount(ctx context.Context, clientID string) ([]byte, error)
	UpdateServiceAccount(ctx context.Context, clientID, description string) ([]byte, error)
}

// ServiceAccountHandlerWithReadOnly extends ServiceAccountHandler with readonly capability
type ServiceAccountHandlerWithReadOnly interface {
	ServiceAccountHandler
	IsReadOnly() bool
}

// RegisterServiceAccountTools registers service account tools with the given handler
func RegisterServiceAccountTools(s *server.MCPServer, handler ServiceAccountHandler) {
	// Check if handler supports readonly mode
	readOnlyHandler, hasReadOnly := handler.(ServiceAccountHandlerWithReadOnly)
	isReadOnly := hasReadOnly && readOnlyHandler.IsReadOnly()

	// List service accounts tool
	listServiceAccountsTool := mcp.NewTool("list_service_accounts",
		mcp.WithDescription("List all service accounts in the workspace"),
		mcp.WithString("filter",
			mcp.Description("Optional filter to match service account names"),
		),
	)

	s.AddTool(listServiceAccountsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filter := request.GetString("filter", "")

		result, err := handler.ListServiceAccounts(ctx, filter)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Get service account tool
	getServiceAccountTool := mcp.NewTool("get_service_account",
		mcp.WithDescription("Get details of a service account by client ID"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Client ID of the service account to retrieve"),
		),
	)

	s.AddTool(getServiceAccountTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID := request.GetString("name", "")
		if clientID == "" {
			return mcp.NewToolResultError("name is required"), nil
		}

		result, err := handler.GetServiceAccount(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Only register write operations if not in read-only mode
	if !isReadOnly {
		// Create service account tool
		createServiceAccountTool := mcp.NewTool("create_service_account",
			mcp.WithDescription("Create a new service account"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Display name for the service account"),
			),
		)

		s.AddTool(createServiceAccountTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := request.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("service account name is required"), nil
			}

			result, err := handler.CreateServiceAccount(ctx, name)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})

		// Delete service account tool
		deleteServiceAccountTool := mcp.NewTool("delete_service_account",
			mcp.WithDescription("Delete a service account by client ID"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Client ID of the service account to delete"),
			),
		)

		s.AddTool(deleteServiceAccountTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			clientID := request.GetString("name", "")
			if clientID == "" {
				return mcp.NewToolResultError("name is required"), nil
			}

			result, err := handler.DeleteServiceAccount(ctx, clientID)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})

		// Update service account tool
		updateServiceAccountTool := mcp.NewTool("update_service_account",
			mcp.WithDescription("Update a service account's name"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Client ID of the service account to update"),
			),
			mcp.WithString("description",
				mcp.Description("New description for the service account"),
			),
		)

		s.AddTool(updateServiceAccountTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			clientID := request.GetString("name", "")
			if clientID == "" {
				return mcp.NewToolResultError("name is required"), nil
			}

			description := request.GetString("description", "")

			result, err := handler.UpdateServiceAccount(ctx, clientID, description)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})
	}
}
