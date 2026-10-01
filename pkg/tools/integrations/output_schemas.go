package integrations

import "github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"

// Output schemas for the integration tools. They describe both the JSON this
// package's SDK handler returns and the shapes the hosted control plane
// handler returns. Secret values are always masked in these results.

func integrationProperties() map[string]any {
	return map[string]any{
		"metadata": tools.MetadataSchema(),
		"spec": tools.NestedObjectSchema("Integration connection specification", map[string]any{
			"integration": tools.StringSchema("Integration type, for example github or openai"),
			"config":      tools.StringMapSchema("Non-secret configuration values"),
			"secret":      tools.StringMapSchema("Secret values, masked"),
		}),
	}
}

func listIntegrationsOutputSchema() map[string]any {
	return tools.ListSchema("Integration connections in the workspace. Secret values are masked.",
		tools.NestedObjectSchema("Integration connection", integrationProperties()), nil)
}

func getIntegrationOutputSchema() map[string]any {
	return tools.ObjectSchema("Integration connection. Secret values are masked.", integrationProperties())
}

func integrationSummarySchema(description string) map[string]any {
	properties := integrationProperties()
	properties["name"] = tools.StringSchema("Integration connection name")
	properties["type"] = tools.StringSchema("Integration type")
	return tools.NestedObjectSchema(description, properties)
}

func createIntegrationOutputSchema() map[string]any {
	return tools.MutationSchema("Result of creating the integration connection. Secret values are masked.", map[string]any{
		"integration": integrationSummarySchema("Created integration connection"),
	})
}

func deleteIntegrationOutputSchema() map[string]any {
	return tools.MutationSchema("Result of deleting the integration connection", map[string]any{
		"integration": integrationSummarySchema("Deleted integration connection"),
	})
}
