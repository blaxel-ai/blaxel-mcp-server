package local

import (
	"context"
	"fmt"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/client"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Request/Response types for type safety
type QuickStartGuideRequest struct {
	ResourceType string `json:"resourceType,omitempty"`
}

type ListTemplatesRequest struct {
	ResourceType string `json:"resourceType"`
}

type CreateResourceRequest struct {
	Directory string `json:"directory"`
	Template  string `json:"template,omitempty"`
}

type DeployResourceRequest struct {
	Directory string `json:"directory"`
}

type RunResourceRequest struct {
	ResourceType string `json:"resourceType"`
	ResourceName string `json:"resourceName"`
}

// RegisterTools registers all local CLI tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client for templates
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// Quick start guide tool
	quickStartTool := mcp.NewTool("local_quick_start_guide",
		mcp.WithDescription("Get a quick start guide for creating Blaxel resources without credentials"),
		mcp.WithString("resourceType",
			mcp.Description("Type of resource to get quick start guide for (agent, job, mcp-server, sandbox, all)"),
			mcp.Enum("agent", "job", "mcp-server", "sandbox", "all"),
		),
	)

	s.AddTool(quickStartTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req QuickStartGuideRequest
		if err := request.BindArguments(&req); err != nil {
			// If binding fails, try to get resourceType directly for backward compatibility
			req.ResourceType = request.GetString("resourceType", "all")
		}

		result, err := localQuickStartGuideHandler(req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	// List templates tool
	listTemplatesTool := mcp.NewTool("local_list_templates",
		mcp.WithDescription("List available templates for a specific resource type"),
		mcp.WithString("resourceType",
			mcp.Required(),
			mcp.Description("Type of resource to list templates for"),
			mcp.Enum("agent", "job", "sandbox", "mcp-server", "all"),
		),
	)

	s.AddTool(listTemplatesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req ListTemplatesRequest
		if err := request.BindArguments(&req); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		if req.ResourceType == "" {
			return mcp.NewToolResultError("resourceType is required"), nil
		}

		result, err := localListTemplatesHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	if !cfg.ReadOnly {
		// Create agent locally
		createAgentTool := mcp.NewTool("local_create_agent",
			mcp.WithDescription("Create a new Blaxel agent app project locally using CLI"),
			mcp.WithString("directory",
				mcp.Required(),
				mcp.Description("Path to create agent in"),
			),
			mcp.WithString("template",
				mcp.Description("Template to use"),
			),
		)

		s.AddTool(createAgentTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req CreateResourceRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Directory == "" {
				return mcp.NewToolResultError("directory is required"), nil
			}

			result, err := localCreateAgentHandler(cfg, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(result), nil
		})

		// Create job locally
		createJobTool := mcp.NewTool("local_create_job",
			mcp.WithDescription("Create a new Blaxel job project locally using CLI"),
			mcp.WithString("directory",
				mcp.Required(),
				mcp.Description("Path to create job in"),
			),
			mcp.WithString("template",
				mcp.Description("Template to use"),
			),
		)

		s.AddTool(createJobTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req CreateResourceRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Directory == "" {
				return mcp.NewToolResultError("directory is required"), nil
			}

			result, err := localCreateJobHandler(cfg, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(result), nil
		})

		// Create MCP server locally
		createMCPServerTool := mcp.NewTool("local_create_mcp_server",
			mcp.WithDescription("Create a new Blaxel MCP server project locally using CLI"),
			mcp.WithString("directory",
				mcp.Required(),
				mcp.Description("Path to create MCP server in"),
			),
			mcp.WithString("template",
				mcp.Description("Template to use"),
			),
		)

		s.AddTool(createMCPServerTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req CreateResourceRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Directory == "" {
				return mcp.NewToolResultError("directory is required"), nil
			}

			result, err := localCreateMCPServerHandler(cfg, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(result), nil
		})

		// Create sandbox locally
		createSandboxTool := mcp.NewTool("local_create_sandbox",
			mcp.WithDescription("Create a new Blaxel sandbox project locally using CLI"),
			mcp.WithString("directory",
				mcp.Required(),
				mcp.Description("Path to create sandbox in"),
			),
			mcp.WithString("template",
				mcp.Description("Template to use"),
			),
		)

		s.AddTool(createSandboxTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req CreateResourceRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Directory == "" {
				return mcp.NewToolResultError("directory is required"), nil
			}

			result, err := localCreateSandboxHandler(cfg, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(result), nil
		})

		// Deploy directory
		deployTool := mcp.NewTool("local_deploy_directory",
			mcp.WithDescription("Deploy a local directory containing agent, MCP server, or job code to Blaxel"),
			mcp.WithString("directory",
				mcp.Description("Path to directory to deploy"),
			),
		)

		s.AddTool(deployTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req DeployResourceRequest
			if err := request.BindArguments(&req); err != nil {
				// Try to get directory for backward compatibility
				req.Directory = request.GetString("directory", "")
			}

			result, err := localDeployHandler(cfg, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(result), nil
		})

		// Run deployed resource
		runTool := mcp.NewTool("local_run_deployed_resource",
			mcp.WithDescription("Run a deployed resource on Blaxel"),
			mcp.WithString("resourceType",
				mcp.Required(),
				mcp.Description("Type of resource to run"),
				mcp.Enum("agent", "model", "job", "function"),
			),
			mcp.WithString("resourceName",
				mcp.Required(),
				mcp.Description("Name of the resource to run"),
			),
		)

		s.AddTool(runTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req RunResourceRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.ResourceType == "" || req.ResourceName == "" {
				return mcp.NewToolResultError("resourceType and resourceName are required"), nil
			}

			result, err := localRunHandler(cfg, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(result), nil
		})
	}
}

// Handler functions will be in local_handlers.go
