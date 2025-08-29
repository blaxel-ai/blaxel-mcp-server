package formatter

import (
	"fmt"
	"strings"
	"time"
)

// FormatAgents formats a list of agent models into a readable string
func FormatAgents(agents []AgentModel) string {
	if len(agents) == 0 {
		return "No agents found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d agent(s):\n\n", len(agents)))

	for i, agent := range agents {
		b.WriteString(fmt.Sprintf("Agent #%d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Name: %s\n", agent.Name))

		if len(agent.Labels) > 0 {
			b.WriteString(fmt.Sprintf("  Labels: %v\n", formatLabels(agent.Labels)))
		}

		if agent.Status != "" {
			b.WriteString(fmt.Sprintf("  Status: %s\n", agent.Status))
		}

		if agent.Image != nil {
			b.WriteString(fmt.Sprintf("  Image: %s\n", *agent.Image))
		}

		if agent.Generation != nil {
			b.WriteString(fmt.Sprintf("  Generation: %s\n", *agent.Generation))
		}

		if agent.Memory != nil {
			b.WriteString(fmt.Sprintf("  Memory: %dMB\n", *agent.Memory))
		}

		if agent.MaxTasks != nil {
			b.WriteString(fmt.Sprintf("  Max Concurrent Tasks: %d\n", *agent.MaxTasks))
		}

		if agent.CreatedAt != nil {
			b.WriteString(fmt.Sprintf("  Created: %s\n", agent.CreatedAt.Format(time.RFC3339)))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// FormatJobs formats a list of job models into a readable string
func FormatJobs(jobs []JobModel) string {
	if len(jobs) == 0 {
		return "No jobs found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d job(s):\n\n", len(jobs)))

	for i, job := range jobs {
		b.WriteString(fmt.Sprintf("Job #%d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Name: %s\n", job.Name))

		if len(job.Labels) > 0 {
			b.WriteString(fmt.Sprintf("  Labels: %v\n", formatLabels(job.Labels)))
		}

		if job.Status != "" {
			b.WriteString(fmt.Sprintf("  Status: %s\n", job.Status))
		}

		if job.Image != nil {
			b.WriteString(fmt.Sprintf("  Image: %s\n", *job.Image))
		}

		if job.Memory != nil {
			b.WriteString(fmt.Sprintf("  Memory: %dMB\n", *job.Memory))
		}

		if job.MaxTasks != nil {
			b.WriteString(fmt.Sprintf("  Max Concurrent Tasks: %d\n", *job.MaxTasks))
		}

		if job.MaxRetries != nil {
			b.WriteString(fmt.Sprintf("  Max Retries: %d\n", *job.MaxRetries))
		}

		if job.CreatedAt != nil {
			b.WriteString(fmt.Sprintf("  Created: %s\n", job.CreatedAt.Format(time.RFC3339)))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// FormatModels formats a list of model API models into a readable string
func FormatModels(models []ModelAPI) string {
	if len(models) == 0 {
		return "No model APIs found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d model API(s):\n\n", len(models)))

	for i, model := range models {
		b.WriteString(fmt.Sprintf("Model API #%d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Name: %s\n", model.Name))

		if len(model.Labels) > 0 {
			b.WriteString(fmt.Sprintf("  Labels: %v\n", formatLabels(model.Labels)))
		}

		if model.Status != "" {
			b.WriteString(fmt.Sprintf("  Status: %s\n", model.Status))
		}

		if model.Type != nil {
			b.WriteString(fmt.Sprintf("  Type: %s\n", *model.Type))
		}

		if model.ModelName != nil {
			b.WriteString(fmt.Sprintf("  Model: %s\n", *model.ModelName))
		}

		if model.Memory != nil {
			b.WriteString(fmt.Sprintf("  Memory: %dMB\n", *model.Memory))
		}

		if model.CreatedAt != nil {
			b.WriteString(fmt.Sprintf("  Created: %s\n", model.CreatedAt.Format(time.RFC3339)))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// FormatFunctions formats a list of function/MCP server models into a readable string
func FormatFunctions(functions []FunctionModel) string {
	if len(functions) == 0 {
		return "No MCP servers found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d MCP server(s):\n\n", len(functions)))

	for i, function := range functions {
		b.WriteString(fmt.Sprintf("MCP Server #%d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Name: %s\n", function.Name))

		if len(function.Labels) > 0 {
			b.WriteString(fmt.Sprintf("  Labels: %v\n", formatLabels(function.Labels)))
		}

		if function.Status != "" {
			b.WriteString(fmt.Sprintf("  Status: %s\n", function.Status))
		}

		if function.Image != nil {
			b.WriteString(fmt.Sprintf("  Image: %s\n", *function.Image))
		}

		if function.Generation != nil {
			b.WriteString(fmt.Sprintf("  Generation: %s\n", *function.Generation))
		}

		if function.Memory != nil {
			b.WriteString(fmt.Sprintf("  Memory: %dMB\n", *function.Memory))
		}

		if len(function.IntegrationConnections) > 0 {
			b.WriteString(fmt.Sprintf("  Integration Connections: %s\n", strings.Join(function.IntegrationConnections, ", ")))
		}

		if function.CreatedAt != nil {
			b.WriteString(fmt.Sprintf("  Created: %s\n", function.CreatedAt.Format(time.RFC3339)))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// FormatSandboxes formats a list of sandbox models into a readable string
func FormatSandboxes(sandboxes []SandboxModel) string {
	if len(sandboxes) == 0 {
		return "No sandboxes found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d sandbox(es):\n\n", len(sandboxes)))

	for i, sandbox := range sandboxes {
		b.WriteString(fmt.Sprintf("Sandbox #%d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Name: %s\n", sandbox.Name))

		if len(sandbox.Labels) > 0 {
			b.WriteString(fmt.Sprintf("  Labels: %v\n", formatLabels(sandbox.Labels)))
		}

		if sandbox.Status != "" {
			b.WriteString(fmt.Sprintf("  Status: %s\n", sandbox.Status))
		}

		if sandbox.Image != nil {
			b.WriteString(fmt.Sprintf("  Image: %s\n", *sandbox.Image))
		}

		if sandbox.Generation != nil {
			b.WriteString(fmt.Sprintf("  Generation: %s\n", *sandbox.Generation))
		}

		if sandbox.Memory != nil {
			b.WriteString(fmt.Sprintf("  Memory: %dMB\n", *sandbox.Memory))
		}

		if sandbox.TTL != nil {
			b.WriteString(fmt.Sprintf("  TTL: %s\n", *sandbox.TTL))
		}

		if sandbox.Expires != nil {
			b.WriteString(fmt.Sprintf("  Expires: %s\n", sandbox.Expires.Format(time.RFC3339)))
		}

		if len(sandbox.Ports) > 0 {
			b.WriteString(fmt.Sprintf("  Ports: %v\n", sandbox.Ports))
		}

		if sandbox.CreatedAt != nil {
			b.WriteString(fmt.Sprintf("  Created: %s\n", sandbox.CreatedAt.Format(time.RFC3339)))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// FormatIntegrations formats a list of integration models into a readable string
func FormatIntegrations(integrations []IntegrationModel) string {
	if len(integrations) == 0 {
		return "No integrations found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d integration(s):\n\n", len(integrations)))

	for i, integration := range integrations {
		b.WriteString(fmt.Sprintf("Integration #%d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Name: %s\n", integration.Name))

		if len(integration.Labels) > 0 {
			b.WriteString(fmt.Sprintf("  Labels: %v\n", formatLabels(integration.Labels)))
		}

		if len(integration.Secrets) > 0 {
			b.WriteString(fmt.Sprintf("  Secrets: %v\n", formatLabels(integration.Secrets)))
		}

		if len(integration.Config) > 0 {
			b.WriteString(fmt.Sprintf("  Config: %v\n", formatLabels(integration.Config)))
		}

		if integration.CreatedAt != nil {
			b.WriteString(fmt.Sprintf("  Created: %s\n", integration.CreatedAt.Format(time.RFC3339)))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// FormatUsers formats a list of user models into a readable string
func FormatUsers(users []UserModel) string {
	if len(users) == 0 {
		return "No users found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d user(s):\n\n", len(users)))

	for i, user := range users {
		b.WriteString(fmt.Sprintf("User #%d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Email: %s\n", user.Email))
		b.WriteString(fmt.Sprintf("  Name: %s\n", user.Name))
		b.WriteString(fmt.Sprintf("  Role: %s\n", user.Role))
		b.WriteString(fmt.Sprintf("  Accepted: %t\n", user.Accepted))
		b.WriteString(fmt.Sprintf("  Email Verified: %t\n", user.EmailVerified))
		b.WriteString("\n")
	}

	return b.String()
}

// FormatServiceAccounts formats a list of service account models into a readable string
func FormatServiceAccounts(serviceAccounts []ServiceAccountModel) string {
	if len(serviceAccounts) == 0 {
		return "No service accounts found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d service account(s):\n\n", len(serviceAccounts)))

	for i, sa := range serviceAccounts {
		b.WriteString(fmt.Sprintf("Service Account #%d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Name: %s\n", sa.Name))
		b.WriteString(fmt.Sprintf("  Client ID: %s\n", sa.ClientID))
		b.WriteString(fmt.Sprintf("  Description: %s\n", sa.Description))

		if sa.CreatedAt != nil {
			b.WriteString(fmt.Sprintf("  Created: %s\n", sa.CreatedAt.Format(time.RFC3339)))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// FormatTemplates formats a list of template models into a readable string
func FormatTemplates(templates []TemplateModel) string {
	if len(templates) == 0 {
		return "No templates found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d template(s):\n\n", len(templates)))

	for i, template := range templates {
		b.WriteString(fmt.Sprintf("Template #%d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Name: %s\n", template.Name))

		if template.Description != nil {
			b.WriteString(fmt.Sprintf("  Description: %s\n", *template.Description))
		}

		if len(template.Topics) > 0 {
			b.WriteString(fmt.Sprintf("  Topics: %s\n", strings.Join(template.Topics, ", ")))
		}

		if template.StarCount != nil {
			b.WriteString(fmt.Sprintf("  Stars: %d\n", *template.StarCount))
		}

		if template.DownloadCount != nil {
			b.WriteString(fmt.Sprintf("  Downloads: %d\n", *template.DownloadCount))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// Helper function to format labels
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return "{}"
	}

	var pairs []string
	for k, v := range labels {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}
