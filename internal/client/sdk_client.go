package client

import (
	"fmt"
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
