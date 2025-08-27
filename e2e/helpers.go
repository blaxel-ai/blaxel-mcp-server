package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/blaxel-ai/toolkit/sdk"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPTestClient wraps the official mcp-go client for testing
type MCPTestClient struct {
	client *client.Client
	ctx    context.Context
	cancel context.CancelFunc
}

// NewMCPTestClient creates a new test client using the official mcp-go library
func NewMCPTestClient(t *testing.T, env map[string]string) *MCPTestClient {
	t.Helper()

	// Find the server binary - handle different test locations
	var serverPath string
	// Try different possible paths based on where the test is running from
	possiblePaths := []string{
		filepath.Join("..", "build", "blaxel-mcp-server"),       // from e2e/tools/
		filepath.Join("..", "..", "build", "blaxel-mcp-server"), // from e2e/
		filepath.Join("build", "blaxel-mcp-server"),             // from project root
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			serverPath = path
			break
		}
	}

	if serverPath == "" {
		t.Fatalf("Server binary not found. Tried paths: %v. Run 'make build' first.", possiblePaths)
	}

	// Prepare environment variables
	envVars := os.Environ()
	for k, v := range env {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	fmt.Println("envVars", envVars)
	// Create the MCP client using stdio transport
	stdioClient, err := client.NewStdioMCPClient(serverPath, envVars)
	if err != nil {
		t.Fatalf("Failed to create MCP client: %v", err)
	}

	// Create context with timeout for initialization
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)

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

// Close shuts down the test client
func (c *MCPTestClient) Close() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.client != nil {
		c.client.Close()
	}
}

// ListTools lists available tools
func (c *MCPTestClient) ListTools() (*mcp.ListToolsResult, error) {
	return c.client.ListTools(c.ctx, mcp.ListToolsRequest{})
}

// Ping sends a ping to the server
func (c *MCPTestClient) Ping() error {
	return c.client.Ping(c.ctx)
}

// GetServerCapabilities returns the server capabilities
func (c *MCPTestClient) GetServerCapabilities() mcp.ServerCapabilities {
	return c.client.GetServerCapabilities()
}

// TestEnv creates a test environment for external packages
func TestEnv() map[string]string {
	env := map[string]string{}
	return env
}

// CheckToolError checks if a tool call resulted in an error
func CheckToolError(result *mcp.CallToolResult) (bool, string) {
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

// ExtractJSONResult extracts JSON result from tool response
func ExtractJSONResult(result *mcp.CallToolResult) (map[string]interface{}, error) {
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

// GenerateRandomTestName generates a random test name with the given prefix
func GenerateRandomTestName(prefix string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%s-%d", prefix, r.Intn(10000))
}

// CleanupTestIntegrations cleans up integrations matching regex patterns
func CleanupTestIntegrations(regexPatterns []string) {
	// Use bl CLI to list integrations
	cmd := exec.Command("bl", "get", "ic", "-ojson")
	output, err := cmd.Output()
	if err != nil {
		return // Skip cleanup if listing fails
	}

	// Parse the JSON output
	var integrations []sdk.IntegrationConnection
	if err := json.Unmarshal(output, &integrations); err != nil {
		return // Skip cleanup if parsing fails
	}

	// Find and delete integrations matching the regex patterns
	for _, integration := range integrations {
		name := *integration.Metadata.Name

		// Check each regex pattern
		for _, pattern := range regexPatterns {
			if matched, _ := regexp.MatchString(pattern, name); matched {
				// This matches our pattern, delete it using bl CLI
				deleteCmd := exec.Command("bl", "delete", "ic", name)
				_ = deleteCmd.Run() // Ignore errors for cleanup
				break               // Delete once per integration
			}
		}
	}
}
