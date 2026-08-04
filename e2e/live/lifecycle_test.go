package live

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

type cleanupEntry struct {
	kind     string
	id       string
	released bool
	remove   func() error
}

type cleanupLedger struct {
	t       *testing.T
	client  *strictClient
	entries []*cleanupEntry
}

func newCleanupLedger(t *testing.T, client *strictClient) *cleanupLedger {
	t.Helper()
	ledger := &cleanupLedger{t: t, client: client}
	t.Cleanup(ledger.finish)
	return ledger
}

func (l *cleanupLedger) add(kind, id string, remove func() error) *cleanupEntry {
	entry := &cleanupEntry{kind: kind, id: id, remove: remove}
	l.entries = append(l.entries, entry)
	return entry
}

func (l *cleanupLedger) release(entry *cleanupEntry) {
	entry.released = true
}

func (l *cleanupLedger) finish() {
	for index := len(l.entries) - 1; index >= 0; index-- {
		entry := l.entries[index]
		if entry.released {
			continue
		}
		if err := entry.remove(); err != nil {
			l.t.Errorf("cleanup %s %q failed: %v", entry.kind, entry.id, err)
			continue
		}
		entry.released = true
	}

	// A successful delete response is not enough: every tracked resource
	// collection must prove that this run prefix has no survivors.
	for _, check := range []struct {
		tool string
		args map[string]any
	}{
		{"list_sandboxes", map[string]any{"filter": liveConfig.Prefix}},
		{"list_service_accounts", map[string]any{"filter": liveConfig.Prefix}},
		{"list_mcp_servers", map[string]any{"filter": liveConfig.Prefix}},
		{"list_model_apis", map[string]any{"filter": liveConfig.Prefix}},
		{"list_integrations", map[string]any{"filter": liveConfig.Prefix}},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		result, text, err := l.client.callRawResult(ctx, check.tool, check.args)
		cancel()
		if err != nil {
			l.t.Errorf("final leak scan %s failed: %v", check.tool, err)
			continue
		}
		if result.IsError {
			l.t.Errorf("final leak scan %s failed: %s", check.tool, redactedSummary(text))
			continue
		}
		if listedActiveResource(check.tool, text, liveConfig.Prefix) && !waitForListAbsence(l.t, l.client, check.tool, check.args, liveConfig.Prefix, 45*time.Second) {
			l.t.Errorf("final leak scan found a surviving prefixed resource via %s", check.tool)
		}
	}
}

func listedResource(text, prefix string) bool {
	pattern := `(?m)^\s*Name:\s*` + regexp.QuoteMeta(prefix)
	return regexp.MustCompile(pattern).MatchString(text)
}

func listedActiveResource(tool, text, prefix string) bool {
	if tool != "list_sandboxes" {
		return listedResource(text, prefix)
	}
	// Soft-deleted sandboxes can remain visible as TERMINATED. Cleanup is
	// complete once no matching sandbox remains in active inventory.
	for _, block := range strings.Split(text, "Sandbox #") {
		if listedResource(block, prefix) && !strings.Contains(strings.ToUpper(block), "STATUS: TERMINATED") {
			return true
		}
	}
	return false
}

func waitForListAbsence(t *testing.T, client *strictClient, tool string, args map[string]any, prefix string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		result, text := client.callRaw(tool, args)
		if result.IsError {
			t.Fatalf("%s failed while proving absence: %s", tool, redactedSummary(text))
		}
		if !listedActiveResource(tool, text, prefix) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(time.Second)
	}
}

func isActionableNotFound(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "not found") || strings.Contains(lower, "does not exist") ||
		strings.Contains(lower, "no such") || (strings.Contains(lower, "no ") && strings.Contains(lower, " found"))
}

func cleanupTool(client *strictClient, tool string, args map[string]any) func() error {
	return func() error {
		// Cleanup must not inherit a canceled test context or the client's
		// long-lived acceptance deadline.
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		result, text, err := client.callRawResult(ctx, tool, args)
		if err != nil {
			return err
		}
		if result.IsError {
			return fmt.Errorf("tool error: %s", redactedSummary(text))
		}
		return nil
	}
}

