package config

import (
	"os"
	"strings"
)

// Config holds the configuration for the MCP server
type Config struct {
	// API configuration
	APIKey      string
	APIEndpoint string
	Workspace   string

	// Server configuration
	ReadOnly bool
	Debug    bool

	// Authentication
	AccessToken string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		APIKey:      os.Getenv("BLAXEL_API_KEY"),
		APIEndpoint: getEnvOrDefault("BLAXEL_API_ENDPOINT", "https://api.blaxel.ai"),
		Workspace:   os.Getenv("BLAXEL_WORKSPACE"),
		AccessToken: os.Getenv("BLAXEL_ACCESS_TOKEN"),
		Debug:       os.Getenv("BLAXEL_DEBUG") == "true",
		ReadOnly:    os.Getenv("BLAXEL_READ_ONLY") == "true",
	}

	// Check for legacy environment variables
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("BL_API_KEY")
	}
	if cfg.Workspace == "" {
		cfg.Workspace = os.Getenv("BL_WORKSPACE")
	}
	if cfg.AccessToken == "" {
		cfg.AccessToken = os.Getenv("BL_ACCESS_TOKEN")
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
