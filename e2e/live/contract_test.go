package live

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// toolWorkflowOwner is the expected inventory and intended lane assignment.
// It is not proof of executable coverage; executableWorkflowCoverage is
// enforced by per-workflow successful-call recording and the TestMain gate.
var toolWorkflowOwner = map[string]string{
	"list_agents":                "runtime-fixtures",
	"get_agent":                  "validation-matrix",
	"delete_agent":               "safe-delete-regression",
	"list_model_apis":            "validation-matrix",
	"get_model_api":              "validation-matrix",
	"create_model_api":           "validation-matrix",
	"delete_model_api":           "safe-delete-regression",
	"list_mcp_servers":           "eng-3887-wait-lifecycle",
	"get_mcp_server":             "eng-3887-wait-lifecycle",
	"create_mcp_server":          "eng-3887-wait-lifecycle",
	"delete_mcp_server":          "safe-delete-regression",
	"list_sandboxes":             "sandbox-lifecycle",
	"get_sandbox":                "sandbox-lifecycle",
	"create_sandbox":             "sandbox-lifecycle",
	"delete_sandbox":             "safe-delete-regression",
	"list_jobs":                  "runtime-fixtures",
	"get_job":                    "validation-matrix",
	"delete_job":                 "safe-delete-regression",
	"list_integrations":          "eng-3888-provider-security",
	"get_integration":            "eng-3888-provider-security",
	"create_integration":         "eng-3888-provider-security",
	"delete_integration":         "safe-delete-regression",
	"list_workspace_users":       "eng-3421-disposable-user",
	"get_workspace_user":         "eng-3421-disposable-user",
	"invite_workspace_user":      "eng-3891-invalid-email",
	"update_workspace_user_role": "user-manifest-required",
	"remove_workspace_user":      "eng-3421-disposable-user",
	"list_service_accounts":      "service-account-lifecycle",
	"get_service_account":        "service-account-lifecycle",
	"create_service_account":     "service-account-lifecycle",
	"update_service_account":     "service-account-lifecycle",
	"delete_service_account":     "safe-delete-regression",
	"run_agent":                  "runtime-fixtures",
	"run_job":                    "runtime-fixtures",
	"run_model":                  "validation-matrix",
	"run_sandbox_command":        "sandbox-lifecycle",
	"list_sandbox_processes":     "sandbox-lifecycle",
	"get_sandbox_process":        "sandbox-lifecycle",
	"get_sandbox_process_logs":   "sandbox-lifecycle",
	"stop_sandbox_process":       "sandbox-lifecycle",
	"kill_sandbox_process":       "sandbox-lifecycle",
}

var executedWorkflowOwners sync.Map

var executableWorkflowCoverage = map[string]string{
	"list_agents": "TestReadOnlyToolsSmoke", "get_agent": "TestRuntimeFixtureRequestMapping",
	"delete_agent": "TestRuntimeFixtureRequestMapping", "run_agent": "TestRuntimeFixtureRequestMapping",
	"list_integrations": "TestReadOnlyToolsSmoke", "get_integration": "TestProviderMutationSecurityLane",
	"create_integration": "TestProviderMutationSecurityLane", "delete_integration": "TestProviderMutationSecurityLane",
	"list_jobs": "TestReadOnlyToolsSmoke", "get_job": "TestRuntimeFixtureRequestMapping",
	"delete_job": "TestRuntimeFixtureRequestMapping", "run_job": "TestRuntimeFixtureRequestMapping",
	"list_mcp_servers": "TestReadOnlyToolsSmoke", "get_mcp_server": "TestMCPServerWaitForCompletionLifecycle",
	"create_mcp_server": "TestMCPServerWaitForCompletionLifecycle", "delete_mcp_server": "TestMCPServerWaitForCompletionLifecycle",
	"list_model_apis": "TestReadOnlyToolsSmoke", "get_model_api": "TestModelAPILifecycle",
	"create_model_api": "TestModelAPILifecycle", "delete_model_api": "TestModelAPILifecycle", "run_model": "TestModelAPILifecycle",
	"list_sandboxes": "TestReadOnlyToolsSmoke", "get_sandbox": "TestSandboxProcessLifecycle",
	"create_sandbox": "TestSandboxProcessLifecycle", "delete_sandbox": "TestSandboxProcessLifecycle",
	"run_sandbox_command": "TestSandboxProcessLifecycle", "list_sandbox_processes": "TestSandboxProcessLifecycle",
	"get_sandbox_process": "TestSandboxProcessLifecycle", "get_sandbox_process_logs": "TestSandboxProcessLifecycle",
	"stop_sandbox_process": "TestSandboxProcessLifecycle", "kill_sandbox_process": "TestSandboxProcessLifecycle",
	"list_service_accounts": "TestReadOnlyToolsSmoke", "get_service_account": "TestServiceAccountLifecycle",
	"create_service_account": "TestServiceAccountLifecycle", "update_service_account": "TestServiceAccountLifecycle",
	"delete_service_account": "TestServiceAccountLifecycle",
	"list_workspace_users":   "TestReadOnlyToolsSmoke", "get_workspace_user": "TestDisposableUserRemovalLifecycle",
	"invite_workspace_user": "TestDisposableUserRemovalLifecycle", "update_workspace_user_role": "TestDisposableUserRemovalLifecycle",
	"remove_workspace_user": "TestDisposableUserRemovalLifecycle",
}

