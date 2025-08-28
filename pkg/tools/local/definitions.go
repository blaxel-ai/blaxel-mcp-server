package local

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// LocalHandler defines the interface for local operations
type LocalHandler interface {
	QuickStartGuide(resourceType string) (string, error)
	ListTemplates(ctx context.Context, resourceType string) (string, error)
	CreateAgent(directory, template string) (string, error)
	CreateJob(directory, template string) (string, error)
	CreateMCPServer(directory, template string) (string, error)
	CreateSandbox(directory, template string) (string, error)
	DeployDirectory(directory string) (string, error)
	RunDeployedResource(resourceType, resourceName string) (string, error)
}

// LocalHandlerWithReadOnly extends LocalHandler with readonly capability
type LocalHandlerWithReadOnly interface {
	LocalHandler
	IsReadOnly() bool
}

// RegisterLocalTools registers local tools with the given handler
func RegisterLocalTools(s *server.MCPServer, handler LocalHandler) {
	// Check if handler supports readonly mode
	readOnlyHandler, hasReadOnly := handler.(LocalHandlerWithReadOnly)
	isReadOnly := hasReadOnly && readOnlyHandler.IsReadOnly()

	// Quick start guide tool
	quickStartTool := mcp.NewTool("local_quick_start_guide",
		mcp.WithDescription("Get a quick start guide for creating Blaxel resources without credentials"),
		mcp.WithString("resourceType",
			mcp.Description("Type of resource to get quick start guide for (agent, job, mcp-server, sandbox, all)"),
			mcp.Enum("agent", "job", "mcp-server", "sandbox", "all"),
		),
	)

	s.AddTool(quickStartTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resourceType := request.GetString("resourceType", "all")

		result, err := handler.QuickStartGuide(resourceType)
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
		resourceType := request.GetString("resourceType", "")
		if resourceType == "" {
			return mcp.NewToolResultError("resourceType is required"), nil
		}

		result, err := handler.ListTemplates(ctx, resourceType)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	// Only register write operations if not in read-only mode
	if !isReadOnly {
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
			directory := request.GetString("directory", "")
			if directory == "" {
				return mcp.NewToolResultError("directory is required"), nil
			}

			template := request.GetString("template", "")

			result, err := handler.CreateAgent(directory, template)
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
			directory := request.GetString("directory", "")
			if directory == "" {
				return mcp.NewToolResultError("directory is required"), nil
			}

			template := request.GetString("template", "")

			result, err := handler.CreateJob(directory, template)
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
			directory := request.GetString("directory", "")
			if directory == "" {
				return mcp.NewToolResultError("directory is required"), nil
			}

			template := request.GetString("template", "")

			result, err := handler.CreateMCPServer(directory, template)
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
			directory := request.GetString("directory", "")
			if directory == "" {
				return mcp.NewToolResultError("directory is required"), nil
			}

			template := request.GetString("template", "")

			result, err := handler.CreateSandbox(directory, template)
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
			directory := request.GetString("directory", "")

			result, err := handler.DeployDirectory(directory)
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
			resourceType := request.GetString("resourceType", "")
			resourceName := request.GetString("resourceName", "")

			if resourceType == "" || resourceName == "" {
				return mcp.NewToolResultError("resourceType and resourceName are required"), nil
			}

			result, err := handler.RunDeployedResource(resourceType, resourceName)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(result), nil
		})
	}
}
