package live

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestENG3423AgentRequestMapping(t *testing.T) {
	agent := strings.TrimSpace(os.Getenv("BLAXEL_E2E_AGENT"))
	if agent == "" {
		t.Fatal("ENG-3423 lane requires BLAXEL_E2E_AGENT naming a real request-echo agent")
	}
	client := newStrictClient(t)
	if listed := client.call("list_agents", map[string]any{"filter": agent}); !strings.Contains(listed, agent) {
		t.Fatal("dedicated request-echo agent fixture is not present")
	}

	marker := "agent-message-" + liveConfig.Prefix
	agentContext := map[string]any{"trace": marker, "nested": map[string]any{"enabled": true}}
	contextJSON, err := json.Marshal(agentContext)
	if err != nil {
		t.Fatal(err)
	}
	shorthand := parseJSONObject(t, client.call("run_agent", map[string]any{
		"name": agent, "message": marker, "context": string(contextJSON),
	}))
	wantShorthand := map[string]any{"input": marker, "context": agentContext}
	if !reflect.DeepEqual(shorthand["body"], wantShorthand) {
		t.Fatalf("echoed shorthand body = %#v, want %#v", shorthand["body"], wantShorthand)
	}

	bodyMarker := "agent-body-" + liveConfig.Prefix
	body := map[string]any{"marker": bodyMarker, "nested": map[string]any{"number": float64(42), "enabled": true}}
	neutral := parseJSONObject(t, client.call("run_agent", map[string]any{
		"name": agent, "path": "/e2e/echo", "body": body,
	}))
	if neutral["path"] != "/e2e/echo" {
		t.Fatalf("echoed path = %#v, want /e2e/echo", neutral["path"])
	}
	if !reflect.DeepEqual(neutral["body"], body) {
		t.Fatalf("echoed body = %#v, want %#v", neutral["body"], body)
	}
}

func TestReviewerJobFixture(t *testing.T) {
	job := strings.TrimSpace(os.Getenv("BLAXEL_E2E_JOB"))
	if job == "" {
		t.Fatal("reviewer fixture lane requires BLAXEL_E2E_JOB naming a real echo job")
	}
	client := newStrictClient(t)
	if listed := client.call("list_jobs", map[string]any{"filter": job}); !strings.Contains(listed, job) {
		t.Fatal("dedicated reviewer job fixture is not present")
	}
	if got := client.call("get_job", map[string]any{"id": job}); !strings.Contains(got, job) {
		t.Fatal("get_job did not return the dedicated reviewer fixture")
	}

	marker := "job-parameters-" + liveConfig.Prefix
	parameters, err := json.Marshal(map[string]any{"tasks": []map[string]any{{"marker": marker}}})
	if err != nil {
		t.Fatal(err)
	}
	result := client.call("run_job", map[string]any{"name": job, "parameters": string(parameters)})
	if !strings.Contains(result, marker) {
		t.Fatal("reviewer job fixture output did not prove parameters mapping")
	}
}
