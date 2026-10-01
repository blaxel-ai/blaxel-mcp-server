package sandboxes

import "github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"

// Output schemas for the sandbox tools. They describe both the JSON this
// package's SDK handler returns and the shapes the hosted control plane
// handler returns.

func sandboxProperties() map[string]any {
	spec := tools.RuntimeSpecProperties()
	runtime := spec["runtime"].(map[string]any)
	runtimeProperties := runtime["properties"].(map[string]any)
	runtimeProperties["ports"] = tools.ArraySchema("Exposed ports", tools.NestedObjectSchema("Port", nil))
	runtimeProperties["expires"] = tools.StringSchema("Expiration timestamp")
	runtimeProperties["ttl"] = tools.StringSchema("Time to live")
	spec["region"] = tools.StringSchema("Region the sandbox runs in")
	spec["volumes"] = tools.ArraySchema("Attached volumes", tools.NestedObjectSchema("Volume attachment", nil))
	spec["lifecycle"] = tools.NestedObjectSchema("Lifecycle policies", nil)

	properties := tools.ResourceProperties(spec)
	properties["lastUsedAt"] = tools.StringSchema("When the sandbox was last used")
	return properties
}

// ListSandboxesOutputSchema is the output schema of list_sandboxes. It is
// exported so a server that replaces the list_sandboxes registration, such as
// one with its own pagination, can keep declaring the same schema.
func ListSandboxesOutputSchema() map[string]any {
	return tools.ListSchema("Sandboxes in the workspace",
		tools.NestedObjectSchema("Sandbox", sandboxProperties()), nil)
}

func getSandboxOutputSchema() map[string]any {
	return tools.ObjectSchema("Sandbox definition and current status. Secret environment values are masked.", sandboxProperties())
}

func createSandboxOutputSchema() map[string]any {
	created := sandboxProperties()
	created["name"] = tools.StringSchema("Sandbox name")
	return tools.MutationSchema("Result of creating the sandbox", map[string]any{
		"sandbox": tools.NestedObjectSchema("Created sandbox", created),
	})
}

func deleteSandboxOutputSchema() map[string]any {
	return tools.MutationSchema("Result of deleting the sandbox", nil)
}
