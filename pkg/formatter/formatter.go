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
	fmt.Fprintf(&b, "Found %d agent(s):\n\n", len(agents))

	for i, agent := range agents {
		fmt.Fprintf(&b, "Agent #%d:\n", i+1)
		fmt.Fprintf(&b, "  Name: %s\n", agent.Name)

		if len(agent.Labels) > 0 {
			fmt.Fprintf(&b, "  Labels: %v\n", formatLabels(agent.Labels))
		}

		if agent.Status != "" {
			fmt.Fprintf(&b, "  Status: %s\n", agent.Status)
		}

		if agent.Image != nil {
			fmt.Fprintf(&b, "  Image: %s\n", *agent.Image)
		}

		if agent.Generation != nil {
			fmt.Fprintf(&b, "  Generation: %s\n", *agent.Generation)
		}

		if agent.Memory != nil {
			fmt.Fprintf(&b, "  Memory: %dMB\n", *agent.Memory)
		}

		if agent.MaxTasks != nil {
			fmt.Fprintf(&b, "  Max Concurrent Tasks: %d\n", *agent.MaxTasks)
		}

		if agent.CreatedAt != nil {
			fmt.Fprintf(&b, "  Created: %s\n", agent.CreatedAt.Format(time.RFC3339))
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
	fmt.Fprintf(&b, "Found %d job(s):\n\n", len(jobs))

	for i, job := range jobs {
		fmt.Fprintf(&b, "Job #%d:\n", i+1)
		fmt.Fprintf(&b, "  Name: %s\n", job.Name)

		if len(job.Labels) > 0 {
			fmt.Fprintf(&b, "  Labels: %v\n", formatLabels(job.Labels))
		}

		if job.Status != "" {
			fmt.Fprintf(&b, "  Status: %s\n", job.Status)
		}

		if job.Image != nil {
			fmt.Fprintf(&b, "  Image: %s\n", *job.Image)
		}

		if job.Memory != nil {
			fmt.Fprintf(&b, "  Memory: %dMB\n", *job.Memory)
		}

		if job.MaxTasks != nil {
			fmt.Fprintf(&b, "  Max Concurrent Tasks: %d\n", *job.MaxTasks)
		}

		if job.MaxRetries != nil {
			fmt.Fprintf(&b, "  Max Retries: %d\n", *job.MaxRetries)
		}

		if job.CreatedAt != nil {
			fmt.Fprintf(&b, "  Created: %s\n", job.CreatedAt.Format(time.RFC3339))
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
	fmt.Fprintf(&b, "Found %d model API(s):\n\n", len(models))

	for i, model := range models {
		fmt.Fprintf(&b, "Model API #%d:\n", i+1)
		fmt.Fprintf(&b, "  Name: %s\n", model.Name)

		if len(model.Labels) > 0 {
			fmt.Fprintf(&b, "  Labels: %v\n", formatLabels(model.Labels))
		}

		if model.Status != "" {
			fmt.Fprintf(&b, "  Status: %s\n", model.Status)
		}

		if model.Type != nil {
			fmt.Fprintf(&b, "  Type: %s\n", *model.Type)
		}

		if model.ModelName != nil {
			fmt.Fprintf(&b, "  Model: %s\n", *model.ModelName)
		}

		if model.Memory != nil {
			fmt.Fprintf(&b, "  Memory: %dMB\n", *model.Memory)
		}

		if model.CreatedAt != nil {
			fmt.Fprintf(&b, "  Created: %s\n", model.CreatedAt.Format(time.RFC3339))
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
	fmt.Fprintf(&b, "Found %d MCP server(s):\n\n", len(functions))

	for i, function := range functions {
		fmt.Fprintf(&b, "MCP Server #%d:\n", i+1)
		fmt.Fprintf(&b, "  Name: %s\n", function.Name)

		if len(function.Labels) > 0 {
			fmt.Fprintf(&b, "  Labels: %v\n", formatLabels(function.Labels))
		}

		if function.Status != "" {
			fmt.Fprintf(&b, "  Status: %s\n", function.Status)
		}

		if function.Image != nil {
			fmt.Fprintf(&b, "  Image: %s\n", *function.Image)
		}

		if function.Generation != nil {
			fmt.Fprintf(&b, "  Generation: %s\n", *function.Generation)
		}

		if function.Memory != nil {
			fmt.Fprintf(&b, "  Memory: %dMB\n", *function.Memory)
		}

		if len(function.IntegrationConnections) > 0 {
			fmt.Fprintf(&b, "  Integration Connections: %s\n", strings.Join(function.IntegrationConnections, ", "))
		}

		if function.CreatedAt != nil {
			fmt.Fprintf(&b, "  Created: %s\n", function.CreatedAt.Format(time.RFC3339))
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
	fmt.Fprintf(&b, "Found %d sandbox(es):\n\n", len(sandboxes))

	for i, sandbox := range sandboxes {
		fmt.Fprintf(&b, "Sandbox #%d:\n", i+1)
		fmt.Fprintf(&b, "  Name: %s\n", sandbox.Name)

		if len(sandbox.Labels) > 0 {
			fmt.Fprintf(&b, "  Labels: %v\n", formatLabels(sandbox.Labels))
		}

		if sandbox.Status != "" {
			fmt.Fprintf(&b, "  Status: %s\n", sandbox.Status)
		}

		if sandbox.Image != nil {
			fmt.Fprintf(&b, "  Image: %s\n", *sandbox.Image)
		}

		if sandbox.Generation != nil {
			fmt.Fprintf(&b, "  Generation: %s\n", *sandbox.Generation)
		}

		if sandbox.Memory != nil {
			fmt.Fprintf(&b, "  Memory: %dMB\n", *sandbox.Memory)
		}

		if sandbox.TTL != nil {
			fmt.Fprintf(&b, "  TTL: %s\n", *sandbox.TTL)
		}

		if sandbox.Expires != nil {
			fmt.Fprintf(&b, "  Expires: %s\n", sandbox.Expires.Format(time.RFC3339))
		}

		if len(sandbox.Ports) > 0 {
			fmt.Fprintf(&b, "  Ports: %v\n", sandbox.Ports)
		}

		if sandbox.CreatedAt != nil {
			fmt.Fprintf(&b, "  Created: %s\n", sandbox.CreatedAt.Format(time.RFC3339))
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
	fmt.Fprintf(&b, "Found %d integration(s):\n\n", len(integrations))

	for i, integration := range integrations {
		fmt.Fprintf(&b, "Integration #%d:\n", i+1)
		fmt.Fprintf(&b, "  Name: %s\n", integration.Name)

		if len(integration.Labels) > 0 {
			fmt.Fprintf(&b, "  Labels: %v\n", formatLabels(integration.Labels))
		}

		if len(integration.Secrets) > 0 {
			fmt.Fprintf(&b, "  Secrets: %v\n", formatLabels(integration.Secrets))
		}

		if len(integration.Config) > 0 {
			fmt.Fprintf(&b, "  Config: %v\n", formatLabels(integration.Config))
		}

		if integration.CreatedAt != nil {
			fmt.Fprintf(&b, "  Created: %s\n", integration.CreatedAt.Format(time.RFC3339))
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
	fmt.Fprintf(&b, "Found %d user(s):\n\n", len(users))

	for i, user := range users {
		fmt.Fprintf(&b, "User #%d:\n", i+1)
		fmt.Fprintf(&b, "  Email: %s\n", user.Email)
		fmt.Fprintf(&b, "  Name: %s\n", user.Name)
		fmt.Fprintf(&b, "  Role: %s\n", user.Role)
		fmt.Fprintf(&b, "  Accepted: %t\n", user.Accepted)
		fmt.Fprintf(&b, "  Email Verified: %t\n", user.EmailVerified)
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
	fmt.Fprintf(&b, "Found %d service account(s):\n\n", len(serviceAccounts))

	for i, sa := range serviceAccounts {
		fmt.Fprintf(&b, "Service Account #%d:\n", i+1)
		fmt.Fprintf(&b, "  Name: %s\n", sa.Name)
		fmt.Fprintf(&b, "  Client ID: %s\n", sa.ClientID)
		fmt.Fprintf(&b, "  Description: %s\n", sa.Description)

		if sa.CreatedAt != nil {
			fmt.Fprintf(&b, "  Created: %s\n", sa.CreatedAt.Format(time.RFC3339))
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
	fmt.Fprintf(&b, "Found %d template(s):\n\n", len(templates))

	for i, template := range templates {
		fmt.Fprintf(&b, "Template #%d:\n", i+1)
		fmt.Fprintf(&b, "  Name: %s\n", template.Name)

		if template.Description != nil {
			fmt.Fprintf(&b, "  Description: %s\n", *template.Description)
		}

		if len(template.Topics) > 0 {
			fmt.Fprintf(&b, "  Topics: %s\n", strings.Join(template.Topics, ", "))
		}

		if template.StarCount != nil {
			fmt.Fprintf(&b, "  Stars: %d\n", *template.StarCount)
		}

		if template.DownloadCount != nil {
			fmt.Fprintf(&b, "  Downloads: %d\n", *template.DownloadCount)
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
