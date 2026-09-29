package modelapis

import "github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"

// Output schemas for the model API tools. They describe both the JSON this
// package's SDK handler returns and the shapes the hosted control plane
// handler returns.

func modelSpecProperties() map[string]any {
	properties := tools.RuntimeSpecProperties()
	runtime := properties["runtime"].(map[string]any)
	runtimeProperties := runtime["properties"].(map[string]any)
	runtimeProperties["model"] = tools.StringSchema("Upstream model name")
	runtimeProperties["endpointName"] = tools.StringSchema("Upstream endpoint name")
	return properties
}

func modelAPISummarySchema(description string) map[string]any {
	properties := tools.ResourceProperties(modelSpecProperties())
	properties["name"] = tools.StringSchema("Model API name")
	properties["model"] = tools.AnySchema("Upstream model name, or the created model API resource")
	properties["endpoint"] = tools.StringSchema("Upstream endpoint")
	properties["provider"] = tools.StringSchema("Provider, when the integration was created inline")
	properties["integrationConnection"] = tools.StringSchema("Integration connection the model API uses")
	return tools.NestedObjectSchema(description, properties)
}

func listModelAPIsOutputSchema() map[string]any {
	return tools.ListSchema("Model APIs in the workspace",
		tools.NestedResourceSchema("Model API", modelSpecProperties()), nil)
}

func getModelAPIOutputSchema() map[string]any {
	return tools.ResourceSchema("Model API definition and deployment status", modelSpecProperties())
}

func createModelAPIOutputSchema() map[string]any {
	return tools.MutationSchema("Result of creating the model API", map[string]any{
		"model_api": modelAPISummarySchema("Created model API summary"),
		"model":     modelAPISummarySchema("Created model API"),
	})
}

func deleteModelAPIOutputSchema() map[string]any {
	return tools.MutationSchema("Result of deleting the model API", nil)
}
