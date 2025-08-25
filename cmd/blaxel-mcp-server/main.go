package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/tools/agents"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/tools/integrations"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/tools/jobs"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/tools/local"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/tools/mcpservers"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/tools/modelapis"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/tools/sandboxes"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/tools/serviceaccounts"
	"github.com/blaxel-ai/blaxel-mcp-server/internal/tools/users"
	"github.com/mark3labs/mcp-go/server"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Parse command-line flags
	versionFlag := flag.Bool("version", false, "Print version information")
	readOnlyFlag := flag.Bool("read-only", false, "Enable read-only mode")
	toolsetsFlag := flag.String("toolsets", "all", "Comma-separated list of toolsets to enable")
	flag.Parse()

	// Handle version flag
	if *versionFlag {
		fmt.Printf("blaxel-mcp-server version %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override read-only mode from flag if provided
	if *readOnlyFlag {
		cfg.ReadOnly = true
	}

	// Create MCP server
	mcp := server.NewMCPServer(
		"blaxel-mcp-server",
		version,
	)

	// Register tools based on enabled toolsets
	if err := registerTools(mcp, cfg, *toolsetsFlag); err != nil {
		log.Fatalf("Failed to register tools: %v", err)
	}

	// Start server using stdio transport
	log.Printf("Starting Blaxel MCP server version %s", version)
	if err := server.ServeStdio(mcp); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func registerTools(mcp *server.MCPServer, cfg *config.Config, toolsets string) error {
	// Parse toolsets
	enabledToolsets := config.ParseToolsets(toolsets)

	// Register tools based on enabled toolsets
	if enabledToolsets["all"] || enabledToolsets["agents"] {
		agents.RegisterTools(mcp, cfg)
	}

	if enabledToolsets["all"] || enabledToolsets["modelapis"] {
		modelapis.RegisterTools(mcp, cfg)
	}

	if enabledToolsets["all"] || enabledToolsets["mcpservers"] {
		mcpservers.RegisterTools(mcp, cfg)
	}

	if enabledToolsets["all"] || enabledToolsets["sandboxes"] {
		sandboxes.RegisterTools(mcp, cfg)
	}

	if enabledToolsets["all"] || enabledToolsets["jobs"] {
		jobs.RegisterTools(mcp, cfg)
	}

	if enabledToolsets["all"] || enabledToolsets["integrations"] {
		integrations.RegisterTools(mcp, cfg)
	}

	if enabledToolsets["all"] || enabledToolsets["users"] {
		users.RegisterTools(mcp, cfg)
	}

	if enabledToolsets["all"] || enabledToolsets["serviceaccounts"] {
		serviceaccounts.RegisterTools(mcp, cfg)
	}

	if enabledToolsets["all"] || enabledToolsets["local"] {
		local.RegisterTools(mcp, cfg)
	}

	return nil
}
