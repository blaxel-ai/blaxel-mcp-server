package config

import (
	"os"
	"strings"
)

// Config holds the configuration for the MCP server
type Config struct {
	APIEndpoint string
	RunEndpoint string
	Workspace   string

	// Server configuration
	ReadOnly bool
	Debug    bool
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Check for BL_ENV to determine environment (like the CLI does)
	env := os.Getenv("BL_ENV")

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
		APIEndpoint: getEnvOrDefault("BL_API_ENDPOINT", apiEndpoint),
		RunEndpoint: getEnvOrDefault("BL_RUN_SERVER", runEndpoint),
		Workspace:   os.Getenv("BL_WORKSPACE"),
		Debug:       os.Getenv("BL_DEBUG") == "true",
		ReadOnly:    os.Getenv("BL_READ_ONLY") == "true",
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

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
