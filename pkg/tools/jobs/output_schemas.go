package jobs

import "github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"

// Output schemas for the job tools. They describe both the JSON this package's
// SDK handler returns and the shapes the hosted control plane handler returns.

func jobSpecProperties() map[string]any {
	properties := tools.RuntimeSpecProperties()
	properties["triggers"] = tools.ArraySchema("Job triggers", tools.NestedObjectSchema("Trigger", nil))
	return properties
}

func listJobsOutputSchema() map[string]any {
	return tools.ListSchema("Jobs in the workspace",
		tools.NestedResourceSchema("Job", jobSpecProperties()), nil)
}

func getJobOutputSchema() map[string]any {
	return tools.ResourceSchema("Job definition and deployment status. Secret environment values are masked.", jobSpecProperties())
}

func deleteJobOutputSchema() map[string]any {
	return tools.MutationSchema("Result of deleting the job", map[string]any{
		"job": tools.NestedObjectSchema("Deleted job", map[string]any{
			"name":   tools.StringSchema("Job name"),
			"status": tools.StringSchema("Job status after the delete request"),
		}),
	})
}
