package local

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/blaxel-ai/blaxel-mcp-server/internal/config"
	"github.com/blaxel-ai/toolkit/sdk"
)

func localQuickStartGuideHandler(params map[string]interface{}) (string, error) {
	resourceType, _ := params["resourceType"].(string)
	if resourceType == "" {
		resourceType = "all"
	}

	guides := map[string]string{
		"agent": `Quick Start Guide for Agents:

1. Install Blaxel CLI:
   npm install -g @blaxel/cli

2. Create a new agent project:
   bl create-agent-app my-agent -y

3. Navigate to the project:
   cd my-agent

4. Deploy the agent:
   bl deploy

Available templates:
- template-google-adk-py: Google ADK Python template
- template-langgraph-py: LangGraph Python template
- template-pydantic-py: Pydantic Python template
- template-crewai-py: CrewAI Python template
- template-mastra-ts: Mastra TypeScript template
- template-controlflow-py: ControlFlow Python template`,

		"job": `Quick Start Guide for Jobs:

1. Install Blaxel CLI:
   npm install -g @blaxel/cli

2. Create a new job project:
   bl create-job my-job -y

3. Navigate to the project:
   cd my-job

4. Deploy the job:
   bl deploy

Job templates available for batch processing tasks.`,

		"mcp-server": `Quick Start Guide for MCP Servers:

1. Install Blaxel CLI:
   npm install -g @blaxel/cli

2. Create a new MCP server project:
   bl create-function my-mcp-server -y

3. Navigate to the project:
   cd my-mcp-server

4. Deploy the MCP server:
   bl deploy

MCP servers provide tool functions for AI agents.`,

		"sandbox": `Quick Start Guide for Sandboxes:

1. Install Blaxel CLI:
   npm install -g @blaxel/cli

2. Create a new sandbox project:
   bl create-sandbox my-sandbox -y

3. Navigate to the project:
   cd my-sandbox

4. Deploy the sandbox:
   bl deploy

Sandboxes provide isolated environments for running code.`,
	}

	if resourceType == "all" {
		var result strings.Builder
		result.WriteString("Quick Start Guide for All Blaxel Resources:\n\n")
		for _, guide := range guides {
			result.WriteString(guide)
			result.WriteString("\n\n---\n\n")
		}
		return result.String(), nil
	}

	guide, ok := guides[resourceType]
	if !ok {
		return "", fmt.Errorf("unknown resource type: %s", resourceType)
	}

	return guide, nil
}

func localListTemplatesHandler(ctx context.Context, sdkClient *sdk.ClientWithResponses, params map[string]interface{}) (string, error) {
	resourceType, _ := params["resourceType"].(string)
	if resourceType == "" {
		return "", fmt.Errorf("resourceType is required")
	}

	// Try to fetch templates from API
	templates, err := sdkClient.ListTemplatesWithResponse(ctx)
	if err != nil {
		// Fall back to hardcoded list if API fails
		return "", fmt.Errorf("failed to list templates: %w", err)
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

func localCreateAgentHandler(cfg *config.Config, params map[string]interface{}) (string, error) {
	directory, ok := params["directory"].(string)
	if !ok || directory == "" {
		return "", fmt.Errorf("directory is required")
	}

	// Check if directory already exists
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		return "", fmt.Errorf("directory '%s' already exists", directory)
	}

	// Build command
	cmd := exec.Command("bl", "create-agent-app", directory, "-y")

	// Add workspace flag if available
	if cfg.Workspace != "" {
		cmd.Args = append(cmd.Args, "--workspace", cfg.Workspace)
	}

	// Add template if specified
	if template, ok := params["template"].(string); ok && template != "" {
		cmd.Args = append(cmd.Args, "--template", template)
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create agent: %s", string(output))
	}

	return fmt.Sprintf("Successfully created agent app in directory: %s\n\n%s", directory, string(output)), nil
}

func localCreateJobHandler(cfg *config.Config, params map[string]interface{}) (string, error) {
	directory, ok := params["directory"].(string)
	if !ok || directory == "" {
		return "", fmt.Errorf("directory is required")
	}

	// Check if directory already exists
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		return "", fmt.Errorf("directory '%s' already exists", directory)
	}

	// Build command
	cmd := exec.Command("bl", "create-job", directory, "-y")

	// Add workspace flag if available
	if cfg.Workspace != "" {
		cmd.Args = append(cmd.Args, "--workspace", cfg.Workspace)
	}

	// Add template if specified
	if template, ok := params["template"].(string); ok && template != "" {
		cmd.Args = append(cmd.Args, "--template", template)
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create job: %s", string(output))
	}

	return fmt.Sprintf("Successfully created job in directory: %s\n\n%s", directory, string(output)), nil
}

