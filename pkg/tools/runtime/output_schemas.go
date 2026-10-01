package runtime

import (
	"strings"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// jobTriggeredPrefix introduces the execution response in run_job's text
// result from the SDK handler. Structured content carries the response alone.
const jobTriggeredPrefix = "Job triggered successfully:\n"

// Output schemas for the runtime tools. Agent, job and model responses are
// whatever the invoked resource returns, so they are wrapped as {format, body}.
// Sandbox process responses come from the sandbox process API and are passed
// through as objects (arrays are wrapped as items).

func runAgentOutputSchema() map[string]any {
	return tools.BodySchema("Response returned by the agent")
}

func runJobOutputSchema() map[string]any {
	return tools.BodySchema("Execution response returned when the job was triggered")
}

func runModelOutputSchema() map[string]any {
	return tools.BodySchema("Response returned by the model API, for example a chat completion")
}

func processProperties() map[string]any {
	return map[string]any{
		"pid":              tools.StringSchema("Process ID"),
		"name":             tools.StringSchema("Process name"),
		"command":          tools.StringSchema("Command the process runs"),
		"status":           tools.StringSchema("Process status, for example running, completed, failed, killed or stopped"),
		"exitCode":         tools.IntegerSchema("Exit code, once the process has exited"),
		"workingDir":       tools.StringSchema("Working directory"),
		"startedAt":        tools.StringSchema("When the process started"),
		"completedAt":      tools.StringSchema("When the process completed"),
		"logs":             tools.StringSchema("Combined output, when the process was awaited"),
		"stdout":           tools.StringSchema("Standard output, when the process was awaited"),
		"stderr":           tools.StringSchema("Standard error, when the process was awaited"),
		"keepAlive":        tools.BooleanSchema("Whether the process keeps the sandbox awake"),
		"restartOnFailure": tools.BooleanSchema("Whether the process restarts after a failure"),
		"maxRestarts":      tools.IntegerSchema("Maximum restart count"),
		"restartCount":     tools.IntegerSchema("Restarts so far"),
		tools.TextKey:      tools.StringSchema("Raw response, returned when the sandbox did not answer with JSON"),
	}
}

func runSandboxCommandOutputSchema() map[string]any {
	return tools.ObjectSchema("Process started by the command", processProperties())
}

func listSandboxProcessesOutputSchema() map[string]any {
	return tools.ListSchema("Processes in the sandbox",
		tools.NestedObjectSchema("Sandbox process", processProperties()), nil)
}

func getSandboxProcessOutputSchema() map[string]any {
	return tools.ObjectSchema("Sandbox process status and metadata", processProperties())
}

func getSandboxProcessLogsOutputSchema() map[string]any {
	return tools.ObjectSchema("Process logs", map[string]any{
		"logs":        tools.StringSchema("Combined stdout and stderr"),
		"stdout":      tools.StringSchema("Standard output"),
		"stderr":      tools.StringSchema("Standard error"),
		tools.TextKey: tools.StringSchema("Raw response, returned when the sandbox did not answer with JSON"),
		"tailLines":   tools.IntegerSchema("Number of trailing lines kept per stream when tail trimmed the logs"),
		"truncated":   tools.BooleanSchema("True when tail removed earlier log lines"),
	})
}

func processSignalOutputSchema(description string) map[string]any {
	properties := processProperties()
	properties["message"] = tools.StringSchema("Outcome reported by the sandbox")
	properties["path"] = tools.StringSchema("Request path the sandbox handled")
	return tools.ObjectSchema(description, properties)
}

func stopSandboxProcessOutputSchema() map[string]any {
	return processSignalOutputSchema("Result of stopping the process")
}

func killSandboxProcessOutputSchema() map[string]any {
	return processSignalOutputSchema("Result of killing the process")
}

// runJobResult keeps run_job's text unchanged and structures the execution
// response without the SDK handler's leading sentence.
func runJobResult(result string) *mcp.CallToolResult {
	return mcp.NewToolResultStructured(tools.StructureBody(strings.TrimPrefix(result, jobTriggeredPrefix)), result)
}

// tailStructuredLogs applies tail to each log stream in structured logs
// content, so the structured result is trimmed the same way as the text.
func tailStructuredLogs(structured any, tail int) any {
	content, ok := structured.(map[string]any)
	if !ok {
		return structured
	}
	trimmed := make(map[string]any, len(content)+2)
	truncated := false
	for key, value := range content {
		trimmed[key] = value
		text, isString := value.(string)
		if !isString {
			continue
		}
		switch key {
		case "logs", "stdout", "stderr", tools.TextKey:
		default:
			continue
		}
		lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
		if len(lines) > tail {
			trimmed[key] = strings.Join(lines[len(lines)-tail:], "\n")
			truncated = true
		}
	}
	if truncated {
		trimmed["tailLines"] = tail
		trimmed["truncated"] = true
	}
	return trimmed
}