func requireMappedWorkflowCalls(t *testing.T, client *strictClient, owner string) {
	t.Helper()
	executedWorkflowOwners.Store(owner, true)
	t.Cleanup(func() {
		var missing []string
		for tool, mappedOwner := range executableWorkflowCoverage {
			if mappedOwner == owner && !client.calledSuccessfully(tool) {
				missing = append(missing, tool)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("%s did not successfully execute mapped tools: %v", owner, missing)
		}
	})
}

func verifyMappedWorkflowsExecuted() error {
	owners := make(map[string]bool)
	for _, owner := range executableWorkflowCoverage {
		owners[owner] = true
	}
	var missing []string
	for owner := range owners {
		if _, ok := executedWorkflowOwners.Load(owner); !ok {
			missing = append(missing, owner)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return fmt.Errorf("mapped executable workflows did not run: %v", missing)
	}
	return nil
}

func TestHostedToolsListContract(t *testing.T) {
	client := newStrictClient(t)
	listed := client.listTools()

	if len(listed.Tools) != 41 {
		t.Fatalf("hosted tools/list returned %d tools, want exactly 41", len(listed.Tools))
	}

	actual := make(map[string]mcp.Tool, len(listed.Tools))
	for _, tool := range listed.Tools {
		if _, duplicate := actual[tool.Name]; duplicate {
			t.Errorf("tools/list returned duplicate tool %q", tool.Name)
		}
		actual[tool.Name] = tool
		if strings.TrimSpace(tool.Title) == "" || strings.TrimSpace(tool.Description) == "" || strings.TrimSpace(tool.Annotations.Title) == "" {
			t.Errorf("tool %q must have nonempty title, description, and annotation title", tool.Name)
		}
		if tool.Annotations.ReadOnlyHint == nil || tool.Annotations.DestructiveHint == nil || tool.Annotations.IdempotentHint == nil || tool.Annotations.OpenWorldHint == nil {
			t.Errorf("tool %q must explicitly set all four annotation hint pointers", tool.Name)
		}
		if strings.HasPrefix(tool.Name, "local_") {
			t.Errorf("hosted toolsets exposed local-only tool %q", tool.Name)
		}
	}

	var missing, unexpected []string
	for name := range toolWorkflowOwner {
		if _, ok := actual[name]; !ok {
			missing = append(missing, name)
		}
	}
	for name := range actual {
		if _, ok := toolWorkflowOwner[name]; !ok {
			unexpected = append(unexpected, name)
		}
	}
	sort.Strings(missing)
	sort.Strings(unexpected)
	if len(missing) > 0 || len(unexpected) > 0 {
		t.Fatalf("hosted tools/list ownership drift: missing=%v unexpected=%v", missing, unexpected)
	}

	assertAnnotationPolicy(t, actual)

	t.Run("run_agent_accepts_body_path_and_message_shorthand", func(t *testing.T) {
		tool := actual["run_agent"]
		for _, field := range []string{"name", "body", "path", "message", "workspace"} {
			if _, ok := tool.InputSchema.Properties[field]; !ok {
				t.Errorf("run_agent schema missing desired %q field", field)
			}
		}
		if !contains(tool.InputSchema.Required, "name") {
			t.Error("run_agent name must remain required")
		}
		if contains(tool.InputSchema.Required, "message") {
			t.Error("run_agent message must be optional shorthand when body is supplied")
		}
	})

	t.Run("update_service_account_uses_exact_REST_aligned_shape", func(t *testing.T) {
		tool := actual["update_service_account"]
		var properties []string
		for name := range tool.InputSchema.Properties {
			properties = append(properties, name)
		}
		sort.Strings(properties)
		wantProperties := []string{"client_id", "name", "workspace"}
		if fmt.Sprint(properties) != fmt.Sprint(wantProperties) {
			t.Errorf("update_service_account properties = %v, want exact %v (no description/new_name aliases)", properties, wantProperties)
		}
		required := append([]string(nil), tool.InputSchema.Required...)
		sort.Strings(required)
		wantRequired := []string{"client_id", "name"}
		if fmt.Sprint(required) != fmt.Sprint(wantRequired) {
			t.Errorf("update_service_account required = %v, want exact %v", required, wantRequired)
		}
	})

	t.Run("create_service_account_requires_explicit_secret_disclosure", func(t *testing.T) {
		tool := actual["create_service_account"]
		var properties []string
		for name := range tool.InputSchema.Properties {
			properties = append(properties, name)
		}
		sort.Strings(properties)
		wantProperties := []string{"name", "revealSecret", "workspace"}
		if fmt.Sprint(properties) != fmt.Sprint(wantProperties) {
			t.Errorf("create_service_account properties = %v, want exact %v", properties, wantProperties)
		}
		if !contains(tool.InputSchema.Required, "name") || contains(tool.InputSchema.Required, "revealSecret") {
			t.Errorf("create_service_account required = %v, want name only", tool.InputSchema.Required)
		}
		revealSecret, ok := tool.InputSchema.Properties["revealSecret"].(map[string]any)
		if !ok || revealSecret["type"] != "boolean" {
			t.Errorf("create_service_account revealSecret schema = %#v, want boolean", tool.InputSchema.Properties["revealSecret"])
		}
		if !strings.Contains(tool.Description, "redacted by default") {
			t.Errorf("create_service_account description must explain the safe default, got %q", tool.Description)
		}
	})
}

func TestMarketplaceReadiness(t *testing.T) {
	client := newStrictClient(t)
	listed := client.listTools()
	for _, tool := range listed.Tools {
		if len(tool.Name) > 64 {
			t.Errorf("marketplace tool name %q is %d bytes, maximum is 64", tool.Name, len(tool.Name))
		}
		verbs := 0
		for _, verb := range []string{"list_", "get_", "create_", "delete_", "invite_", "update_", "remove_", "run_", "stop_", "kill_"} {
			if strings.HasPrefix(tool.Name, verb) {
				verbs++
			}
		}
		if verbs != 1 || strings.Contains(tool.Name, "_and_") || strings.Contains(tool.Name, "_or_") || strings.HasPrefix(tool.Name, "manage_") {
			t.Errorf("marketplace tool %q must expose one explicit safe or unsafe method, not a combined catch-all", tool.Name)
		}
		if _, hasMethodSelector := tool.InputSchema.Properties["method"]; hasMethodSelector {
			t.Errorf("marketplace tool %q exposes a catch-all method selector", tool.Name)
		}
	}
}

func TestDiscoveredToolsHaveExecutableWorkflowOwner(t *testing.T) {
	client := newStrictClient(t)
	listed := client.listTools()
	var uncovered []string
	for _, tool := range listed.Tools {
		if executableWorkflowCoverage[tool.Name] == "" {
			uncovered = append(uncovered, tool.Name)
		}
	}
	sort.Strings(uncovered)
	if len(uncovered) > 0 {
		t.Fatalf("%d discovered tools lack an executable workflow owner: %v", len(uncovered), uncovered)
	}
}

func TestReadOnlyToolsSmoke(t *testing.T) {
	client := newStrictClient(t)
	requireMappedWorkflowCalls(t, client, "TestReadOnlyToolsSmoke")
	for _, tool := range []string{"list_agents", "list_integrations", "list_jobs", "list_mcp_servers", "list_model_apis", "list_sandboxes", "list_service_accounts", "list_workspace_users"} {
		t.Run(tool, func(t *testing.T) {
			client.call(tool, map[string]any{})
		})
	}
}

func TestRunPrefixIsUnused(t *testing.T) {
	client := newStrictClient(t)
	for _, check := range []struct {
		tool string
		args map[string]any
	}{
		{"list_sandboxes", map[string]any{"filter": liveConfig.Prefix}},
		{"list_service_accounts", map[string]any{"filter": liveConfig.Prefix}},
		{"list_mcp_servers", map[string]any{"filter": liveConfig.Prefix}},
		{"list_integrations", map[string]any{"filter": liveConfig.Prefix}},
	} {
		text := client.call(check.tool, check.args)
		if listedResource(text, liveConfig.Prefix) {
			t.Fatalf("BLAXEL_E2E_RUN_PREFIX is not unique: %s found an existing resource", check.tool)
		}
	}
}

func TestPM2609AllDeleteToolsReturnSafeErrors(t *testing.T) {
	client := newStrictClient(t)
	missing := liveConfig.Prefix + "missing"
	rows := []struct {
		tool string
		args map[string]any
	}{
		{"delete_agent", map[string]any{"name": missing + "-agent"}},
		{"delete_integration", map[string]any{"name": missing + "-integration"}},
		{"delete_job", map[string]any{"id": missing + "-job"}},
		{"delete_mcp_server", map[string]any{"name": missing + "-mcp", "waitForCompletion": "true"}},
		{"delete_model_api", map[string]any{"name": missing + "-model", "waitForCompletion": "true"}},
		{"delete_sandbox", map[string]any{"name": missing + "-sandbox"}},
		{"delete_service_account", map[string]any{"name": missing + "-service-account"}},
	}
	for _, row := range rows {
		t.Run(row.tool, func(t *testing.T) {
			result, text := client.callRaw(row.tool, row.args)
			if !result.IsError {
				t.Fatalf("nonexistent delete unexpectedly succeeded: %s", redactedSummary(text))
			}
			lower := strings.ToLower(text)
			if !strings.Contains(lower, "not found") && !strings.Contains(lower, "does not exist") && !strings.Contains(lower, "no such") {
				t.Fatalf("delete error is not an actionable not-found response: %s", redactedSummary(text))
			}
		})
	}
}

func TestStrictToolValidationMatrix(t *testing.T) {
	client := newStrictClient(t)
	listed := client.listTools()
	for _, tool := range listed.Tools {
		tool := tool
		t.Run(tool.Name, func(t *testing.T) {
			result, text := client.callRaw(tool.Name, map[string]any{})
			if len(tool.InputSchema.Required) == 0 {
				if result.IsError {
					t.Fatalf("argument-free live call failed: %s", redactedSummary(text))
				}
				return
			}
			if !result.IsError {
				t.Fatalf("missing required fields %v unexpectedly succeeded", tool.InputSchema.Required)
			}
			lower := strings.ToLower(text)
			for _, forbidden := range []string{"401", "403", "404", "unauthorized", "forbidden", "quota", "rate limit", "not found"} {
				if strings.Contains(lower, forbidden) {
					t.Fatalf("validation escaped to remote/setup failure %q", forbidden)
				}
			}
			if strings.TrimSpace(text) == "" {
				t.Fatal("validation error had no diagnostic")
			}
			if !strings.Contains(lower, "required") && !strings.Contains(lower, "missing") {
				t.Fatalf("expected local required-argument validation, got: %s", redactedSummary(text))
			}
		})
	}
}

func assertAnnotationPolicy(t *testing.T, tools map[string]mcp.Tool) {
	t.Helper()
	for name, tool := range tools {
		if tool.Annotations.ReadOnlyHint == nil || tool.Annotations.DestructiveHint == nil {
			continue
		}
		wantReadOnly := strings.HasPrefix(name, "list_") || strings.HasPrefix(name, "get_")
		wantDestructive := strings.HasPrefix(name, "create_") || strings.HasPrefix(name, "delete_") || strings.HasPrefix(name, "update_") ||
			strings.HasPrefix(name, "invite_") ||
			strings.HasPrefix(name, "remove_") || strings.HasPrefix(name, "run_") ||
			strings.HasPrefix(name, "stop_") || strings.HasPrefix(name, "kill_")
		if got := *tool.Annotations.ReadOnlyHint; got != wantReadOnly {
			t.Errorf("tool %q readOnlyHint = %t, want %t", name, got, wantReadOnly)
		}
		if got := *tool.Annotations.DestructiveHint; got != wantDestructive {
			t.Errorf("tool %q destructiveHint = %t, want %t", name, got, wantDestructive)
		}
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
