package agents

import (
	"fmt"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/client"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all agent-related tools using SDK client
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Initialize SDK client
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		// Log error but continue - tools will return errors when called
		fmt.Printf("Warning: Failed to initialize SDK client: %v\n", err)
	}

	// Create SDK-based handler
	handler := NewSDKAgentHandler(sdkClient, cfg.ReadOnly)

	// Register tools using shared definitions
	RegisterAgentTools(s, handler)
}
