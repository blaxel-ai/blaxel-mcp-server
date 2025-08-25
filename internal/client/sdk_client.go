package client

import (
	"context"
	"fmt"
	"net/http"

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

// NewSDKClient creates a new SDK ClientWithResponses with authentication configured
func NewSDKClient(cfg *config.Config) (*sdk.ClientWithResponses, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("BLAXEL_API_KEY is required")
	}

	// Create request editor for authentication
	requestEditor := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.APIKey))
		if cfg.Workspace != "" {
			req.Header.Set("x-blaxel-workspace", cfg.Workspace)
		}
		return nil
	}

	// Create SDK client with authentication
	sdkClient, err := sdk.NewClientWithResponses(
		cfg.APIEndpoint, // server URL
		cfg.APIEndpoint, // runServer URL (same as server for now)
		sdk.WithRequestEditorFn(requestEditor),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SDK client: %w", err)
	}

	return sdkClient, nil
}
