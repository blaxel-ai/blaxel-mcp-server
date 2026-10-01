package mcpservers

import "github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"

// Output schemas for the MCP server tools. They describe both the JSON this
// package's SDK handler returns and the shapes the hosted control plane
// handler returns.

func mcpServerSpecProperties() map[string]any {
	return tools.RuntimeSpecProperties()
}

func mcpServerSummarySchema(description string) map[string]any {
	properties := tools.ResourceProperties(mcpServerSpecProperties())
	properties["name"] = tools.StringSchema("MCP server name")
	properties["integrationConnection"] = tools.StringSchema("Integration connection the server uses")
	properties["integrationType"] = tools.StringSchema("Integration type, when the integration was created inline")
	return tools.NestedObjectSchema(description, properties)
}

func listMCPServersOutputSchema() map[string]any {
	return tools.ListSchema("MCP servers in the workspace",
		tools.NestedResourceSchema("MCP server", mcpServerSpecProperties()), nil)
}

func getMCPServerOutputSchema() map[string]any {
	return tools.ResourceSchema("MCP server definition and deployment status. Secret environment values are masked.", mcpServerSpecProperties())
}

func createMCPServerOutputSchema() map[string]any {
	return tools.MutationSchema("Result of creating the MCP server", map[string]any{
		"mcp_server": mcpServerSummarySchema("Created MCP server summary"),
		"server":     mcpServerSummarySchema("Created MCP server"),
	})
}

func deleteMCPServerOutputSchema() map[string]any {
	return tools.MutationSchema("Result of deleting the MCP server", map[string]any{
		"server": mcpServerSummarySchema("Deleted MCP server"),
	})
}