func localCreateMCPServerHandler(cfg *config.Config, params map[string]interface{}) (string, error) {
	directory, ok := params["directory"].(string)
	if !ok || directory == "" {
		return "", fmt.Errorf("directory is required")
	}

	// Check if directory already exists
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		return "", fmt.Errorf("directory '%s' already exists", directory)
	}

	// Build command - use create-function for MCP servers
	cmd := exec.Command("bl", "create-function", directory, "-y")

	// Add workspace flag if available
	if cfg.Workspace != "" {
		cmd.Args = append(cmd.Args, "--workspace", cfg.Workspace)
	}

	// Add template if specified
	if template, ok := params["template"].(string); ok && template != "" {
		cmd.Args = append(cmd.Args, "--template", template)
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create MCP server: %s", string(output))
	}

	return fmt.Sprintf("Successfully created MCP server in directory: %s\n\n%s", directory, string(output)), nil
}

func localCreateSandboxHandler(cfg *config.Config, params map[string]interface{}) (string, error) {
	directory, ok := params["directory"].(string)
	if !ok || directory == "" {
		return "", fmt.Errorf("directory is required")
	}

	// Check if directory already exists
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		return "", fmt.Errorf("directory '%s' already exists", directory)
	}

	// Build command
	cmd := exec.Command("bl", "create-sandbox", directory, "-y")

	// Add workspace flag if available
	if cfg.Workspace != "" {
		cmd.Args = append(cmd.Args, "--workspace", cfg.Workspace)
	}

	// Add template if specified
	if template, ok := params["template"].(string); ok && template != "" {
		cmd.Args = append(cmd.Args, "--template", template)
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create sandbox: %s", string(output))
	}

	return fmt.Sprintf("Successfully created sandbox in directory: %s\n\n%s", directory, string(output)), nil
}

func localDeployHandler(cfg *config.Config, params map[string]interface{}) (string, error) {
	// Determine target directory
	directory, _ := params["directory"].(string)
	targetPath := directory
	if targetPath == "" {
		var err error
		targetPath, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
	} else {
		targetPath = filepath.Join(".", directory)
	}

	// Check if directory exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return "", fmt.Errorf("directory '%s' does not exist", targetPath)
	}

	// Check for blaxel.yaml or blaxel.yml
	blaxelYaml := filepath.Join(targetPath, "blaxel.yaml")
	blaxelYml := filepath.Join(targetPath, "blaxel.yml")
	if _, err := os.Stat(blaxelYaml); os.IsNotExist(err) {
		if _, err := os.Stat(blaxelYml); os.IsNotExist(err) {
			return "", fmt.Errorf("directory '%s' does not appear to be a valid Blaxel project (missing blaxel.yaml)", targetPath)
		}
	}

	// Build command
	cmd := exec.Command("bl", "deploy")

	// Add workspace flag if available
	if cfg.Workspace != "" {
		cmd.Args = append(cmd.Args, "--workspace", cfg.Workspace)
	}

	// Add directory if specified
	if directory != "" {
		cmd.Args = append(cmd.Args, "--directory", directory)
	}

	// Add name if specified
	if name, ok := params["name"].(string); ok && name != "" {
		cmd.Args = append(cmd.Args, "--name", name)
	}

	// Add skip build flag
	if skipBuild, ok := params["skipBuild"].(bool); ok && skipBuild {
		cmd.Args = append(cmd.Args, "--skip-build")
	}

	// Add dry run flag
	if dryRun, ok := params["dryRun"].(bool); ok && dryRun {
		cmd.Args = append(cmd.Args, "--dryrun")
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to deploy: %s", string(output))
	}

	statusMessage := "Successfully deployed"
	if dryRun, ok := params["dryRun"].(bool); ok && dryRun {
		statusMessage = "Dry run completed successfully"
	}

	return fmt.Sprintf("%s from directory: %s\n\n%s", statusMessage, targetPath, string(output)), nil
}

func localRunHandler(cfg *config.Config, params map[string]interface{}) (string, error) {
	resourceType, ok := params["resourceType"].(string)
	if !ok || resourceType == "" {
		return "", fmt.Errorf("resourceType is required")
	}

	resourceName, ok := params["resourceName"].(string)
	if !ok || resourceName == "" {
		return "", fmt.Errorf("resourceName is required")
	}

	// Build command
	cmd := exec.Command("bl", "run", resourceType, resourceName)

	// Add workspace flag if available
	if cfg.Workspace != "" {
		cmd.Args = append(cmd.Args, "--workspace", cfg.Workspace)
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run %s '%s': %s", resourceType, resourceName, string(output))
	}

	return fmt.Sprintf("Successfully ran %s '%s':\n\n%s", resourceType, resourceName, string(output)), nil
}
