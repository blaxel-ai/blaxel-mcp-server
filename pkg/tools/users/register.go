package users

import (
	"fmt"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all user-related tools using SDK client
func RegisterTools(s *server.MCPServer, cfg *config.Config) {
	// Create SDK-based handler
	handler, err := NewSDKHandler(cfg)
	if err != nil {
		// Log error but continue - tools will return errors when called
		fmt.Printf("Warning: Failed to initialize SDK handler: %v\n", err)
		return
	}

	// Register tools using shared definitions
	RegisterUserTools(s, handler)
}
