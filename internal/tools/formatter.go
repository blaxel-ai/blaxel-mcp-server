package tools

import (
	"fmt"
	"strings"

	"github.com/blaxel-ai/toolkit/sdk"
)

// FormatAgents formats a list of agents into a readable string
func FormatAgents(agents []sdk.Agent) string {
	if len(agents) == 0 {
		return "No agents found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d agent(s):\n\n", len(agents)))

	for i, agent := range agents {
		b.WriteString(fmt.Sprintf("Agent #%d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Name: %s\n", getStringValue(agent.Metadata.Name)))

		if agent.Metadata.Labels != nil && len(*agent.Metadata.Labels) > 0 {
			b.WriteString(fmt.Sprintf("  Labels: %v\n", formatLabels(*agent.Metadata.Labels)))
		}

		if agent.Status != nil {
			b.WriteString(fmt.Sprintf("  Status: %s\n", *agent.Status))
		}

		if agent.Spec.Runtime != nil {
			if agent.Spec.Runtime.Image != nil {
				b.WriteString(fmt.Sprintf("  Image: %s\n", *agent.Spec.Runtime.Image))
			}
			if agent.Spec.Runtime.Generation != nil {
				b.WriteString(fmt.Sprintf("  Generation: %s\n", *agent.Spec.Runtime.Generation))
			}
			if agent.Spec.Runtime.Memory != nil {
				b.WriteString(fmt.Sprintf("  Memory: %dMB\n", *agent.Spec.Runtime.Memory))
			}
			if agent.Spec.Runtime.MaxConcurrentTasks != nil {
				b.WriteString(fmt.Sprintf("  Max Concurrent Tasks: %d\n", *agent.Spec.Runtime.MaxConcurrentTasks))
			}
		}

		if agent.Metadata.CreatedAt != nil {
			b.WriteString(fmt.Sprintf("  Created: %s\n", *agent.Metadata.CreatedAt))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// FormatJobs formats a list of jobs into a readable string
func FormatJobs(jobs []sdk.Job) string {
	if len(jobs) == 0 {
		return "No jobs found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d job(s):\n\n", len(jobs)))

	for i, job := range jobs {
		b.WriteString(fmt.Sprintf("Job #%d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Name: %s\n", getStringValue(job.Metadata.Name)))

		if job.Metadata.Labels != nil && len(*job.Metadata.Labels) > 0 {
			b.WriteString(fmt.Sprintf("  Labels: %v\n", formatLabels(*job.Metadata.Labels)))
		}

		if job.Status != nil {
			b.WriteString(fmt.Sprintf("  Status: %s\n", *job.Status))
		}

		// Job spec doesn't have direct schedule field in SDK

		if job.Spec.Runtime != nil {
			if job.Spec.Runtime.Image != nil {
				b.WriteString(fmt.Sprintf("  Image: %s\n", *job.Spec.Runtime.Image))
			}
			if job.Spec.Runtime.Memory != nil {
				b.WriteString(fmt.Sprintf("  Memory: %dMB\n", *job.Spec.Runtime.Memory))
			}
			if job.Spec.Runtime.MaxConcurrentTasks != nil {
				b.WriteString(fmt.Sprintf("  Max Concurrent Tasks: %d\n", *job.Spec.Runtime.MaxConcurrentTasks))
			}
			if job.Spec.Runtime.MaxRetries != nil {
				b.WriteString(fmt.Sprintf("  Max Retries: %d\n", *job.Spec.Runtime.MaxRetries))
			}
		}

		if job.Metadata.CreatedAt != nil {
			b.WriteString(fmt.Sprintf("  Created: %s\n", *job.Metadata.CreatedAt))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// FormatModels formats a list of models into a readable string
func FormatModels(models []sdk.Model) string {
	if len(models) == 0 {
		return "No models found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d model(s):\n\n", len(models)))

	for i, model := range models {
		b.WriteString(fmt.Sprintf("Model #%d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Name: %s\n", getStringValue(model.Metadata.Name)))

		if model.Metadata.Labels != nil && len(*model.Metadata.Labels) > 0 {
			b.WriteString(fmt.Sprintf("  Labels: %v\n", formatLabels(*model.Metadata.Labels)))
		}

		if model.Status != nil {
			b.WriteString(fmt.Sprintf("  Status: %s\n", *model.Status))
		}

		if model.Spec.Runtime != nil {
			if model.Spec.Runtime.Type != nil {
				b.WriteString(fmt.Sprintf("  Type: %s\n", *model.Spec.Runtime.Type))
			}
			if model.Spec.Runtime.Model != nil {
				b.WriteString(fmt.Sprintf("  Model: %s\n", *model.Spec.Runtime.Model))
			}
			if model.Spec.Runtime.Memory != nil {
				b.WriteString(fmt.Sprintf("  Memory: %dMB\n", *model.Spec.Runtime.Memory))
			}
		}

		if model.Metadata.CreatedAt != nil {
			b.WriteString(fmt.Sprintf("  Created: %s\n", *model.Metadata.CreatedAt))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// FormatFunctions formats a list of functions (MCP servers) into a readable string
func FormatFunctions(functions []sdk.Function) string {
	if len(functions) == 0 {
		return "No MCP servers found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d MCP server(s):\n\n", len(functions)))

	for i, fn := range functions {
		b.WriteString(fmt.Sprintf("MCP Server #%d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Name: %s\n", getStringValue(fn.Metadata.Name)))

		if fn.Metadata.Labels != nil && len(*fn.Metadata.Labels) > 0 {
			b.WriteString(fmt.Sprintf("  Labels: %v\n", formatLabels(*fn.Metadata.Labels)))
		}

		if fn.Status != nil {
			b.WriteString(fmt.Sprintf("  Status: %s\n", *fn.Status))
		}

		if fn.Spec.Runtime != nil {
			if fn.Spec.Runtime.Image != nil {
				b.WriteString(fmt.Sprintf("  Image: %s\n", *fn.Spec.Runtime.Image))
			}
			if fn.Spec.Runtime.Memory != nil {
				b.WriteString(fmt.Sprintf("  Memory: %dMB\n", *fn.Spec.Runtime.Memory))
			}
			if fn.Spec.Runtime.Generation != nil {
				b.WriteString(fmt.Sprintf("  Generation: %s\n", *fn.Spec.Runtime.Generation))
			}
		}
		if fn.Spec.IntegrationConnections != nil {
			b.WriteString(fmt.Sprintf("  Integration Connections: %v\n", *fn.Spec.IntegrationConnections))
		}
		if fn.Metadata.CreatedAt != nil {
			b.WriteString(fmt.Sprintf("  Created: %s\n", *fn.Metadata.CreatedAt))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// FormatSandboxes formats a list of sandboxes into a readable string
func FormatSandboxes(sandboxes []sdk.Sandbox) string {
	if len(sandboxes) == 0 {
		return "No sandboxes found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d sandbox(es):\n\n", len(sandboxes)))

	for i, sandbox := range sandboxes {
		b.WriteString(fmt.Sprintf("Sandbox #%d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Name: %s\n", getStringValue(sandbox.Metadata.Name)))

		if sandbox.Metadata.Labels != nil && len(*sandbox.Metadata.Labels) > 0 {
			b.WriteString(fmt.Sprintf("  Labels: %v\n", formatLabels(*sandbox.Metadata.Labels)))
		}

		if sandbox.Status != nil {
			b.WriteString(fmt.Sprintf("  Status: %s\n", *sandbox.Status))
		}

		if sandbox.Spec.Runtime != nil {
			if sandbox.Spec.Runtime.Image != nil {
				b.WriteString(fmt.Sprintf("  Image: %s\n", *sandbox.Spec.Runtime.Image))
			}
			if sandbox.Spec.Runtime.Memory != nil {
				b.WriteString(fmt.Sprintf("  Memory: %dMB\n", *sandbox.Spec.Runtime.Memory))
			}
			if sandbox.Spec.Runtime.Generation != nil {
				b.WriteString(fmt.Sprintf("  Generation: %s\n", *sandbox.Spec.Runtime.Generation))
			}
			if sandbox.Spec.Runtime.Ttl != nil {
				b.WriteString(fmt.Sprintf("  TTL: %s\n", *sandbox.Spec.Runtime.Ttl))
			}
			if sandbox.Spec.Runtime.Expires != nil {
				b.WriteString(fmt.Sprintf("  Expires: %s\n", *sandbox.Spec.Runtime.Expires))
			}
			if sandbox.Spec.Runtime.Ports != nil {
				b.WriteString(fmt.Sprintf("  Ports: %v\n", *sandbox.Spec.Runtime.Ports))
			}
		}

		if sandbox.Metadata.CreatedAt != nil {
			b.WriteString(fmt.Sprintf("  Created: %s\n", *sandbox.Metadata.CreatedAt))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// FormatIntegrations formats a list of integrations into a readable string
func FormatIntegrations(integrations []sdk.IntegrationConnection) string {
	if len(integrations) == 0 {
		return "No integrations found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d integration(s):\n\n", len(integrations)))

	for i, integration := range integrations {
		b.WriteString(fmt.Sprintf("Integration #%d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Name: %s\n", getStringValue(integration.Metadata.Name)))

		if integration.Metadata.Labels != nil && len(*integration.Metadata.Labels) > 0 {
			b.WriteString(fmt.Sprintf("  Labels: %v\n", formatLabels(*integration.Metadata.Labels)))
		}

		// Integration spec may have different fields

		// Don't display secrets, but indicate if they exist
		if integration.Spec.Secret != nil && len(*integration.Spec.Secret) > 0 {
			b.WriteString(fmt.Sprintf("  Secrets: %d configured\n", len(*integration.Spec.Secret)))
		}

		if integration.Spec.Config != nil && len(*integration.Spec.Config) > 0 {
			b.WriteString(fmt.Sprintf("  Config: %d setting(s)\n", len(*integration.Spec.Config)))
		}

		if integration.Metadata.CreatedAt != nil {
			b.WriteString(fmt.Sprintf("  Created: %s\n", *integration.Metadata.CreatedAt))
		}

		b.WriteString("\n")
	}

	return b.String()
}

// FormatUsers formats a list of workspace users into a readable string
func FormatUsers(users []sdk.WorkspaceUser) string {
	if len(users) == 0 {
		return "No users found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d user(s):\n\n", len(users)))

	for i, _ := range users {
		b.WriteString(fmt.Sprintf("User #%d:\n", i+1))
		// TODO: Add user field formatting when SDK types are confirmed
		b.WriteString("\n")
	}

	return b.String()
}

// Helper functions

func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return "none"
	}

	var pairs []string
	for k, v := range labels {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, ", ")
}

func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
