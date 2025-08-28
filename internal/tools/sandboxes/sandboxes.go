package sandboxes

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/client"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/tools"
	"github.com/blaxel-ai/toolkit/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Request/Response types for type safety
type ListSandboxesRequest struct {
	Filter string `json:"filter,omitempty"`
}

type ListSandboxesResponse json.RawMessage

type GetSandboxRequest struct {
	Name string `json:"name"`
}

type GetSandboxResponse json.RawMessage

type CreateSandboxRequest struct {
	Name   string  `json:"name"`
	Image  string  `json:"image,omitempty"`
	Memory float64 `json:"memory,omitempty"`
	Env    string  `json:"env,omitempty"`
	Ports  string  `json:"ports,omitempty"`
}

type CreateSandboxResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Sandbox map[string]interface{} `json:"sandbox"`
}

type DeleteSandboxRequest struct {
	Name string `json:"name"`
}

type DeleteSandboxResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RegisterTools registers all sandbox-related tools
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		// Log error but continue - tools will return errors when called
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// List sandboxes tool
	listSandboxesTool := mcp.NewTool("list_sandboxes",
		mcp.WithDescription("List all sandboxes in the workspace"),
		mcp.WithString("filter",
			mcp.Description("Optional filter string"),
		),
	)

	s.AddTool(listSandboxesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req ListSandboxesRequest
		if err := request.BindArguments(&req); err != nil {
			// If binding fails, try to get filter directly for backward compatibility
			req.Filter = request.GetString("filter", "")
		}

		result, err := listSandboxesHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(*result)), nil
	})

	// Get sandbox tool
	getSandboxTool := mcp.NewTool("get_sandbox",
		mcp.WithDescription("Get details of a specific sandbox"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the sandbox to retrieve"),
		),
	)

	s.AddTool(getSandboxTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var req GetSandboxRequest
		if err := request.BindArguments(&req); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		if req.Name == "" {
			return mcp.NewToolResultError("sandbox name is required"), nil
		}

		result, err := getSandboxHandler(ctx, sdkClient, req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(string(*result)), nil
	})

	// Only register write operations if not in read-only mode
	if !cfg.ReadOnly {
		// Create sandbox tool
		createSandboxTool := mcp.NewTool("create_sandbox",
			mcp.WithDescription("Create a new sandbox"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name for the sandbox"),
			),
			mcp.WithString("image",
				mcp.Description("Docker image to use for the sandbox"),
			),
			mcp.WithNumber("memory",
				mcp.Description("Memory in MB (default: 512)"),
			),
			mcp.WithString("ports", mcp.Description("Ports to expose from the sandbox, separated by commas (eg. 8080,8081)")),
			mcp.WithString("env", mcp.Description("Environment variables to set in the sandbox, separated by commas (eg. FOO=bar,BAR=baz)")),
		)

		s.AddTool(createSandboxTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req CreateSandboxRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Name == "" {
				return mcp.NewToolResultError("sandbox name is required"), nil
			}

			result, err := createSandboxHandler(ctx, sdkClient, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(jsonResult)), nil
		})

		// Delete sandbox tool
		deleteSandboxTool := mcp.NewTool("delete_sandbox",
			mcp.WithDescription("Delete a sandbox by name"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Name of the sandbox to delete"),
			),
		)

		s.AddTool(deleteSandboxTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var req DeleteSandboxRequest
			if err := request.BindArguments(&req); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
			}

			if req.Name == "" {
				return mcp.NewToolResultError("sandbox name is required"), nil
			}

			result, err := deleteSandboxHandler(ctx, sdkClient, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResultText(string(jsonResult)), nil
		})
	}
}

func listSandboxesHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req ListSandboxesRequest) (*ListSandboxesResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := sdkClient.ListSandboxesWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sandboxes: %w", err)
	}

	sandboxes := []sdk.Sandbox{}
	if resp.JSON200 != nil {
		sandboxes = *resp.JSON200
	}

	// Apply filter if requested
	if req.Filter != "" {
		var filtered []sdk.Sandbox
		for _, sandbox := range sandboxes {
			if sandbox.Metadata != nil && sandbox.Metadata.Name != nil &&
				tools.ContainsString(*sandbox.Metadata.Name, req.Filter) {
				filtered = append(filtered, sandbox)
			}
		}
		sandboxes = filtered
	}

	// Format the sandboxes using the new formatter
	formatted := tools.FormatSandboxes(sandboxes)
	jsonData, err := json.Marshal(formatted)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal formatted sandboxes: %w", err)
	}

	response := ListSandboxesResponse(jsonData)
	return &response, nil
}

func getSandboxHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req GetSandboxRequest) (*GetSandboxResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	sandbox, err := sdkClient.GetSandboxWithResponse(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox: %w", err)
	}
	if sandbox.JSON200 == nil {
		return nil, fmt.Errorf("no sandbox found")
	}

	jsonData, _ := json.MarshalIndent(*sandbox.JSON200, "", "  ")
	response := GetSandboxResponse(jsonData)
	return &response, nil
}

func createSandboxHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req CreateSandboxRequest) (*CreateSandboxResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	// Build sandbox request
	sandboxData := sdk.CreateSandboxJSONRequestBody{
		Metadata: &sdk.Metadata{
			Name: &req.Name,
		},
		Spec: &sdk.SandboxSpec{
			Runtime: &sdk.Runtime{},
		},
	}

	// Add optional image
	if req.Image != "" {
		sandboxData.Spec.Runtime.Image = &req.Image
	}

	// Add optional memory
	if req.Memory > 0 {
		mem := int(req.Memory)
		sandboxData.Spec.Runtime.Memory = &mem
	}

	// Add optional ports
	if req.Ports != "" {
		defaultProtocol := "TCP"
		portStrings := strings.Split(req.Ports, ",")
		ports := make([]sdk.Port, 0, len(portStrings))
		for _, portStr := range portStrings {
			portStr = strings.TrimSpace(portStr)
			if portStr == "" {
				continue
			}
			intPort, err := strconv.Atoi(portStr)
			if err != nil {
				return nil, fmt.Errorf("invalid port '%s': %w", portStr, err)
			}
			portData := sdk.Port{
				Target:   &intPort,
				Protocol: &defaultProtocol,
			}
			ports = append(ports, portData)
		}
		if len(ports) > 0 {
			sandboxData.Spec.Runtime.Ports = &ports
		}
	}
	// Add optional environment variables
	if req.Env != "" {
		sandboxData.Spec.Runtime.Envs = tools.SetRuntimeEnv(req.Env)
	}

	// Add optional environment variables
	sandbox, err := sdkClient.CreateSandboxWithResponse(ctx, sandboxData)
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox: %w", err)
	}

	if sandbox.JSON200 == nil {
		if sandbox.StatusCode() == 409 {
			return nil, fmt.Errorf("sandbox with name '%s' already exists", req.Name)
		}
		return nil, fmt.Errorf("failed to create sandbox with status %d", sandbox.StatusCode())
	}

	result := &CreateSandboxResponse{
		Success: true,
		Message: fmt.Sprintf("Sandbox '%s' created successfully", req.Name),
		Sandbox: map[string]interface{}{
			"name": req.Name,
		},
	}

	if sandbox.JSON200 != nil {
		if sandbox.JSON200.Status != nil {
			result.Sandbox["status"] = *sandbox.JSON200.Status
		}
	}

	return result, nil
}

func deleteSandboxHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, req DeleteSandboxRequest) (*DeleteSandboxResponse, error) {
	if sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	_, err := sdkClient.DeleteSandboxWithResponse(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete sandbox: %w", err)
	}

	return &DeleteSandboxResponse{
		Success: true,
		Message: fmt.Sprintf("Sandbox '%s' deleted successfully", req.Name),
	}, nil
}

// Helper function to check if a string contains a substring (case-insensitive)
