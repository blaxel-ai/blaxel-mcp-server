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
