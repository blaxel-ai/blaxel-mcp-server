package modelapis

import (
	"context"
	"fmt"
	"strings"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
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
func RegisterModelAPITools(s *server.MCPServer, handler ModelAPIHandler, cfg *config.Config) {
	// Check if handler supports readonly mode
	readOnlyHandler, hasReadOnly := handler.(ModelAPIHandlerWithReadOnly)
	isReadOnly := hasReadOnly && readOnlyHandler.IsReadOnly()

	// List model APIs tool
	listModelAPIsTool := mcp.NewTool("list_model_apis",
		mcp.WithToolTitle("List Model APIs"),
		mcp.WithTitleAnnotation("List Model APIs"),
		mcp.WithDescription("List all model APIs in the workspace"),
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

	s.AddTool(listModelAPIsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
		}
		filter := request.GetString("filter", "")

		result, err := activeHandler.ListModelAPIs(ctx, filter)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Get model API tool
	getModelAPITool := mcp.NewTool("get_model_api",
		mcp.WithToolTitle("Get Model API"),
		mcp.WithTitleAnnotation("Get Model API"),
		mcp.WithDescription("Get details of a specific model API"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the model API"),
		),
		mcp.WithString("workspace",
			mcp.Description("Optional workspace name to override the default workspace"),
		),
	)

	s.AddTool(getModelAPITool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
		}
		name := request.GetString("name", "")
		if name == "" {
			return mcp.NewToolResultError("model API name is required"), nil
		}

		result, err := activeHandler.GetModelAPI(ctx, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})

	// Only register write operations if not in read-only mode
	if !isReadOnly {
		// Create model API tool
		createModelAPITool := mcp.NewTool("create_model_api",
			mcp.WithToolTitle("Create Model API"),
			mcp.WithTitleAnnotation("Create Model API"),
			mcp.WithDescription("Create a model API with flexible integration options"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
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
				mcp.Description("Optional provider endpoint name"),
			),
			mcp.WithObject("config",
				mcp.Description("Optional string configuration for a new inline integration"),
			),
			mcp.WithString("waitForCompletion", modelAPILifecycleWaitOptions(
				cfg,
				"Async-only calls do not wait. Use false or omit this field, then poll get_model_api until status is DEPLOYED or FAILED.",
				"Whether to wait for the model API to reach DEPLOYED or FAILED (true/false, default: true).",
			)...),
			mcp.WithString("workspace",
				mcp.Description("Optional workspace name to override the default workspace"),
			),
		)

		s.AddTool(createModelAPITool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			type createModelAPIArguments struct {
				Name                      string                 `json:"name"`
				Model                     string                 `json:"model,omitempty"`
				Endpoint                  string                 `json:"endpoint,omitempty"`
				IntegrationConnectionName string                 `json:"integrationConnectionName,omitempty"`
				Provider                  string                 `json:"provider,omitempty"`
				APIKey                    string                 `json:"apiKey,omitempty"`
				Config                    map[string]interface{} `json:"config,omitempty"`
				WaitForCompletion         string                 `json:"waitForCompletion,omitempty"`
			}
			var args createModelAPIArguments
			if err := request.BindArguments(&args); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
			if args.Name == "" {
				return mcp.NewToolResultError("model API name is required"), nil
			}
			if cfg.AsyncLifecycleOnly {
				normalizedWait, err := normalizeAsyncModelAPIWait(args.WaitForCompletion, "creation", "until status is DEPLOYED or FAILED")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				args.WaitForCompletion = normalizedWait
			} else if strings.TrimSpace(args.WaitForCompletion) == "" {
				args.WaitForCompletion = "true"
			}

			activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
			}
			result, err := activeHandler.CreateModelAPI(ctx, args.Name, args.Model, args.Endpoint, args.IntegrationConnectionName, args.Provider, args.APIKey, args.WaitForCompletion, args.Config)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})

		// Delete model API tool
		deleteModelAPITool := mcp.NewTool("delete_model_api",
			mcp.WithToolTitle("Delete Model API"),
			mcp.WithTitleAnnotation("Delete Model API"),
			mcp.WithDescription("Delete a model API by name"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the model API to delete"),
			),
			mcp.WithString("waitForCompletion", modelAPILifecycleWaitOptions(
				cfg,
				"Async-only calls do not wait. Use false or omit this field, then poll get_model_api until the model API is not found.",
				"Whether to wait for the model API to be fully deleted (true/false, default: true).",
			)...),
			mcp.WithString("workspace",
				mcp.Description("Optional workspace name to override the default workspace"),
			),
		)

		s.AddTool(deleteModelAPITool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := request.GetString("name", "")
			if name == "" {
				return mcp.NewToolResultError("model API name is required"), nil
			}

			defaultWait := "true"
			if cfg.AsyncLifecycleOnly {
				defaultWait = "false"
			}
			waitForCompletion := request.GetString("waitForCompletion", defaultWait)
			if cfg.AsyncLifecycleOnly {
				normalizedWait, err := normalizeAsyncModelAPIWait(waitForCompletion, "deletion", "until the model API is not found")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				waitForCompletion = normalizedWait
			}

			activeHandler, err := resolveHandler(handler, cfg, request.GetString("workspace", ""))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to override workspace: %v", err)), nil
			}
			result, err := activeHandler.DeleteModelAPI(ctx, name, waitForCompletion)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(string(result)), nil
		})
	}
}

func modelAPILifecycleWaitOptions(cfg *config.Config, asyncDescription, standaloneDescription string) []mcp.PropertyOption {
	if cfg.AsyncLifecycleOnly {
		return []mcp.PropertyOption{mcp.Description(asyncDescription), mcp.Enum("false"), mcp.DefaultString("false")}
	}
	return []mcp.PropertyOption{mcp.Description(standaloneDescription), mcp.Enum("true", "false"), mcp.DefaultString("true")}
}

func normalizeAsyncModelAPIWait(value, operation, pollCondition string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "false":
		return "false", nil
	case "true":
		return "", fmt.Errorf("waitForCompletion=true is not supported for async-only model API %s because the request may outlive the call. Use false or omit waitForCompletion, then poll get_model_api %s", operation, pollCondition)
	default:
		return "", fmt.Errorf("waitForCompletion must be false or omitted for async-only model API %s. Use false or omit waitForCompletion, then poll get_model_api %s", operation, pollCondition)
	}
}
