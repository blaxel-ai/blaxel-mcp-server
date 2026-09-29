package users

import "github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"

// Output schemas for the workspace user tools. They describe both the JSON
// this package's SDK handler returns and the shapes the hosted control plane
// handler returns.

func userProperties() map[string]any {
	return map[string]any{
		"email":          tools.StringSchema("User email address"),
		"sub":            tools.StringSchema("User subject identifier"),
		"name":           tools.StringSchema("Full name"),
		"given_name":     tools.StringSchema("Given name"),
		"family_name":    tools.StringSchema("Family name"),
		"role":           tools.StringSchema("Workspace role, for example admin, member or viewer"),
		"accepted":       tools.BooleanSchema("Whether the user accepted the workspace invitation"),
		"email_verified": tools.BooleanSchema("Whether the email address is verified"),
		"source":         tools.StringSchema("How the user joined the workspace"),
		"expired":        tools.BooleanSchema("Whether a pending invitation has expired"),
		"mfa_enabled":    tools.BooleanSchema("Whether multi-factor authentication is enabled, when the caller may see it"),
	}
}

func userSchema(description string) map[string]any {
	return tools.NestedObjectSchema(description, userProperties())
}

func listWorkspaceUsersOutputSchema() map[string]any {
	return tools.ListSchema("Users in the workspace", userSchema("Workspace user"), map[string]any{
		"users": tools.ArraySchema("Workspace users, after any filter is applied", userSchema("Workspace user")),
	})
}

func getWorkspaceUserOutputSchema() map[string]any {
	properties := userProperties()
	properties["user"] = userSchema("Workspace user")
	return tools.ObjectSchema("Workspace user, returned either as the object itself or under user", properties)
}

func inviteWorkspaceUserOutputSchema() map[string]any {
	return tools.MutationSchema("Result of inviting the user", map[string]any{
		"email": tools.StringSchema("Invited email address"),
		"role":  tools.StringSchema("Role granted once the invitation is accepted"),
		"invitation": tools.NestedObjectSchema("Pending invitation", map[string]any{
			"workspace": tools.StringSchema("Workspace the user is invited to"),
			"email":     tools.StringSchema("Invited email address"),
			"role":      tools.StringSchema("Role granted once the invitation is accepted"),
			"invitedBy": tools.StringSchema("Who sent the invitation"),
			"expiresAt": tools.StringSchema("When the invitation expires"),
		}),
	})
}

func updateWorkspaceUserRoleOutputSchema() map[string]any {
	return tools.MutationSchema("Result of updating the user's role", map[string]any{
		"user": userSchema("Updated workspace user"),
	})
}

func removeWorkspaceUserOutputSchema() map[string]any {
	return tools.MutationSchema("Result of removing the user from the workspace", nil)
}
