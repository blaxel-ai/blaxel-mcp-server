package modelapis

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ModelAPIHandler defines the interface for model API operations
type ModelAPIHandler interface {
	ListModelAPIs(ctx context.Context, filter string) ([]byte, error)
	GetModelAPI(ctx context.Context, name string) ([]byte, error)
	CreateModelAPI(ctx context.Context, name, model, endpoint, integrationConnectionName, provider, apiKey, waitForCompletion string, config map[string]interface{}) ([]byte, error)
	DeleteModelAPI(ctx context.Context, name, waitForCompletion string) ([]byte, error)
}

// ModelAPIHandlerWithReadOnly extends ModelAPIHandler with readonly capability
type ModelAPIHandlerWithReadOnly interface {
	ModelAPIHandler
	IsReadOnly() bool
}

// RegisterModelAPITools registers model API tools with the given handler
func RegisterModelAPITools(s *server.MCPServer, handler ModelAPIHandler) {
	// Check if handler supports readonly mode
	readOnlyHandler, hasReadOnly := handler.(ModelAPIHandlerWithReadOnly)
	isReadOnly := hasReadOnly && readOnlyHandler.IsReadOnly()

	// List model APIs tool
	listModelAPIsTool := mcp.NewTool("list_model_apis",
		mcp.WithDescription("List all model APIs in the workspace"),
		mcp.WithString("filter",
			mcp.Description("Optional filter string"),
		),
	)

	s.AddTool(listModelAPIsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filter := request.GetString("filter", "")

		result, err := handler.ListModelAPIs(ctx, filter)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Get model API tool
	getModelAPITool := mcp.NewTool("get_model_api",
		mcp.WithDescription("Get details of a specific model API"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the model API"),
		),
	)

	s.AddTool(getModelAPITool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := request.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("model API name is required"), nil
		}

		result, err := handler.GetModelAPI(ctx, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Only register write operations if not in read-only mode
	if !isReadOnly {
		// Create model API tool
		createModelAPITool := mcp.NewTool("create_model_api",
			mcp.WithDescription("Create a model API with flexible integration options"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name for the model API"),
			),
			mcp.WithString("integrationConnectionName",
				mcp.Description("Existing integration to use"),
			),
			mcp.WithString("provider",
				mcp.Description("Provider for new integration (e.g., openai)"),
			),
			mcp.WithString("apiKey",
				mcp.Description("API key for new integration"),
			),
			mcp.WithString("model",
				mcp.Description("Model identifier"),
			),
			mcp.WithString("endpoint",
				mcp.Description("Optional endpoint URL"),
			),
			mcp.WithObject("config",
				mcp.Description("Additional configuration"),
			),
			mcp.WithString("waitForCompletion",
				mcp.Description("Whether to wait for the model API to reach a final status (true/false, default: true)"),
			),
		)

		s.AddTool(createModelAPITool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := request.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("model API name is required"), nil
			}

			model := request.GetString("model", "")
			endpoint := request.GetString("endpoint", "")
			integrationConnectionName := request.GetString("integrationConnectionName", "")
			provider := request.GetString("provider", "")
			apiKey := request.GetString("apiKey", "")
			waitForCompletion := request.GetString("waitForCompletion", "true")

			// Handle config object
			var config map[string]interface{}
			if configStr := request.GetString("config", ""); configStr != "" {
				// TODO: Parse config string to map if needed
				config = make(map[string]interface{})
			}

			result, err := handler.CreateModelAPI(ctx, name, model, endpoint, integrationConnectionName, provider, apiKey, waitForCompletion, config)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})

		// Delete model API tool
		deleteModelAPITool := mcp.NewTool("delete_model_api",
			mcp.WithDescription("Delete a model API by name"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the model API to delete"),
			),
			mcp.WithString("waitForCompletion",
				mcp.Description("Whether to wait for the model API to be fully deleted (true/false, default: true)"),
			),
		)

		s.AddTool(deleteModelAPITool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := request.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("model API name is required"), nil
			}

			waitForCompletion := request.GetString("waitForCompletion", "true")

			result, err := handler.DeleteModelAPI(ctx, name, waitForCompletion)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})
	}
}
