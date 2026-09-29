package serviceaccounts

import "github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"

// Output schemas for the service account tools. They describe both the JSON
// this package's SDK handler returns and the shapes the hosted control plane
// handler returns. Read paths never return a client secret; the create result
// carries it exactly as the text result does.

func serviceAccountProperties() map[string]any {
	return map[string]any{
		"client_id":     tools.StringSchema("Client ID that identifies the service account"),
		"clientId":      tools.StringSchema("Client ID that identifies the service account"),
		"name":          tools.StringSchema("Service account name"),
		"description":   tools.StringSchema("Service account description"),
		"workspace":     tools.StringSchema("Workspace that owns the service account"),
		"client_secret": tools.StringSchema("Client secret. Read and update results mask it; only the create result can carry the one-time value."),
		"redirect_uris": tools.ArraySchema("OAuth redirect URIs", tools.StringSchema("")),
		"created_at":    tools.StringSchema("Creation timestamp"),
		"updated_at":    tools.StringSchema("Last update timestamp"),
		"createdAt":     tools.StringSchema("Creation timestamp"),
		"updatedAt":     tools.StringSchema("Last update timestamp"),
		"createdBy":     tools.StringSchema("Creator identifier"),
		"updatedBy":     tools.StringSchema("Last updater identifier"),
	}
}

func serviceAccountSchema(description string) map[string]any {
	return tools.NestedObjectSchema(description, serviceAccountProperties())
}

func serviceAccountMutationSchema(description string) map[string]any {
	return tools.MutationSchema(description, map[string]any{
		"service_account": serviceAccountSchema("Service account"),
		"serviceAccount":  serviceAccountSchema("Service account"),
	})
}

func listServiceAccountsOutputSchema() map[string]any {
	return tools.ListSchema("Service accounts in the workspace. Client secrets are never included.",
		serviceAccountSchema("Service account"), nil)
}

func getServiceAccountOutputSchema() map[string]any {
	return tools.ObjectSchema("Service account. The client secret is never returned in plaintext.", serviceAccountProperties())
}

func createServiceAccountOutputSchema() map[string]any {
	return serviceAccountMutationSchema("Result of creating the service account, with its client ID and the client secret shown only in this response")
}

func deleteServiceAccountOutputSchema() map[string]any {
	return serviceAccountMutationSchema("Result of deleting the service account")
}

func updateServiceAccountOutputSchema() map[string]any {
	return serviceAccountMutationSchema("Result of updating the service account")
}
