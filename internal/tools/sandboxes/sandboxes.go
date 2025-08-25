package sandboxes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/client"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/blaxel-ai/toolkit/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all sandbox-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		// Log error but continue - tools will return errors when called
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// List sandboxes tool
	listSandboxesSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"filter": {
				"type": "string",
				"description": "Optional filter string"
			}
		}
	}`)

	s.AddTool(
		mcp.NewToolWithRawSchema("list_sandboxes", "List all sandboxes in the workspace", listSandboxesSchema),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, ok := request.Params.Arguments.(map[string]interface{})
			if !ok {
				args = make(map[string]interface{})
			}

			result, err := listSandboxesHandler(ctx, sdkClient, args)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent(err.Error()),
					},
					IsError: true,
				}, nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(string(jsonResult)),
				},
			}, nil
		},
	)

	// Get sandbox tool
	getSandboxSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": {
				"type": "string",
				"description": "Name of the sandbox to retrieve"
			}
		},
		"required": ["name"]
	}`)

	s.AddTool(
		mcp.NewToolWithRawSchema("get_sandbox", "Get details of a specific sandbox", getSandboxSchema),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, ok := request.Params.Arguments.(map[string]interface{})
			if !ok {
				args = make(map[string]interface{})
			}

			result, err := getSandboxHandler(ctx, sdkClient, args)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent(err.Error()),
					},
					IsError: true,
				}, nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(string(jsonResult)),
				},
			}, nil
		},
	)

	// Only register write operations if not in read-only mode
	if !cfg.ReadOnly {
		// Delete sandbox tool
		deleteSandboxSchema := json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {
					"type": "string",
					"description": "Name of the sandbox to delete"
				}
			},
			"required": ["name"]
		}`)

		s.AddTool(
			mcp.NewToolWithRawSchema("delete_sandbox", "Delete a sandbox by name", deleteSandboxSchema),
			func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args, ok := request.Params.Arguments.(map[string]interface{})
				if !ok {
					args = make(map[string]interface{})
				}

				result, err := deleteSandboxHandler(ctx, sdkClient, args)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.NewTextContent(err.Error()),
						},
						IsError: true,
					}, nil
				}

				jsonResult, _ := json.MarshalIndent(result, "", "  ")
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent(string(jsonResult)),
					},
				}, nil
			},
		)
	}
}

func listSandboxesHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	sandboxes, err := sdkClient.ListSandboxesWithResponse(ctx)
	if sandboxes.JSON200 == nil {
		return nil, fmt.Errorf("no sandboxes found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list sandboxes: %w", err)
	}

	// Apply optional filter
	filter, _ := params["filter"].(string)
	var filteredSandboxes []map[string]interface{}

	for _, sandbox := range *sandboxes.JSON200 {
		// Check if filter matches
		if filter != "" {
			name := ""
			if sandbox.Metadata != nil && sandbox.Metadata.Name != nil {
				name = *sandbox.Metadata.Name
			}
			// Skip if name doesn't contain filter
			if name == "" || !containsString(name, filter) {
				continue
			}
		}

		// Build sandbox info
		sandboxInfo := map[string]interface{}{
			"name": "",
		}

		if sandbox.Metadata != nil && sandbox.Metadata.Name != nil {
			sandboxInfo["name"] = *sandbox.Metadata.Name
		}

		filteredSandboxes = append(filteredSandboxes, sandboxInfo)
	}

	return map[string]interface{}{
		"sandboxes": filteredSandboxes,
		"count":     len(filteredSandboxes),
	}, nil
}

func getSandboxHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("sandbox name is required")
	}

	sandbox, err := sdkClient.GetSandboxWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox: %w", err)
	}
	if sandbox.JSON200 == nil {
		return nil, fmt.Errorf("no sandbox found")
	}

	jsonData, _ := json.MarshalIndent(*sandbox.JSON200, "", "  ")

	return map[string]interface{}{
		"sandbox": json.RawMessage(jsonData),
		"name":    name,
	}, nil
}

func deleteSandboxHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (interface{}, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("sandbox name is required")
	}

	_, err := sdkClient.DeleteSandboxWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete sandbox: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Sandbox '%s' deleted successfully", name),
	}, nil
}

// Helper function to check if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && strings.Contains(strings.ToLower(s), strings.ToLower(substr)))
}
