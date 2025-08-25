# Blaxel MCP Server (Go)

A Model Context Protocol (MCP) server for Blaxel, written in Go using the [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) library. This server provides tools for managing Blaxel resources including agents, model APIs, MCP servers, sandboxes, jobs, integrations, users, and service accounts.

## Architecture

This implementation follows the same pattern as the [GitHub MCP Server](https://github.com/github/github-mcp-server), converting all resources to tools for better client compatibility.

## Quick Start

```bash
# Build the server
make build

# Run the server
export BLAXEL_API_KEY="your-api-key"
export BLAXEL_WORKSPACE="your-workspace"
./build/blaxel-mcp-server

# Test the server
./test_mcp.sh
```

## Current Status

**Note**: This is a complete migration from TypeScript to Go, following the GitHub MCP server pattern. All resources have been converted to tools for better client compatibility.

### Implemented Features:
- ✅ Basic MCP server infrastructure
- ✅ Complete tool registration system
- ✅ Configuration via environment variables
- ✅ Read-only mode support
- ✅ Toolset filtering

### Tool Implementation Status:
- ✅ **Agents**: List, get, and delete operations working with SDK
- ✅ **Model APIs**: Simplified creation with automatic integration setup
- ✅ **MCP Servers**: Simplified creation with automatic integration setup
- ✅ **Integrations**: Full CRUD operations using SDK's `IntegrationConnection` API
- ✅ **Local Tools**: All local development tools functional
- ⚠️ **Sandboxes**: Placeholder implementation (API not available in SDK)
- ⚠️ **Jobs**: Placeholder implementation (API not available in SDK)
- ⚠️ **Users**: Placeholder implementation (API not available in SDK)
- ⚠️ **Service Accounts**: Placeholder implementation (API not available in SDK)

Note: Some tools return placeholder responses as the corresponding APIs are not yet available in the Blaxel SDK

## Features

- **Complete Resource Management**: All Blaxel resources are exposed as tools instead of MCP resources for better client compatibility
- **Read-Only Mode**: Support for running in read-only mode to prevent destructive operations
- **Toolset Filtering**: Ability to enable/disable specific toolsets
- **Local Development Tools**: Tools for creating and deploying Blaxel projects locally
- **Configuration via Environment Variables**: Flexible configuration options

## Installation

### Prerequisites

- Go 1.21 or later
- Blaxel CLI installed (`npm install -g @blaxel/cli`)
- Valid Blaxel credentials

### Building from Source

```bash
# Clone the repository
git clone https://github.com/blaxel-ai/blaxel-mcp-server.git
cd blaxel-mcp-server

# Install dependencies
go mod download

# Build the binary
go build -o blaxel-mcp-server ./cmd/blaxel-mcp-server

# Run the server
./blaxel-mcp-server
```

## Configuration

### Environment Variables

```bash
# Authentication (choose one method)
export BLAXEL_API_KEY="your-api-key"        # API Key authentication
export BLAXEL_ACCESS_TOKEN="your-token"     # Access token authentication

# Optional configuration
export BLAXEL_WORKSPACE="your-workspace"    # Target workspace
export BLAXEL_API_ENDPOINT="https://api.blaxel.ai"  # API endpoint (default: https://api.blaxel.ai)
export BLAXEL_DEBUG="true"                  # Enable debug logging
export BLAXEL_READ_ONLY="true"              # Run in read-only mode
```

### Command Line Flags

```bash
# Show version information
./blaxel-mcp-server --version

# Run in read-only mode
./blaxel-mcp-server --read-only

# Enable specific toolsets
./blaxel-mcp-server --toolsets agents,modelapis,integrations

# Enable all toolsets (default)
./blaxel-mcp-server --toolsets all
```

## Available Tools

### Agent Management
- `list_agents` - List all agents in the workspace
- `get_agent` - Get details of a specific agent
- `delete_agent` - Delete an agent by name

### Model API Management
- `list_model_apis` - List all model APIs
- `get_model_api` - Get details of a specific model API
- `create_model_api` - **Dual Mode!** Create a model API
  - **Mode 1**: Provide `provider`, `apiKey` to create new integration automatically
  - **Mode 2**: Provide `integrationConnectionName` to use existing integration
  - Flexible approach for different use cases
- `delete_model_api` - Delete a model API

### MCP Server Management
- `list_mcp_servers` - List all MCP servers (functions)
- `get_mcp_server` - Get details of a specific MCP server
- `create_mcp_server` - **Dual Mode!** Create an MCP server
  - **Mode 1**: Provide `integrationType`, `secret`, `config` to create new integration
  - **Mode 2**: Provide `integrationConnectionName` to use existing integration
  - Flexible approach for different use cases
- `delete_mcp_server` - Delete an MCP server

### Sandbox Management
- `list_sandboxes` - List all sandboxes
- `get_sandbox` - Get details of a specific sandbox
- `delete_sandbox` - Delete a sandbox

### Job Management
- `list_jobs` - List all jobs
- `get_job` - Get details of a specific job
- `delete_job` - Delete a job by ID

### Integration Management
- `list_integrations` - List all integration connections
- `get_integration` - Get details of a specific integration
- `create_mcp_integration` - Create an MCP integration
- `create_model_api_integration` - Create a model API integration
- `delete_integration` - Delete an integration
- `list_mcp_integrations` - List available MCP Hub integrations
- `get_mcp_integration` - Get MCP Hub integration details
- `list_integration_models` - List models for an integration

### User Management
- `list_users` - List all users in the workspace
- `invite_user` - Invite a user to the workspace
- `delete_user` - Remove a user from the workspace

### Service Account Management
- `list_service_accounts` - List all service accounts
- `create_service_account` - Create a new service account
- `delete_service_account` - Delete a service account

### Local Development Tools
- `local_create_agent` - Create a new agent project locally
- `local_create_job` - Create a new job project locally
- `local_create_mcp_server` - Create a new MCP server project locally
- `local_create_sandbox` - Create a new sandbox project locally
- `local_deploy_directory` - Deploy a local directory to Blaxel
- `local_run_deployed_resource` - Run a deployed resource
- `local_list_templates` - List available templates
- `local_quick_start_guide` - Get quick start guide

## Simplified Tool Usage

### Key Improvements

**Integration with Blaxel SDK**: The MCP server now fully utilizes the Blaxel SDK's `CreateIntegrationConnection` API, enabling actual integration creation rather than simulated operations. This means:

- **Real Integration Creation**: When you create an MCP server or Model API, the integration is actually created in Blaxel
- **Automatic Linking**: Resources are automatically linked to their integrations
- **Error Recovery**: If resource creation fails after integration creation, appropriate error messages guide recovery
- **Flexible Integration Options**: Support for both creating new integrations and reusing existing ones

### Creating Resources - Two Flexible Modes

Both MCP servers and Model APIs now support two modes of creation:

#### Mode 1: Create with New Integration (One-Step)

Create a resource and its integration in a single tool call:

**MCP Server with New Integration:**
```json
{
  "tool": "create_mcp_server",
  "arguments": {
    "name": "my-github-mcp",
    "integrationType": "github",
    "secret": {
      "token": "ghp_..."
    },
    "config": {
      "owner": "my-org"
    }
  }
}
```

**Model API with New Integration:**
```json
{
  "tool": "create_model_api",
  "arguments": {
    "name": "my-gpt4-api",
    "provider": "openai",
    "apiKey": "sk-...",
    "model": "gpt-4",
    "endpoint": "https://api.openai.com/v1"  // optional
  }
}
```

#### Mode 2: Create with Existing Integration

Reuse an existing integration connection:

**MCP Server with Existing Integration:**
```json
{
  "tool": "create_mcp_server",
  "arguments": {
    "name": "another-github-mcp",
    "integrationConnectionName": "my-github-integration"
  }
}
```

**Model API with Existing Integration:**
```json
{
  "tool": "create_model_api",
  "arguments": {
    "name": "another-gpt4-api",
    "integrationConnectionName": "my-openai-integration",
    "model": "gpt-4-turbo"  // optional model override
  }
}
```

### Benefits

- **Maximum Flexibility**: Choose whether to create new integrations or reuse existing ones
- **Resource Efficiency**: Multiple resources can share a single integration
- **Simpler for Agents**: AI agents can work with either mode based on context
- **Better Organization**: Integrations can be managed separately from resources

## Usage with Claude Desktop

Add the following to your Claude Desktop configuration (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "blaxel": {
      "command": "/path/to/blaxel-mcp-server",
      "env": {
        "BLAXEL_API_KEY": "${env:BLAXEL_API_KEY}",
        "BLAXEL_WORKSPACE": "${env:BLAXEL_WORKSPACE}"
      }
    }
  }
}
```

## Usage with Cursor

Add the following to your Cursor MCP settings:

```json
{
  "mcpServers": {
    "blaxel": {
      "command": "/path/to/blaxel-mcp-server",
      "args": ["--toolsets", "all"],
      "env": {
        "BLAXEL_API_KEY": "${BLAXEL_API_KEY}",
        "BLAXEL_WORKSPACE": "${BLAXEL_WORKSPACE}"
      }
    }
  }
}
```

## Development

### Project Structure

```
blaxel-mcp-server/
├── cmd/
│   └── blaxel-mcp-server/     # Main application entry point
├── internal/
│   ├── config/                # Configuration management
│   └── tools/                 # Tool implementations
│       ├── agents/
│       ├── modelapis/
│       ├── mcpservers/
│       ├── sandboxes/
│       ├── jobs/
│       ├── integrations/
│       ├── users/
│       ├── serviceaccounts/
│       └── local/
└── pkg/
    └── mcp/                   # MCP server implementation
```

### Running Tests

```bash
go test ./...
```

### Building for Different Platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o blaxel-mcp-server-linux ./cmd/blaxel-mcp-server

# macOS
GOOS=darwin GOARCH=amd64 go build -o blaxel-mcp-server-darwin ./cmd/blaxel-mcp-server

# Windows
GOOS=windows GOARCH=amd64 go build -o blaxel-mcp-server.exe ./cmd/blaxel-mcp-server
```

## Troubleshooting

### Authentication Issues

If you encounter authentication errors:
1. Ensure your API key or access token is valid
2. Check that the workspace name is correct
3. Verify the API endpoint is reachable

### Tool Availability

If certain tools are not available:
1. Check if you're running in read-only mode
2. Verify the toolsets configuration
3. Ensure you have the necessary permissions in your workspace

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please submit pull requests with:
- Clear commit messages
- Updated tests
- Documentation updates

## Support

For issues and questions:
- Open an issue on GitHub
- Contact Blaxel support
- Check the [Blaxel documentation](https://docs.blaxel.ai)
