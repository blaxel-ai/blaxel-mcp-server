package e2e

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// MCPTestClient represents a test client for the MCP server
type MCPTestClient struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	scanner *bufio.Scanner
	mu      sync.Mutex
	reqID   int
}

// NewMCPTestClient creates a new test client
func NewMCPTestClient(t *testing.T, env map[string]string) *MCPTestClient {
	t.Helper()

	// Find the server binary
	serverPath := filepath.Join("..", "build", "blaxel-mcp-server")
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		t.Fatalf("Server binary not found at %s. Run 'make build' first.", serverPath)
	}

	// Set up command
	cmd := exec.Command(serverPath)

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	// Start the server
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	client := &MCPTestClient{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		scanner: bufio.NewScanner(stdout),
		reqID:   1,
	}

	// Wait for server to be ready
	time.Sleep(500 * time.Millisecond)

	return client
}

// SendRequest sends a JSON-RPC request and returns the response
func (c *MCPTestClient) SendRequest(method string, params interface{}) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create request
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"id":      c.reqID,
	}
	if params != nil {
		req["params"] = params
	}
	c.reqID++

	// Send request
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if _, err := fmt.Fprintf(c.stdin, "%s\n", reqBytes); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	if !c.scanner.Scan() {
		if err := c.scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		return nil, fmt.Errorf("no response received")
	}

	// Parse response
	var resp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    string `json:"data,omitempty"`
		} `json:"error"`
	}

	if err := json.Unmarshal([]byte(c.scanner.Text()), &resp); err != nil {
		// Skip non-JSON lines (like startup logs)
		if c.scanner.Scan() {
			if err := json.Unmarshal([]byte(c.scanner.Text()), &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal response: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("server error: %s (code: %d)", resp.Error.Message, resp.Error.Code)
	}

	return resp.Result, nil
}

// Close shuts down the test client
func (c *MCPTestClient) Close() {
	c.stdin.Close()
	c.stdout.Close()

	// Give the server time to shut down gracefully
	done := make(chan error, 1)
	go func() {
		done <- c.cmd.Wait()
	}()

	select {
	case <-done:
		// Server exited
	case <-time.After(2 * time.Second):
		// Force kill if it doesn't exit
		c.cmd.Process.Kill()
		<-done
	}
}

// Helper function to create test environment
func testEnv() map[string]string {
	env := map[string]string{
		"BLAXEL_API_KEY":   getEnvOrDefault("BLAXEL_API_KEY", "test-key"),
		"BLAXEL_WORKSPACE": getEnvOrDefault("BLAXEL_WORKSPACE", "test-workspace"),
	}
	return env
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
