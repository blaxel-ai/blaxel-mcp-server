package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestMain runs before all tests and loads environment variables
func TestMain(m *testing.M) {
	// Load .env file from the project root (parent directory of e2e)
	envPath := filepath.Join("..", ".env")
	if err := godotenv.Load(envPath); err != nil {
		// It's okay if .env doesn't exist, we might use actual env vars
		fmt.Printf("Note: Could not load .env file from %s: %v\n", envPath, err)
	}

	os.Exit(m.Run())
}

// MCPTestClient wraps the official mcp-go client for testing
type MCPTestClient struct {
	client *client.Client
	ctx    context.Context
	cancel context.CancelFunc
}

// NewMCPTestClient creates a new test client using the official mcp-go library
func NewMCPTestClient(t *testing.T, env map[string]string) *MCPTestClient {
	t.Helper()

	// Find the server binary
	serverPath := filepath.Join("..", "build", "blaxel-mcp-server")
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		t.Fatalf("Server binary not found at %s. Run 'make build' first.", serverPath)
	}

	// Prepare environment variables
	envVars := os.Environ()
	for k, v := range env {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	// Create the MCP client using stdio transport
	stdioClient, err := client.NewStdioMCPClient(serverPath, envVars)
	if err != nil {
		t.Fatalf("Failed to create MCP client: %v", err)
	}

	// Create context with timeout for initialization
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	// Initialize the client with the server
	_, err = stdioClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "Integration Test Client",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		cancel()
		stdioClient.Close()
		t.Fatalf("Failed to initialize MCP client: %v", err)
	}

	return &MCPTestClient{
		client: stdioClient,
		ctx:    ctx,
		cancel: cancel,
	}
}

// CallTool calls a tool and returns the result
func (c *MCPTestClient) CallTool(name string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	return c.client.CallTool(c.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: arguments,
		},
	})
}

// ListTools lists available tools
func (c *MCPTestClient) ListTools() (*mcp.ListToolsResult, error) {
	return c.client.ListTools(c.ctx, mcp.ListToolsRequest{})
}

// Ping sends a ping to the server
func (c *MCPTestClient) Ping() error {
	return c.client.Ping(c.ctx)
}

// GetInitializeResult returns the initialization result
func (c *MCPTestClient) GetServerCapabilities() mcp.ServerCapabilities {
	return c.client.GetServerCapabilities()
}

// Close shuts down the test client
func (c *MCPTestClient) Close() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.client != nil {
		c.client.Close()
	}
}

// Helper function to create test environment
func testEnv() map[string]string {
	env := map[string]string{
		"BL_API_KEY":   getEnvOrDefault("BL_API_KEY", "test-key"),
		"BL_WORKSPACE": getEnvOrDefault("BL_WORKSPACE", "test-workspace"),
	}
	return env
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// Helper function to check if a tool call resulted in an error
func checkToolError(result *mcp.CallToolResult) (bool, string) {
	if result == nil {
		return false, ""
	}

	if result.IsError {
		if len(result.Content) > 0 {
			for _, content := range result.Content {
				// Try to cast to TextContent
				if textContent, ok := mcp.AsTextContent(content); ok {
					return true, textContent.Text
				}
				// Try to marshal and unmarshal to check content type
				if data, err := json.Marshal(content); err == nil {
					var contentMap map[string]interface{}
					if err := json.Unmarshal(data, &contentMap); err == nil {
						if contentMap["type"] == "text" {
							if text, ok := contentMap["text"].(string); ok {
								return true, text
							}
						}
					}
				}
			}
		}
		return true, "Error occurred but no message available"
	}

	return false, ""
}

// Helper function to extract JSON result from tool response
func extractJSONResult(result *mcp.CallToolResult) (map[string]interface{}, error) {
	if result == nil || len(result.Content) == 0 {
		return nil, fmt.Errorf("no content in result")
	}

	for _, content := range result.Content {
		var text string
		// Try to cast to TextContent
		if textContent, ok := mcp.AsTextContent(content); ok {
			text = textContent.Text
		} else {
			// Try to marshal and unmarshal to extract text
			if data, err := json.Marshal(content); err == nil {
				var contentMap map[string]interface{}
				if err := json.Unmarshal(data, &contentMap); err == nil {
					if contentMap["type"] == "text" {
						if str, ok := contentMap["text"].(string); ok {
							text = str
						}
					}
				}
			}
		}

		if text != "" {
			var resp map[string]interface{}
			if err := json.Unmarshal([]byte(text), &resp); err == nil {
				return resp, nil
			}
		}
	}

	return nil, fmt.Errorf("no valid JSON content found")
}
