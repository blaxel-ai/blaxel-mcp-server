package client

import (
	"fmt"
	"os"
	"runtime"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/blaxel-ai/toolkit/sdk"
)

// ServiceAccount represents a workspace service account (for responses that use inline structs)
type ServiceAccount struct {
	ClientId     *string `json:"client_id,omitempty"`
	ClientSecret *string `json:"client_secret,omitempty"` // Only returned on creation
	CreatedAt    *string `json:"created_at,omitempty"`
	Description  *string `json:"description,omitempty"`
	Name         *string `json:"name,omitempty"`
	UpdatedAt    *string `json:"updated_at,omitempty"`
}

// NewSDKClient creates a new SDK ClientWithResponses using the toolkit approach
// This mimics how the CLI initializes its client
func NewSDKClient(cfg *config.Config) (*sdk.ClientWithResponses, error) {
	// Build credentials from config
	var credentials sdk.Credentials
	workspace := cfg.Workspace

	// If no workspace specified in environment, use the current context from CLI config
	if workspace == "" {
		currentContext := sdk.CurrentContext()
		workspace = currentContext.Workspace
	}

	// Load credentials from the .blaxel config (like the CLI does)
	if workspace != "" {
		credentials = sdk.LoadCredentials(workspace)
		// Update the config with the workspace from context
		if cfg.Workspace == "" {
			cfg.Workspace = workspace
		}

		// Also get the environment from workspace config if BL_ENV is not set
		if os.Getenv("BL_ENV") == "" {
			env := sdk.LoadEnv(workspace)
			// Adjust endpoints based on workspace environment
			switch env {
			case "dev":
				if cfg.APIEndpoint == "https://api.blaxel.ai" || cfg.APIEndpoint == "" {
					cfg.APIEndpoint = "https://api.blaxel.dev/v0"
				}
				if cfg.RunEndpoint == "https://run.blaxel.ai" || cfg.RunEndpoint == "" {
					cfg.RunEndpoint = "https://run.blaxel.dev"
				}
			case "local":
				if cfg.APIEndpoint == "https://api.blaxel.ai" || cfg.APIEndpoint == "" {
					cfg.APIEndpoint = "http://localhost:8080/v0"
				}
				if cfg.RunEndpoint == "https://run.blaxel.ai" || cfg.RunEndpoint == "" {
					cfg.RunEndpoint = "https://run.blaxel.dev"
				}
			}
		}
	}

	// If still no valid credentials, try to use env vars directly
	if !credentials.IsValid() {
		if apiKey := os.Getenv("BL_API_KEY"); apiKey != "" {
			credentials.APIKey = apiKey
		}
	}

	if !credentials.IsValid() {
		return nil, fmt.Errorf("no valid Blaxel credentials found (check BL_API_KEY or run 'bl login')")
	}

	// Build user agent like the CLI
	osName := runtime.GOOS
	arch := runtime.GOARCH
	version := "mcp-server/1.0.0"

	// Create client using the toolkit's method (like the CLI does)
	sdkClient, err := sdk.NewClientWithCredentials(
		sdk.RunClientWithCredentials{
			ApiURL:      cfg.APIEndpoint,
			RunURL:      cfg.RunEndpoint,
			Credentials: credentials,
			Workspace:   workspace, // Use the resolved workspace
			Headers: map[string]string{
				"User-Agent": fmt.Sprintf("blaxel-mcp/%s (%s/%s)", version, osName, arch),
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SDK client: %w", err)
	}

	return sdkClient, nil
}