func cleanupServiceAccount(client *strictClient, name string, clientID *string) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		resolvedID := *clientID
		deadline := time.Now().Add(60 * time.Second)
		for resolvedID == "" {
			result, text, err := client.callRawResult(ctx, "list_service_accounts", map[string]any{"filter": name})
			if err != nil {
				return fmt.Errorf("resolve client ID for cleanup: %w", err)
			}
			if result.IsError {
				return fmt.Errorf("resolve client ID for cleanup: %s", redactedSummary(text))
			}
			for _, block := range strings.Split(text, "Service Account #") {
				if strings.Contains(block, "Name: "+name) {
					match := regexp.MustCompile(`(?m)^\s*Client ID:\s*(\S+)\s*$`).FindStringSubmatch(block)
					if len(match) == 2 {
						resolvedID = match[1]
						break
					}
				}
			}
			if resolvedID != "" {
				break
			}
			if !strings.Contains(text, "Name: "+name) && time.Now().After(deadline) {
				return fmt.Errorf("could not prove ambiguous service-account create %q stayed absent for cleanup", name)
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("matching service account exists but its client ID was not recoverable")
			}
			time.Sleep(time.Second)
		}
		result, text, err := client.callRawResult(ctx, "delete_service_account", map[string]any{"name": resolvedID})
		if err != nil {
			return err
		}
		if result.IsError {
			return fmt.Errorf("tool error: %s", redactedSummary(text))
		}
		return nil
	}
}

func cleanupMCPServer(client *strictClient, name string, createdOwned *bool) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
		defer cancel()
		if !*createdOwned {
			deadline := time.Now().Add(60 * time.Second)
			for {
				listed, text, err := client.callRawResult(ctx, "list_mcp_servers", map[string]any{"filter": name})
				if err != nil {
					return err
				}
				if listed.IsError {
					return fmt.Errorf("list before MCP cleanup failed: %s", redactedSummary(text))
				}
				if listedResource(text, name) || time.Now().After(deadline) {
					break
				}
				time.Sleep(time.Second)
			}
		}
		result, text, err := client.callRawResult(ctx, "delete_mcp_server", map[string]any{"name": name, "waitForCompletion": "true"})
		if err != nil {
			return err
		}
		if result.IsError {
			return fmt.Errorf("delete tool error: %s", redactedSummary(text))
		}
		get, getText, err := client.callRawResult(ctx, "get_mcp_server", map[string]any{"name": name})
		if err != nil {
			return err
		}
		if !get.IsError || !isActionableNotFound(getText) || strings.Contains(strings.ToUpper(getText), "DELETING") {
			return fmt.Errorf("waitForCompletion cleanup did not end in actionable not-found: %s", redactedSummary(getText))
		}
		return nil
	}
}

func cleanupModelAPI(client *strictClient, workspace, name string) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
		defer cancel()
		args := map[string]any{"name": name, "waitForCompletion": "true", "workspace": workspace}
		result, text, err := client.callRawResult(ctx, "delete_model_api", args)
		if err != nil {
			return err
		}
		if result.IsError && !isActionableNotFound(text) {
			return fmt.Errorf("delete tool error: %s", redactedSummaryWithSecrets(text, client.sensitiveValues()))
		}

		get, getText, err := client.callRawResult(ctx, "get_model_api", map[string]any{"name": name, "workspace": workspace})
		if err != nil {
			return err
		}
		if !get.IsError || !isActionableNotFound(getText) {
			return fmt.Errorf("model API cleanup did not end in actionable not-found: %s", redactedSummaryWithSecrets(getText, client.sensitiveValues()))
		}
		return cleanupListAbsence(ctx, client, "list_model_apis", workspace, name)
	}
}

