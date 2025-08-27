package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/blaxel-ai/toolkit/sdk"
)

// Config holds the configuration for the MCP server
type Config struct {
	APIEndpoint string
	RunEndpoint string
	Workspace   string
	Env         string
	Credentials sdk.Credentials
	// Server configuration
	ReadOnly bool
	Debug    bool
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Check for BL_ENV to determine environment (like the CLI does)
	// Build credentials from config
	var credentials sdk.Credentials
	workspace := os.Getenv("BL_WORKSPACE")

	// If no workspace specified in environment, use the current context from CLI config
	if workspace == "" {
		currentContext := sdk.CurrentContext()
		workspace = currentContext.Workspace
	}

	if workspace == "" {
		return nil, fmt.Errorf("no workspace found")
	}

	credentials = sdk.LoadCredentials(workspace)
	env := sdk.LoadEnv(workspace)

	// If still no valid credentials, try to use env vars directly
	if !credentials.IsValid() {
		if apiKey := os.Getenv("BL_API_KEY"); apiKey != "" {
			credentials.APIKey = apiKey
		}
	}

	if !credentials.IsValid() {
		return nil, fmt.Errorf("no valid Blaxel credentials found (check BL_API_KEY or run 'bl login')")
	}

	// Default endpoints
	apiEndpoint := "https://api.blaxel.ai/v0"
	runEndpoint := "https://run.blaxel.ai"

	// Adjust endpoints based on environment
	switch env {
	case "dev":
		apiEndpoint = "https://api.blaxel.dev/v0"
		runEndpoint = "https://run.blaxel.dev"
	}

	cfg := &Config{
		APIEndpoint: apiEndpoint,
		RunEndpoint: runEndpoint,
		Workspace:   workspace,
		Env:         env,
		Debug:       os.Getenv("BL_DEBUG") == "true",
		ReadOnly:    os.Getenv("BL_READ_ONLY") == "true",
		Credentials: credentials,
	}

	return cfg, nil
}

// ParseToolsets parses a comma-separated list of toolsets
func ParseToolsets(toolsets string) map[string]bool {
	result := make(map[string]bool)
	for _, toolset := range strings.Split(toolsets, ",") {
		toolset = strings.TrimSpace(toolset)
		if toolset != "" {
			result[toolset] = true
		}
	}
	return result
}
