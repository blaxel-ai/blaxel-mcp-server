package agents

import "github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"

// Output schemas for the agent tools. They describe both the JSON this
// package's SDK handler returns and the shapes the hosted control plane
// handler returns, so the same tool definitions serve both.

func listAgentsOutputSchema() map[string]any {
	return tools.ListSchema("Agents in the workspace",
		tools.NestedResourceSchema("Agent", tools.RuntimeSpecProperties()), nil)
}

func getAgentOutputSchema() map[string]any {
	return tools.ResourceSchema("Agent definition and deployment status. Secret environment values are masked.", tools.RuntimeSpecProperties())
}

func deleteAgentOutputSchema() map[string]any {
	return tools.MutationSchema("Result of deleting the agent", nil)
}