func cleanupIntegration(client *strictClient, workspace, name string) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		result, text, err := client.callRawResult(ctx, "delete_integration", map[string]any{"name": name, "workspace": workspace})
		if err != nil {
			return err
		}
		if result.IsError && !isActionableNotFound(text) {
			return fmt.Errorf("delete tool error: %s", redactedSummaryWithSecrets(text, client.sensitiveValues()))
		}

		get, getText, err := client.callRawResult(ctx, "get_integration", map[string]any{"name": name, "workspace": workspace})
		if err != nil {
			return err
		}
		if !get.IsError || !isActionableNotFound(getText) {
			return fmt.Errorf("integration cleanup did not end in actionable not-found: %s", redactedSummaryWithSecrets(getText, client.sensitiveValues()))
		}
		return cleanupListAbsence(ctx, client, "list_integrations", workspace, name)
	}
}

func cleanupListAbsence(ctx context.Context, client *strictClient, tool, workspace, name string) error {
	for {
		result, text, err := client.callRawResult(ctx, tool, map[string]any{"filter": name, "workspace": workspace})
		if err != nil {
			return err
		}
		if result.IsError {
			return fmt.Errorf("%s failed while proving cleanup: %s", tool, redactedSummaryWithSecrets(text, client.sensitiveValues()))
		}
		if !listedResource(text, name) {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s still lists %q after cleanup: %w", tool, name, ctx.Err())
		case <-time.After(time.Second):
		}
	}
}

func TestMCPServerWaitForCompletionLifecycle(t *testing.T) {
	integration := strings.TrimSpace(os.Getenv("BLAXEL_E2E_MCP_SERVER_INTEGRATION"))
	if integration == "" {
		t.Fatal("MCP server lifecycle requires BLAXEL_E2E_MCP_SERVER_INTEGRATION naming an existing deployable MCP integration")
	}
	client := newStrictClient(t)
	requireMappedWorkflowCalls(t, client, "TestMCPServerWaitForCompletionLifecycle")
	ledger := newCleanupLedger(t, client)
	name := liveConfig.Prefix + "-mcp"
	if existing := client.call("list_mcp_servers", map[string]any{"filter": name}); listedResource(existing, name) {
		t.Fatalf("refusing to mutate colliding MCP server %q", name)
	}
	createdOwned := false
	cleanup := ledger.add("MCP server", name, cleanupMCPServer(client, name, &createdOwned))

	created := client.call("create_mcp_server", map[string]any{
		"name": name, "integrationConnectionName": integration, "waitForCompletion": "true",
	})
	createdOwned = true
	if !strings.Contains(created, name) {
		t.Fatal("create_mcp_server did not identify the created server")
	}
	get := client.call("get_mcp_server", map[string]any{"name": name})
	status := regexp.MustCompile(`(?i)"status"\s*:\s*"(DEPLOYED|FAILED|DEPLOYING)"`).FindStringSubmatch(get)
	if len(status) != 2 || (strings.ToUpper(status[1]) != "DEPLOYED" && strings.ToUpper(status[1]) != "FAILED") {
		t.Fatalf("waitForCompletion create followed by immediate get must be terminal DEPLOYED or FAILED, got: %s", redactedSummary(get))
	}

	client.call("delete_mcp_server", map[string]any{"name": name, "waitForCompletion": "true"})
	result, text := client.callRaw("get_mcp_server", map[string]any{"name": name})
	if !result.IsError || !isActionableNotFound(text) || strings.Contains(strings.ToUpper(text), "DELETING") {
		t.Fatalf("waitForCompletion delete followed by immediate get must be actionable not-found, got: %s", redactedSummary(text))
	}
	ledger.release(cleanup)
}

