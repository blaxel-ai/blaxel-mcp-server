package local

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/client"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/toolkit/sdk"
)

// SDKHandler implements LocalHandler using the SDK client
type SDKHandler struct {
	sdkClient *sdk.ClientWithResponses
	cfg       *config.Config
	readOnly  bool
}

// NewSDKHandler creates a new SDK-based local handler
func NewSDKHandler(cfg *config.Config) (LocalHandler, error) {
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SDK client: %w", err)
	}

	return &SDKHandler{
		sdkClient: sdkClient,
		cfg:       cfg,
		readOnly:  cfg.ReadOnly,
	}, nil
}

// QuickStartGuide implements LocalHandler.QuickStartGuide
func (h *SDKHandler) QuickStartGuide(resourceType string) (string, error) {
	if resourceType == "" {
		resourceType = "all"
	}

	guides := map[string]string{
		"agent": `Quick Start Guide for Agents:

1. Install Blaxel CLI:
   	brew tap blaxel-ai/blaxel
  	brew install blaxel

2. Login:
	  bl login WORKSPACE_NAME

3. Create a new agent project:
   	bl create-agent-app my-agent --template TEMPLATE_NAME -y

4. Navigate to the project:
   	cd my-agent

5. Deploy the agent:
   	bl deploy
`,

		"job": `Quick Start Guide for Jobs:

1. Install Blaxel CLI:
   brew tap blaxel-ai/blaxel
   brew install blaxel

2. Login:
   bl login WORKSPACE_NAME

3. Create a new job project:
   bl create-job my-job --template TEMPLATE_NAME -y

4. Navigate to the project:
   cd my-job

5. Deploy the job:
   bl deploy

Job templates available for batch processing tasks.`,

		"mcp-server": `Quick Start Guide for MCP Servers:

1. Install Blaxel CLI:
   brew tap blaxel-ai/blaxel
   brew install blaxel

2. Login:
   bl login WORKSPACE_NAME

3. Create a new MCP server project:
   bl create-mcp-server my-mcp-server --template TEMPLATE_NAME -y

4. Navigate to the project:
   cd my-mcp-server

5. Deploy the MCP server:
   bl deploy

MCP servers provide tool functions for AI agents.`,

		"sandbox": `Quick Start Guide for Sandboxes:

1. Install Blaxel CLI:
   brew tap blaxel-ai/blaxel
   brew install blaxel

2. Login:
   bl login WORKSPACE_NAME

3. Create a new sandbox project:
   bl create-sandbox my-sandbox -y

4. Navigate to the project:
   cd my-sandbox

5. Deploy the sandbox:
   bl deploy

Sandboxes provide isolated environments for running code.`,
	}

	if resourceType == "all" {
		var allGuides strings.Builder
		allGuides.WriteString("Blaxel Quick Start Guides:\n\n")
		for _, guide := range guides {
			allGuides.WriteString(guide)
			allGuides.WriteString("\n\n")
		}
		return allGuides.String(), nil
	}

	if guide, exists := guides[resourceType]; exists {
		return guide, nil
	}

	return "", fmt.Errorf("unknown resource type: %s", resourceType)
}

