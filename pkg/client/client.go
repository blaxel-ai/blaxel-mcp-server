package client

import (
	"fmt"
	"runtime"

	blaxel "github.com/blaxel-ai/sdk-go"
	"github.com/blaxel-ai/sdk-go/option"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
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
	// Build user agent like the CLI
	osName := runtime.GOOS
	arch := runtime.GOARCH
	version := "mcp-server/1.0.0"

	// Create client using the toolkit's method (like the CLI does)
	sdkClient, err := sdk.NewClientWithCredentials(
		sdk.RunClientWithCredentials{
			ApiURL:      cfg.APIEndpoint,
			RunURL:      cfg.RunEndpoint,
			Credentials: cfg.Credentials,
			Workspace:   cfg.Workspace, // Use the resolved workspace
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

// NewBlaxelClient creates a new blaxel.Client using the official Go SDK.
// This is the preferred client for operations that need metadata URL support
// (e.g., sandbox RunWithMetadata).
func NewBlaxelClient(cfg *config.Config) (*blaxel.Client, error) {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	version := "mcp-server/1.0.0"

	// Initialize the new SDK's environment so RunURL is set correctly
	blaxel.InitializeEnvironment(cfg.Workspace)
	// Override with explicit endpoints from config if they differ from defaults
	if cfg.RunEndpoint != "" {
		blaxel.SetRunURL(cfg.RunEndpoint)
	}

	opts := []option.RequestOption{
		option.WithBaseURL(cfg.APIEndpoint),
		option.WithWorkspace(cfg.Workspace),
		option.WithHeader("User-Agent", fmt.Sprintf("blaxel-mcp/%s (%s/%s)", version, osName, arch)),
	}

	// Set authentication based on available credentials
	if cfg.Credentials.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.Credentials.APIKey))
	} else if cfg.Credentials.AccessToken != "" {
		opts = append(opts, option.WithAccessToken(cfg.Credentials.AccessToken))
		if cfg.Credentials.RefreshToken != "" {
			opts = append(opts, option.WithRefreshToken(cfg.Credentials.RefreshToken))
		}
	} else if cfg.Credentials.ClientCredentials != "" {
		opts = append(opts, option.WithClientCredentials(cfg.Credentials.ClientCredentials))
	}

	client := blaxel.NewClient(opts...)
	return &client, nil
}