func TestModelAPILifecycle(t *testing.T) {
	if os.Getenv("BLAXEL_E2E_ALLOW_PROVIDER_MUTATIONS") != "true" {
		t.Fatal("model API lifecycle requires BLAXEL_E2E_ALLOW_PROVIDER_MUTATIONS=true")
	}
	openAIKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if openAIKey == "" {
		t.Fatal("model API lifecycle requires OPENAI_API_KEY")
	}

	// The provider key is passed only in the create payload, never inherited by
	// the standalone server. Keep the captured value in memory for this test.
	t.Setenv("OPENAI_API_KEY", "")
	client := newStrictClient(t)
	requireMappedWorkflowCalls(t, client, "TestModelAPILifecycle")
	client.protectSecret(openAIKey) // Register the real provider key before its first use.
	ledger := newCleanupLedger(t, client)
	workspace := liveConfig.Workspace
	modelName := liveConfig.Prefix + "-model"
	integrationName := modelName + "-openai-integration"
	endpointName := liveConfig.Prefix + "-endpoint"

	for _, collision := range []struct {
		tool string
		kind string
	}{
		{"list_integrations", "integration"},
		{"list_model_apis", "model API"},
	} {
		existing := client.call(collision.tool, map[string]any{"filter": liveConfig.Prefix, "workspace": workspace})
		if listedResource(existing, liveConfig.Prefix) {
			t.Fatalf("refusing to mutate workspace %q: unique run prefix collides with an existing %s", workspace, collision.kind)
		}
	}

	// The inline connection name is deterministic and known before mutation, so
	// both resources can be cleanup-ledgered before create_model_api runs.
	integrationCleanup := ledger.add("inline model integration", integrationName, cleanupIntegration(client, workspace, integrationName))
	modelCleanup := ledger.add("model API", modelName, cleanupModelAPI(client, workspace, modelName))
	createdText := client.call("create_model_api", map[string]any{
		"name": modelName, "model": "gpt-4o-mini", "endpoint": endpointName,
		"provider": "openai", "apiKey": openAIKey,
		"config":            map[string]any{"organization": "mcp-live-acceptance"},
		"waitForCompletion": "true", "workspace": workspace,
	})
	created := parseJSONObject(t, createdText)
	createdModel, ok := created["model_api"].(map[string]any)
	if !ok || strings.ToUpper(fmt.Sprint(createdModel["status"])) != "DEPLOYED" {
		t.Fatalf("create_model_api wait did not return DEPLOYED: %s", redactedSummaryWithSecrets(createdText, client.sensitiveValues()))
	}

	getText := client.call("get_model_api", map[string]any{"name": modelName, "workspace": workspace})
	got := parseJSONObject(t, getText)
	if strings.ToUpper(fmt.Sprint(got["status"])) != "DEPLOYED" || findNestedString(got, "spec", "runtime", "endpointName") != endpointName {
		t.Fatalf("get_model_api did not preserve DEPLOYED status and endpoint name: %s", redactedSummaryWithSecrets(getText, client.sensitiveValues()))
	}
	integrationText := client.call("get_integration", map[string]any{"name": integrationName, "workspace": workspace})
	if !strings.Contains(integrationText, "mcp-live-acceptance") {
		t.Fatalf("inline integration did not preserve config: %s", redactedSummaryWithSecrets(integrationText, client.sensitiveValues()))
	}

	runText := client.call("run_model", map[string]any{
		"name": modelName, "path": "/v1/chat/completions", "workspace": workspace,
		"body": map[string]any{
			"messages":    []map[string]string{{"role": "user", "content": "Reply only with OK."}},
			"temperature": 0,
			"max_tokens":  3,
		},
	})
	run := parseJSONObject(t, runText)
	choices, ok := run["choices"].([]any)
	if !ok || len(choices) == 0 {
		t.Fatalf("run_model returned no chat completion choices: %s", redactedSummaryWithSecrets(runText, client.sensitiveValues()))
	}

	client.call("delete_model_api", map[string]any{
		"name": modelName, "waitForCompletion": "true", "workspace": workspace,
	})
	if result, text := client.callRaw("get_model_api", map[string]any{"name": modelName, "workspace": workspace}); !result.IsError || !isActionableNotFound(text) {
		t.Fatalf("deleted model API remained readable: %s", redactedSummaryWithSecrets(text, client.sensitiveValues()))
	}
	if !waitForListAbsence(t, client, "list_model_apis", map[string]any{"filter": modelName, "workspace": workspace}, modelName, 45*time.Second) {
		t.Fatal("deleted model API remains in list_model_apis")
	}
	ledger.release(modelCleanup)

	client.call("delete_integration", map[string]any{"name": integrationName, "workspace": workspace})
	if result, text := client.callRaw("get_integration", map[string]any{"name": integrationName, "workspace": workspace}); !result.IsError || !isActionableNotFound(text) {
		t.Fatalf("deleted integration remained readable: %s", redactedSummaryWithSecrets(text, client.sensitiveValues()))
	}
	if !waitForListAbsence(t, client, "list_integrations", map[string]any{"filter": integrationName, "workspace": workspace}, integrationName, 45*time.Second) {
		t.Fatal("deleted integration remains in list_integrations")
	}
	ledger.release(integrationCleanup)
}

