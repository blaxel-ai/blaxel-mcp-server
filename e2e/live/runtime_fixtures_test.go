package live

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRuntimeFixtureRequestMapping(t *testing.T) {
	fixtures, err := loadRuntimeFixtures()
	if err != nil {
		t.Fatal(err)
	}
	client := newStrictClient(t)
	requireMappedWorkflowCalls(t, client, "TestRuntimeFixtureRequestMapping")
	ledger := newCleanupLedger(t, client)
	agentCleanup := ledger.add("runtime agent fixture", fixtures.Agent.Name, cleanupDisposableRuntimeFixture(client, "delete_agent", "get_agent", "name", fixtures.Agent))
	jobCleanup := ledger.add("runtime job fixture", fixtures.Job.Name, cleanupDisposableRuntimeFixture(client, "delete_job", "get_job", "id", fixtures.Job))

	agents := client.call("list_agents", map[string]any{"filter": fixtures.Agent.Name})
	if !strings.Contains(agents, fixtures.Agent.Name) {
		t.Fatal("dedicated live agent fixture is not present in BLAXEL_E2E_WORKSPACE")
	}
	jobs := client.call("list_jobs", map[string]any{"filter": fixtures.Job.Name})
	if !strings.Contains(jobs, fixtures.Job.Name) {
		t.Fatal("dedicated live job fixture is not present in BLAXEL_E2E_WORKSPACE")
	}

	t.Run("run_agent_message_shorthand_and_context", func(t *testing.T) {
		marker := "agent-message-" + liveConfig.Prefix
		agentContext := map[string]any{"trace": marker, "nested": map[string]any{"enabled": true}}
		contextJSON, err := json.Marshal(agentContext)
		if err != nil {
			t.Fatal(err)
		}
		result := client.call("run_agent", map[string]any{
			"name": fixtures.Agent.Name, "message": marker, "context": string(contextJSON),
		})
		echo := parseJSONObject(t, result)
		wantBody := map[string]any{"input": marker, "context": agentContext}
		if !reflect.DeepEqual(echo["body"], wantBody) {
			t.Fatalf("echoed shorthand body = %#v, want exact top-level body %#v (no inputs wrapper or JSON-encoded values)", echo["body"], wantBody)
		}
	})

	t.Run("run_agent_body_and_path", func(t *testing.T) {
		marker := "agent-body-" + liveConfig.Prefix
		body := map[string]any{"marker": marker, "nested": map[string]any{"number": float64(42), "enabled": true}}
		result := client.call("run_agent", map[string]any{
			"name": fixtures.Agent.Name, "path": "/e2e/echo", "body": body,
		})
		echo := parseJSONObject(t, result)
		if echo["path"] != "/e2e/echo" {
			t.Fatalf("echoed path = %#v, want /e2e/echo", echo["path"])
		}
		if !reflect.DeepEqual(echo["body"], body) {
			t.Fatalf("echoed body = %#v, want exact arbitrary nested body %#v", echo["body"], body)
		}
	})

	t.Run("run_job_parameters", func(t *testing.T) {
		marker := "job-parameters-" + liveConfig.Prefix
		parameters, err := json.Marshal(map[string]any{"tasks": []map[string]any{{"marker": marker}}})
		if err != nil {
			t.Fatal(err)
		}
		result := client.call("run_job", map[string]any{
			"name":       fixtures.Job.Name,
			"parameters": string(parameters),
		})
		if !strings.Contains(result, marker) {
			t.Fatal("job fixture output did not prove parameters mapping")
		}
	})

	t.Run("get_agent", func(t *testing.T) {
		if got := client.call("get_agent", map[string]any{"name": fixtures.Agent.Name}); !strings.Contains(got, fixtures.Agent.Name) {
			t.Fatal("get_agent did not return the dedicated disposable fixture")
		}
	})
	t.Run("get_job", func(t *testing.T) {
		if got := client.call("get_job", map[string]any{"id": fixtures.Job.Name}); !strings.Contains(got, fixtures.Job.Name) {
			t.Fatal("get_job did not return the dedicated disposable fixture")
		}
	})

	t.Run("delete_agent", func(t *testing.T) {
		client.call("delete_agent", map[string]any{"name": fixtures.Agent.Name})
		requireRuntimeFixtureNotFound(t, client, "get_agent", "name", fixtures.Agent.Name)
		if !waitForListAbsence(t, client, "list_agents", map[string]any{"filter": fixtures.Agent.Name}, fixtures.Agent.Name, 45*time.Second) {
			t.Fatal("deleted agent fixture remains visible in list_agents")
		}
		ledger.release(agentCleanup)
	})
	t.Run("delete_job", func(t *testing.T) {
		client.call("delete_job", map[string]any{"id": fixtures.Job.Name})
		requireRuntimeFixtureNotFound(t, client, "get_job", "id", fixtures.Job.Name)
		if !waitForListAbsence(t, client, "list_jobs", map[string]any{}, fixtures.Job.Name, 45*time.Second) {
			t.Fatal("deleted job fixture remains visible in list_jobs")
		}
		ledger.release(jobCleanup)
	})
}

func cleanupDisposableRuntimeFixture(client *strictClient, deleteTool, getTool, argument string, fixture disposableRuntimeFixture) func() error {
	return func() error {
		if !fixture.DeleteAfterTest || !strings.HasPrefix(fixture.Name, "mcp-live-") {
			return fmt.Errorf("refusing to delete fixture without delete_after_test=true and an mcp-live- name")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		result, text, err := client.callRawResult(ctx, deleteTool, map[string]any{argument: fixture.Name})
		if err != nil {
			return err
		}
		if result.IsError && !isRuntimeFixtureNotFound(text) {
			return fmt.Errorf("delete tool error: %s", redactedSummary(text))
		}
		for {
			result, text, err = client.callRawResult(ctx, getTool, map[string]any{argument: fixture.Name})
			if err != nil {
				return err
			}
			if result.IsError {
				if isRuntimeFixtureNotFound(text) {
					return nil
				}
				return fmt.Errorf("get after cleanup failed: %s", redactedSummary(text))
			}
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("fixture remained readable after cleanup: %w", err)
			}
			time.Sleep(time.Second)
		}
	}
}

func requireRuntimeFixtureNotFound(t *testing.T, client *strictClient, getTool, argument, name string) {
	t.Helper()
	deadline := time.Now().Add(45 * time.Second)
	for {
		result, text := client.callRaw(getTool, map[string]any{argument: name})
		if result.IsError {
			if isRuntimeFixtureNotFound(text) {
				return
			}
			t.Fatalf("%s failed with a non-not-found error after deletion: %s", getTool, redactedSummary(text))
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s still returned deleted fixture %q", getTool, name)
		}
		time.Sleep(time.Second)
	}
}

func isRuntimeFixtureNotFound(text string) bool {
	return isActionableNotFound(text) || strings.Contains(strings.ToLower(text), "status 404")
}
