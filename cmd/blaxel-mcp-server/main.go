package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/logger"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/agents"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/integrations"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/jobs"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/local"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/mcpservers"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/modelapis"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/runtime"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/sandboxes"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/serviceaccounts"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/users"
	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/server"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Parse command-line flags first
	versionFlag := flag.Bool("version", false, "Print version information")
	readOnlyFlag := flag.Bool("read-only", false, "Enable read-only mode")
	toolsetsFlag := flag.String("toolsets", "all", "Comma-separated list of toolsets to enable")
	transportFlag := flag.String("transport", "stdio", "Transport mode: stdio (default) or http")
	flag.Parse()

	// Handle version flag (before logger init since it doesn't need logging)
	if *versionFlag {
		fmt.Printf("blaxel-mcp-server version %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Initialize logger based on transport mode
	isStdio := *transportFlag == "stdio"
	if err := logger.Init(isStdio); err != nil {
		// Can't use logger here since it failed to init
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Load .env file if it exists (like the CLI does)
	if err := godotenv.Load(); err != nil {
		// Only log error if .env file exists but can't be loaded
		if !os.IsNotExist(err) {
			logger.Warnf("Could not load .env file: %v", err)
		}
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
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
		logger.Fatalf("Failed to register tools: %v", err)
	}

	// Start server based on transport mode
	logger.Printf("Starting Blaxel MCP server version %s (transport: %s)", version, *transportFlag)

	if isStdio {
		// Use stdio transport (default for MCP)
		if err := server.ServeStdio(mcp); err != nil {
			logger.Fatalf("Server error: %v", err)
		}
	} else {
		// Future: could support HTTP or other transports here
		logger.Fatalf("Transport '%s' not yet implemented", *transportFlag)
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

	// Register runtime execution tools (unless in read-only mode)
	if !cfg.ReadOnly && (enabledToolsets["all"] || enabledToolsets["runtime"]) {
		runtime.RegisterTools(mcp, cfg)
	}

	return nil
}