func TestProviderMutationSecurityLane(t *testing.T) {
	if os.Getenv("BLAXEL_E2E_ALLOW_PROVIDER_MUTATIONS") != "true" {
		t.Fatal("ENG-3888 lane requires BLAXEL_E2E_ALLOW_PROVIDER_MUTATIONS=true")
	}
	integrationType := strings.TrimSpace(os.Getenv("BLAXEL_E2E_INTEGRATION_TYPE"))
	if integrationType == "" {
		t.Fatal("ENG-3888 lane requires BLAXEL_E2E_INTEGRATION_TYPE")
	}
	client := newStrictClient(t)
	requireMappedWorkflowCalls(t, client, "TestProviderMutationSecurityLane")
	ledger := newCleanupLedger(t, client)
	name := liveConfig.Prefix + "-integration"
	if existing := client.call("list_integrations", map[string]any{"filter": name}); listedResource(existing, name) {
		t.Fatalf("refusing to mutate colliding integration %q", name)
	}
	cleanup := ledger.add("integration", name, cleanupTool(client, "delete_integration", map[string]any{"name": name}))

	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		t.Fatal("generate in-memory dummy secret")
	}
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)
	client.protectSecret(secret) // Must precede create: its response is security-sensitive too.
	client.call("create_integration", map[string]any{
		"name": name, "integrationType": integrationType, "secret": map[string]any{"apiKey": secret},
	})
	for _, probe := range []struct {
		name string
		text string
	}{
		{"get_integration", client.call("get_integration", map[string]any{"name": name})},
		{"list_integrations", client.call("list_integrations", map[string]any{"filter": name})},
	} {
		// Omitting write-only credentials is safe. If the response includes the
		// submitted key, it must be visibly masked; the dynamic scanner above
		// already rejects the exact plaintext value.
		lower := strings.ToLower(probe.text)
		if strings.Contains(probe.text, "apiKey") && !strings.Contains(probe.text, "*") && !strings.Contains(lower, "redacted") && !strings.Contains(lower, "masked") {
			t.Fatalf("%s exposed an unmasked apiKey field", probe.name)
		}
	}
	client.call("delete_integration", map[string]any{"name": name})
	if result, text := client.callRaw("get_integration", map[string]any{"name": name}); !result.IsError || !isActionableNotFound(text) {
		t.Fatalf("deleted integration immediate get was not actionable not-found: %s", redactedSummary(text))
	}
	ledger.release(cleanup)
}

