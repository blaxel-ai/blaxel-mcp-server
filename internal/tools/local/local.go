package local

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/client"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all local CLI tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client for templates
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// Quick start guide tool
	s.AddTool(
		mcp.NewToolWithRawSchema("local_quick_start_guide",
			"Get a quick start guide for creating Blaxel resources without credentials",
			json.RawMessage(`{"type": "object", "properties": {"resourceType": {"type": "string", "enum": ["agent", "job", "mcp-server", "sandbox", "all"], "default": "all", "description": "Type of resource to get quick start guide for"}}}`)),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			result, err := localQuickStartGuideHandler(args)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{mcp.NewTextContent(err.Error())},
					IsError: true,
				}, nil
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.NewTextContent(result)},
			}, nil
		},
	)

	// List templates tool
	s.AddTool(
		mcp.NewToolWithRawSchema("local_list_templates",
			"List available templates for a specific resource type",
			json.RawMessage(`{"type": "object", "properties": {"resourceType": {"type": "string", "enum": ["agent", "job", "sandbox", "mcp-server", "all"], "description": "Type of resource to list templates for"}}, "required": ["resourceType"]}`)),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, _ := request.Params.Arguments.(map[string]interface{})
			result, err := localListTemplatesHandler(ctx, sdkClient, args)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{mcp.NewTextContent(err.Error())},
					IsError: true,
				}, nil
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.NewTextContent(result)},
			}, nil
		},
	)

	if !cfg.ReadOnly {
		// Create agent locally
		s.AddTool(
			mcp.NewToolWithRawSchema("local_create_agent",
				"Create a new Blaxel agent app project locally using CLI",
				json.RawMessage(`{"type": "object", "properties": {"directory": {"type": "string", "description": "Path to create agent in"}, "template": {"type": "string", "description": "Template to use"}}, "required": ["directory"]}`)),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args, _ := request.Params.Arguments.(map[string]interface{})
				result, err := localCreateAgentHandler(cfg, args)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{mcp.NewTextContent(err.Error())},
						IsError: true,
					}, nil
				}
				return &mcp.CallToolResult{
					Content: []mcp.Content{mcp.NewTextContent(result)},
				}, nil
			},
		)

		// Create job locally
		s.AddTool(
			mcp.NewToolWithRawSchema("local_create_job",
				"Create a new Blaxel job project locally using CLI",
				json.RawMessage(`{"type": "object", "properties": {"directory": {"type": "string", "description": "Path to create job in"}, "template": {"type": "string", "description": "Template to use"}}, "required": ["directory"]}`)),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args, _ := request.Params.Arguments.(map[string]interface{})
				result, err := localCreateJobHandler(cfg, args)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{mcp.NewTextContent(err.Error())},
						IsError: true,
					}, nil
				}
				return &mcp.CallToolResult{
					Content: []mcp.Content{mcp.NewTextContent(result)},
				}, nil
			},
		)

		// Create MCP server locally
		s.AddTool(
			mcp.NewToolWithRawSchema("local_create_mcp_server",
				"Create a new Blaxel MCP server project locally using CLI",
				json.RawMessage(`{"type": "object", "properties": {"directory": {"type": "string", "description": "Path to create MCP server in"}, "template": {"type": "string", "description": "Template to use"}}, "required": ["directory"]}`)),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args, _ := request.Params.Arguments.(map[string]interface{})
				result, err := localCreateMCPServerHandler(cfg, args)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{mcp.NewTextContent(err.Error())},
						IsError: true,
					}, nil
				}
				return &mcp.CallToolResult{
					Content: []mcp.Content{mcp.NewTextContent(result)},
				}, nil
			},
		)

		// Create sandbox locally
		s.AddTool(
			mcp.NewToolWithRawSchema("local_create_sandbox",
				"Create a new Blaxel sandbox project locally using CLI",
				json.RawMessage(`{"type": "object", "properties": {"directory": {"type": "string", "description": "Path to create sandbox in"}, "template": {"type": "string", "description": "Template to use"}}, "required": ["directory"]}`)),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args, _ := request.Params.Arguments.(map[string]interface{})
				result, err := localCreateSandboxHandler(cfg, args)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{mcp.NewTextContent(err.Error())},
						IsError: true,
					}, nil
				}
				return &mcp.CallToolResult{
					Content: []mcp.Content{mcp.NewTextContent(result)},
				}, nil
			},
		)

		// Deploy directory
		s.AddTool(
			mcp.NewToolWithRawSchema("local_deploy_directory",
				"Deploy a local directory containing agent, MCP server, or job code to Blaxel",
				json.RawMessage(`{"type": "object", "properties": {"directory": {"type": "string", "description": "Path to directory to deploy"}, "name": {"type": "string", "description": "Optional name for deployment"}, "skipBuild": {"type": "boolean", "description": "Skip the build step"}, "dryRun": {"type": "boolean", "description": "Perform a dry run without actually deploying"}}}`)),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args, _ := request.Params.Arguments.(map[string]interface{})
				result, err := localDeployHandler(cfg, args)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{mcp.NewTextContent(err.Error())},
						IsError: true,
					}, nil
				}
				return &mcp.CallToolResult{
					Content: []mcp.Content{mcp.NewTextContent(result)},
				}, nil
			},
		)

		// Run deployed resource
		s.AddTool(
			mcp.NewToolWithRawSchema("local_run_deployed_resource",
				"Run a deployed resource on Blaxel",
				json.RawMessage(`{"type": "object", "properties": {"resourceType": {"type": "string", "enum": ["agent", "model", "job", "function"]}, "resourceName": {"type": "string"}}, "required": ["resourceType", "resourceName"]}`)),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args, _ := request.Params.Arguments.(map[string]interface{})
				result, err := localRunHandler(cfg, args)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{mcp.NewTextContent(err.Error())},
						IsError: true,
					}, nil
				}
				return &mcp.CallToolResult{
					Content: []mcp.Content{mcp.NewTextContent(result)},
				}, nil
			},
		)
	}
}

// Handler functions will be in local_handlers.go