// ListTemplates implements LocalHandler.ListTemplates
func (h *SDKHandler) ListTemplates(ctx context.Context, resourceType string) (string, error) {
	if h.sdkClient == nil {
		return "", fmt.Errorf("SDK client not initialized")
	}

	// Try to fetch templates from API
	templates, err := h.sdkClient.ListTemplatesWithResponse(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list templates: %w", err)
	}

	if templates.JSON200 == nil {
		return fmt.Sprintf("No templates found for resource type: %s", resourceType), nil
	}

	// Filter templates based on resource type
	topicKeywords := map[string][]string{
		"agent":      {"agent", "agents", "adk", "langgraph", "pydantic", "crewai", "mastra", "controlflow"},
		"job":        {"job", "jobs", "batch"},
		"sandbox":    {"sandbox", "sandboxes", "vm"},
		"mcp-server": {"mcp", "mcp-server", "function", "functions", "tool", "tools"},
	}

	var result strings.Builder
	if resourceType == "all" {
		result.WriteString("All available templates:\n\n")
	} else {
		result.WriteString(fmt.Sprintf("Available templates for %s:\n\n", resourceType))
	}

	keywords := topicKeywords[resourceType]
	count := 0

	for _, template := range *templates.JSON200 {
		if template.Name == nil {
			continue
		}

		if resourceType != "all" {
			// Check if template matches resource type
			matched := false
			if template.Topics != nil {
				for _, topic := range *template.Topics {
					topicLower := strings.ToLower(topic)
					for _, keyword := range keywords {
						if strings.Contains(topicLower, keyword) {
							matched = true
							break
						}
					}
					if matched {
						break
					}
				}
			}
			if !matched {
				continue
			}
		}

		count++
		result.WriteString(fmt.Sprintf("• %s", *template.Name))
		if template.Description != nil {
			result.WriteString(fmt.Sprintf(" - %s", *template.Description))
		}
		if template.StarCount != nil || template.DownloadCount != nil {
			stars := 0
			downloads := 0
			if template.StarCount != nil {
				stars = *template.StarCount
			}
			if template.DownloadCount != nil {
				downloads = *template.DownloadCount
			}
			result.WriteString(fmt.Sprintf(" (⭐ %d, 📥 %d)", stars, downloads))
		}
		if template.Topics != nil && len(*template.Topics) > 0 {
			result.WriteString(fmt.Sprintf("\n  Topics: %s", strings.Join(*template.Topics, ", ")))
		}
		result.WriteString("\n\n")
	}

	if count == 0 {
		if resourceType == "all" {
			result.WriteString("No templates found.")
		} else {
			result.WriteString(fmt.Sprintf("No templates found for %s.", resourceType))
		}
	} else {
		if resourceType != "all" {
			result.WriteString(fmt.Sprintf("\nUse any of these templates when creating a new %s by specifying the template name.", resourceType))
		}
	}

	return result.String(), nil
}

// CreateAgent implements LocalHandler.CreateAgent
func (h *SDKHandler) CreateAgent(directory, template string) (string, error) {
	args := []string{"create-agent-app", directory}
	if template != "" {
		args = append(args, "--template", template)
	}
	args = append(args, "-y")

	cmd := exec.Command("bl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create agent: %w", err)
	}

	return fmt.Sprintf("Agent created successfully in directory: %s", directory), nil
}

// CreateJob implements LocalHandler.CreateJob
func (h *SDKHandler) CreateJob(directory, template string) (string, error) {
	args := []string{"create-job", directory}
	if template != "" {
		args = append(args, "--template", template)
	}
	args = append(args, "-y")

	cmd := exec.Command("bl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create job: %w", err)
	}

	return fmt.Sprintf("Job created successfully in directory: %s", directory), nil
}

// CreateMCPServer implements LocalHandler.CreateMCPServer
func (h *SDKHandler) CreateMCPServer(directory, template string) (string, error) {
	args := []string{"create-mcp-server", directory}
	if template != "" {
		args = append(args, "--template", template)
	}
	args = append(args, "-y")

	cmd := exec.Command("bl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create MCP server: %w", err)
	}

	return fmt.Sprintf("MCP server created successfully in directory: %s", directory), nil
}

// CreateSandbox implements LocalHandler.CreateSandbox
func (h *SDKHandler) CreateSandbox(directory, template string) (string, error) {
	args := []string{"create-sandbox", directory}
	if template != "" {
		args = append(args, "--template", template)
	}
	args = append(args, "-y")

	cmd := exec.Command("bl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create sandbox: %w", err)
	}

	return fmt.Sprintf("Sandbox created successfully in directory: %s", directory), nil
}

// DeployDirectory implements LocalHandler.DeployDirectory
func (h *SDKHandler) DeployDirectory(directory string) (string, error) {
	if directory == "" {
		// Use current directory if none specified
		var err error
		directory, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Change to the directory
	if err := os.Chdir(directory); err != nil {
		return "", fmt.Errorf("failed to change to directory %s: %w", directory, err)
	}

	// Check if blaxel.json exists
	if _, err := os.Stat("blaxel.json"); os.IsNotExist(err) {
		return "", fmt.Errorf("blaxel.json not found in directory: %s", directory)
	}

	// Run deploy command
	cmd := exec.Command("bl", "deploy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to deploy directory: %w", err)
	}

	return fmt.Sprintf("Directory deployed successfully: %s", directory), nil
}

// RunDeployedResource implements LocalHandler.RunDeployedResource
func (h *SDKHandler) RunDeployedResource(resourceType, resourceName string) (string, error) {
	args := []string{"run", resourceType, resourceName}

	cmd := exec.Command("bl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run deployed resource: %w", err)
	}

	return fmt.Sprintf("Resource %s of type %s started successfully", resourceName, resourceType), nil
}

// IsReadOnly implements LocalHandlerWithReadOnly.IsReadOnly
func (h *SDKHandler) IsReadOnly() bool {
	return h.readOnly
}