func TestSandboxProcessLifecycle(t *testing.T) {
	client := newStrictClient(t)
	requireMappedWorkflowCalls(t, client, "TestSandboxProcessLifecycle")
	ledger := newCleanupLedger(t, client)
	sandbox := liveConfig.Prefix + "-sb"
	if existing := client.call("list_sandboxes", map[string]any{"filter": sandbox}); listedResource(existing, sandbox) {
		t.Fatalf("refusing to mutate colliding sandbox %q", sandbox)
	}

	// The name is known before the mutation. Register cleanup first so an
	// ambiguous transport/response failure cannot strand a created sandbox.
	sandboxCleanup := ledger.add("sandbox", sandbox, cleanupTool(client, "delete_sandbox", map[string]any{"name": sandbox}))
	client.call("create_sandbox", map[string]any{"name": sandbox, "memory": 1024})

	get := client.call("get_sandbox", map[string]any{"name": sandbox})
	if !strings.Contains(get, sandbox) {
		t.Fatal("get_sandbox did not return the created sandbox")
	}
	list := client.call("list_sandboxes", map[string]any{"filter": liveConfig.Prefix})
	if !strings.Contains(list, sandbox) {
		t.Fatal("list_sandboxes did not return the created sandbox")
	}

	stopProcess := "stop-" + liveConfig.Prefix
	stopMarker := "stdout-" + liveConfig.Prefix
	client.call("run_sandbox_command", map[string]any{
		"name": sandbox, "processName": stopProcess,
		"command":           "sh -lc 'echo " + stopMarker + "; sleep 300'",
		"waitForCompletion": false,
	})
	stopCleanup := ledger.add("sandbox process", stopProcess, cleanupTool(client, "stop_sandbox_process", map[string]any{"name": sandbox, "identifier": stopProcess}))

	process := client.call("get_sandbox_process", map[string]any{"name": sandbox, "identifier": stopProcess})
	if !strings.Contains(process, stopProcess) {
		t.Fatal("get_sandbox_process did not identify the created process")
	}
	processes := client.call("list_sandbox_processes", map[string]any{"name": sandbox})
	if !strings.Contains(processes, stopProcess) {
		t.Fatal("list_sandbox_processes did not include the created process")
	}
	logs := client.call("get_sandbox_process_logs", map[string]any{"name": sandbox, "identifier": stopProcess})
	if !strings.Contains(logs, stopMarker) {
		t.Fatal("sandbox process logs did not contain the unique stdout marker")
	}
	client.call("stop_sandbox_process", map[string]any{"name": sandbox, "identifier": stopProcess})
	ledger.release(stopCleanup)

	killProcess := "kill-" + liveConfig.Prefix
	client.call("run_sandbox_command", map[string]any{
		"name": sandbox, "processName": killProcess,
		"command":           "sh -lc 'echo kill-ready; sleep 300'",
		"waitForCompletion": false,
	})
	killCleanup := ledger.add("sandbox process", killProcess, cleanupTool(client, "kill_sandbox_process", map[string]any{"name": sandbox, "identifier": killProcess}))
	client.call("get_sandbox_process", map[string]any{"name": sandbox, "identifier": killProcess})
	client.call("kill_sandbox_process", map[string]any{"name": sandbox, "identifier": killProcess})
	ledger.release(killCleanup)

	client.call("delete_sandbox", map[string]any{"name": sandbox})
	if !waitForListAbsence(t, client, "list_sandboxes", map[string]any{"filter": sandbox}, sandbox, 45*time.Second) {
		t.Fatal("deleted sandbox remains visible in active list_sandboxes inventory")
	}
	ledger.release(sandboxCleanup)
}

func requireUserMutationGate(t *testing.T) {
	t.Helper()
	if os.Getenv("BLAXEL_E2E_ALLOW_USER_MUTATIONS") != "true" {
		t.Fatal("user security lane requires BLAXEL_E2E_ALLOW_USER_MUTATIONS=true")
	}
}

func requireUserMutationLane(t *testing.T) string {
	t.Helper()
	requireUserMutationGate(t)
	email := strings.TrimSpace(os.Getenv("BLAXEL_E2E_DISPOSABLE_USER_EMAIL"))
	lower := strings.ToLower(email)
	if !strings.HasPrefix(lower, "mcp-live-") || !strings.HasSuffix(lower, "@example.invalid") {
		t.Fatal("user security lane requires a unique mcp-live-*@example.invalid disposable address")
	}
	return email
}

func userListed(text, email string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(email))
}

func TestDisposableUserRemovalLifecycle(t *testing.T) {
	disposableEmail := requireUserMutationLane(t)
	existingEmail := strings.TrimSpace(os.Getenv("BLAXEL_E2E_EXISTING_USER_EMAIL"))
	existingRole := strings.TrimSpace(os.Getenv("BLAXEL_E2E_EXISTING_USER_ROLE"))
	if existingEmail == "" || existingRole == "" {
		t.Fatal("user lifecycle requires BLAXEL_E2E_EXISTING_USER_EMAIL and BLAXEL_E2E_EXISTING_USER_ROLE for an idempotent role-update proof")
	}

	client := newStrictClient(t)
	requireMappedWorkflowCalls(t, client, "TestDisposableUserRemovalLifecycle")
	ledger := newCleanupLedger(t, client)

	// Prove update_workspace_user_role without changing effective access: set an
	// existing approved fixture to the role it already has, then read it back.
	existing := client.call("get_workspace_user", map[string]any{"email": existingEmail})
	if !userListed(existing, existingEmail) || !regexp.MustCompile(`(?i)"role"\s*:\s*"`+regexp.QuoteMeta(existingRole)+`"`).MatchString(existing) {
		t.Fatal("approved existing user fixture or expected role was not present")
	}
	client.call("update_workspace_user_role", map[string]any{"email": existingEmail, "role": existingRole})
	updated := client.call("get_workspace_user", map[string]any{"email": existingEmail})
	if !regexp.MustCompile(`(?i)"role"\s*:\s*"` + regexp.QuoteMeta(existingRole) + `"`).MatchString(updated) {
		t.Fatal("idempotent user role update was not observable")
	}

	if listed := client.call("list_workspace_users", map[string]any{"filter": disposableEmail}); userListed(listed, disposableEmail) {
		t.Fatalf("refusing to reuse colliding disposable user fixture %q", disposableEmail)
	}
	cleanup := ledger.add("disposable workspace user", disposableEmail, cleanupWorkspaceUser(client, disposableEmail))
	client.call("invite_workspace_user", map[string]any{"email": disposableEmail, "role": "member"})

	deadline := time.Now().Add(45 * time.Second)
	for {
		listed := client.call("list_workspace_users", map[string]any{"filter": disposableEmail})
		if userListed(listed, disposableEmail) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("invited disposable user did not appear in list_workspace_users")
		}
		time.Sleep(time.Second)
	}
	if got := client.call("get_workspace_user", map[string]any{"email": disposableEmail}); !userListed(got, disposableEmail) {
		t.Fatal("invited disposable user was not returned by get_workspace_user")
	}

	missing := liveConfig.Prefix + "-missing@example.invalid"
	result, text := client.callRaw("remove_workspace_user", map[string]any{"email": missing})
	if !result.IsError || !isActionableNotFound(text) {
		t.Fatalf("nonexistent remove must return actionable not-found and never false success: %s", redactedSummary(text))
	}

	client.call("remove_workspace_user", map[string]any{"email": disposableEmail})
	if err := waitForWorkspaceUserAbsence(client, disposableEmail, 45*time.Second); err != nil {
		t.Fatal(err)
	}
	ledger.release(cleanup)
}

func cleanupWorkspaceUser(client *strictClient, email string) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		result, text, err := client.callRawResult(ctx, "remove_workspace_user", map[string]any{"email": email})
		if err != nil {
			return err
		}
		if result.IsError && !isActionableNotFound(text) {
			return fmt.Errorf("remove tool error: %s", redactedSummary(text))
		}
		return waitForWorkspaceUserAbsence(client, email, 45*time.Second)
	}
}

func waitForWorkspaceUserAbsence(client *strictClient, email string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		listedResult, listed, err := client.callRawResult(ctx, "list_workspace_users", map[string]any{"filter": email})
		if err != nil {
			return err
		}
		if listedResult.IsError {
			return fmt.Errorf("list_workspace_users failed while proving absence: %s", redactedSummary(listed))
		}
		if !userListed(listed, email) {
			result, text, err := client.callRawResult(ctx, "get_workspace_user", map[string]any{"email": email})
			if err != nil {
				return err
			}
			if result.IsError && isActionableNotFound(text) {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("removed disposable user %q remained visible or readable: %w", email, ctx.Err())
		case <-time.After(time.Second):
		}
	}
}

func TestMalformedInviteEmailIsLocallyRejected(t *testing.T) {
	requireUserMutationGate(t)
	client := newStrictClient(t)
	result, text := client.callRaw("invite_workspace_user", map[string]any{"email": "not-an-email"})
	if !result.IsError || strings.TrimSpace(text) == "" {
		t.Fatalf("malformed invite email must return a clean local validation error: %s", redactedSummary(text))
	}
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "email") || (!strings.Contains(lower, "invalid") && !strings.Contains(lower, "valid")) {
		t.Fatalf("malformed invite diagnostic is not actionable local email validation: %s", redactedSummary(text))
	}
}

func TestInvalidSandboxInputsCreateNothing(t *testing.T) {
	client := newStrictClient(t)
	for _, test := range []struct {
		name      string
		wantField string
		args      map[string]any
	}{
		{"negative_memory", "memory", map[string]any{"memory": -100}},
		{"absurd_memory", "memory", map[string]any{"memory": 999999999999}},
		{"nonnumeric_port", "port", map[string]any{"ports": "not-a-port"}},
		{"zero_port", "port", map[string]any{"ports": "0"}},
		{"port_above_65535", "port", map[string]any{"ports": "70000"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ledger := newCleanupLedger(t, client)
			name := liveConfig.Prefix + "-invalid-" + strings.ReplaceAll(test.name, "_", "-")
			if existing := client.call("list_sandboxes", map[string]any{"filter": name}); listedResource(existing, name) {
				t.Fatalf("refusing invalid-input probe against colliding sandbox %q", name)
			}
			cleanup := ledger.add("invalid-input sandbox", name, cleanupTool(client, "delete_sandbox", map[string]any{"name": name}))
			args := map[string]any{"name": name}
			for key, value := range test.args {
				args[key] = value
			}
			result, text := client.callRaw("create_sandbox", args)
			if !result.IsError || !strings.Contains(strings.ToLower(text), test.wantField) {
				t.Fatalf("invalid sandbox input unexpectedly succeeded or lacked an actionable %s error: %s", test.wantField, redactedSummary(text))
			}
			listed := client.call("list_sandboxes", map[string]any{"filter": name})
			if listedResource(listed, name) {
				t.Fatalf("invalid sandbox input created resource %q", name)
			}
			ledger.release(cleanup)
		})
	}
}

func TestServiceAccountLifecycle(t *testing.T) {
	client := newStrictClient(t)
	requireMappedWorkflowCalls(t, client, "TestServiceAccountLifecycle")
	ledger := newCleanupLedger(t, client)
	name := liveConfig.Prefix + "-sa"
	renamed := liveConfig.Prefix + "-renamed"
	clientID := ""
	if existing := client.call("list_service_accounts", map[string]any{"filter": name}); listedResource(existing, name) {
		t.Fatalf("refusing to mutate colliding service account %q", name)
	}
	// If create succeeds remotely but its response is lost or malformed, cleanup
	// resolves the client ID from the unique name before deleting the account.
	accountCleanup := ledger.add("service account", name, cleanupServiceAccount(client, name, &clientID))

	createdText := client.call("create_service_account", map[string]any{"name": name})
	created := parseJSONObject(t, createdText)
	account, ok := created["service_account"].(map[string]any)
	if !ok {
		t.Fatal("create_service_account result missing service_account object")
	}
	clientID, _ = account["client_id"].(string)
	if clientID == "" {
		t.Fatal("create_service_account result missing client_id")
	}
	secret, _ := account["client_secret"].(string)
	if secret == "" || secret == "[REDACTED]" {
		t.Fatal("create_service_account result missing one-time client_secret")
	}
	// Register the one-time secret before any later tool call or stderr drain.
	// Leak detection stores it only in memory and never prints it.
	client.protectSecret(secret)
	delete(account, "client_secret")

	client.call("get_service_account", map[string]any{"name": clientID})
	client.call("update_service_account", map[string]any{"client_id": clientID, "name": renamed})
	deadline := time.Now().Add(45 * time.Second)
	for {
		listed := client.call("list_service_accounts", map[string]any{"filter": liveConfig.Prefix})
		if strings.Contains(listed, renamed) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("list_service_accounts did not observe the renamed service account")
		}
		time.Sleep(time.Second)
	}

	client.call("delete_service_account", map[string]any{"name": clientID})
	if !waitForListAbsence(t, client, "list_service_accounts", map[string]any{"filter": liveConfig.Prefix}, liveConfig.Prefix, 45*time.Second) {
		t.Fatal("deleted service account remains visible in list_service_accounts")
	}
	ledger.release(accountCleanup)
}
