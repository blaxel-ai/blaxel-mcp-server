package sandboxes

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/mark3labs/mcp-go/server"
)

// creatorTestHandler implements SandboxCreator and records what it receives.
type creatorTestHandler struct {
	createSandboxTestHandler
	options []CreateSandboxOptions
	err     error
}

func (h *creatorTestHandler) CreateSandboxWithOptions(_ context.Context, opts CreateSandboxOptions) ([]byte, error) {
	h.options = append(h.options, opts)
	if h.err != nil {
		return nil, h.err
	}
	return []byte(`{"success":true,"message":"created"}`), nil
}

func newCreatorToolTestServer(err error) (*server.MCPServer, *creatorTestHandler) {
	mcpServer := server.NewMCPServer("sandbox-test", "1.0.0")
	handler := &creatorTestHandler{err: err}
	RegisterSandboxTools(mcpServer, handler, &config.Config{})
	return mcpServer, handler
}

func lifecycleArguments() map[string]any {
	return map[string]any{
		"name":   "lifecycle",
		"image":  "blaxel/base-image:latest",
		"memory": 2048,
		"region": "us-pdx-1",
		"ports":  "3000",
		"env":    "FOO=bar",
		"labels": map[string]any{"task": "build"},
		"ttl":    "24h",
		"expirationPolicies": []any{
			map[string]any{"type": "ttl-idle", "value": "1h"},
			map[string]any{"type": "date", "value": "2030-01-01T00:00:00Z", "action": "delete"},
		},
		"terminatedRetention": "1h",
		"volumes": []any{
			map[string]any{"name": "data", "mountPath": "/data", "readOnly": true},
			map[string]any{"name": "scratch", "mountPath": "/scratch", "type": "ephemeral", "sizeMb": 2048},
		},
		"standbyGracePeriodSeconds":     10,
		"terminationGracePeriodSeconds": 30,
		"createIfNotExist":              true,
	}
}

func TestCreateSandboxToolPassesEverySettingToCreator(t *testing.T) {
	mcpServer, handler := newCreatorToolTestServer(nil)

	result := callCreateSandboxTool(t, mcpServer, lifecycleArguments())
	if result.IsError {
		t.Fatalf("create_sandbox failed: %s", toolResultText(result))
	}
	if len(handler.options) != 1 || len(handler.createCalls) != 0 {
		t.Fatalf("options calls = %d, legacy calls = %d, want 1 and 0", len(handler.options), len(handler.createCalls))
	}

	standby, termination := 10, 30
	want := CreateSandboxOptions{
		Name:   "lifecycle",
		Image:  "blaxel/base-image:latest",
		Memory: 2048,
		Region: "us-pdx-1",
		Ports:  "3000",
		Env:    "FOO=bar",
		Labels: map[string]string{"task": "build"},
		TTL:    "24h",
		ExpirationPolicies: []ExpirationPolicy{
			{Type: "ttl-idle", Value: "1h", Action: "delete"},
			{Type: "date", Value: "2030-01-01T00:00:00Z", Action: "delete"},
		},
		TerminatedRetention: "1h",
		Volumes: []VolumeAttachment{
			{Name: "data", MountPath: "/data", ReadOnly: true},
			{Name: "scratch", MountPath: "/scratch", Type: "ephemeral", SizeMB: 2048},
		},
		StandbyGracePeriodSeconds:     &standby,
		TerminationGracePeriodSeconds: &termination,
		CreateIfNotExist:              true,
	}
	if got := handler.options[0]; !reflect.DeepEqual(got, want) {
		t.Fatalf("options =\n%#v\nwant\n%#v", got, want)
	}
}

// Durations are the API's to judge: a value under the minimum still reaches
// the handler, and the API's 400 explains the limit.
func TestCreateSandboxToolLeavesDurationRulesToTheAPI(t *testing.T) {
	mcpServer, handler := newCreatorToolTestServer(nil)

	result := callCreateSandboxTool(t, mcpServer, map[string]any{
		"name": "short",
		"ttl":  "1m",
		"expirationPolicies": []any{
			map[string]any{"type": "ttl-idle", "value": "10s"},
		},
	})
	if result.IsError {
		t.Fatalf("create_sandbox rejected a duration locally: %s", toolResultText(result))
	}
	if len(handler.options) != 1 || handler.options[0].TTL != "1m" || handler.options[0].ExpirationPolicies[0].Value != "10s" {
		t.Fatalf("durations were not passed through: %#v", handler.options)
	}
}

func TestCreateSandboxToolRejectsLifecycleSettingsTheHandlerCannotApply(t *testing.T) {
	for _, field := range []string{"ttl", "expirationPolicies", "volumes", "labels", "createIfNotExist"} {
		t.Run(field, func(t *testing.T) {
			mcpServer, handler := newCreateSandboxToolTestServer()
			args := map[string]any{"name": "legacy"}
			args[field] = lifecycleArguments()[field]

			result := callCreateSandboxTool(t, mcpServer, args)
			if !result.IsError || !strings.Contains(toolResultText(result), field) {
				t.Fatalf("expected an error naming %s, got error=%v text=%q", field, result.IsError, toolResultText(result))
			}
			if len(handler.createCalls) != 0 {
				t.Fatalf("legacy handler was called and would have dropped %s", field)
			}
		})
	}
}

func TestCreateSandboxToolRejectsMalformedLifecycleArguments(t *testing.T) {
	for _, test := range []struct {
		name      string
		field     string
		value     any
		wantInErr string
	}{
		{"misspelled volume field", "volumes", []any{map[string]any{"name": "data", "mount_path": "/data"}}, "mount_path"},
		{"volume without mount path", "volumes", []any{map[string]any{"name": "data"}}, "mountPath"},
		{"volume without name", "volumes", []any{map[string]any{"mountPath": "/data"}}, "name"},
		{"ephemeral volume without size", "volumes", []any{map[string]any{"name": "tmp", "mountPath": "/tmp", "type": "ephemeral"}}, "sizeMb"},
		{"volumes not a list", "volumes", "data:/data", "volumes"},
		{"policy without type", "expirationPolicies", []any{map[string]any{"value": "1h"}}, "type"},
		{"policy without value", "expirationPolicies", []any{map[string]any{"type": "ttl-idle"}}, "value"},
		{"misspelled policy field", "expirationPolicies", []any{map[string]any{"type": "ttl-idle", "duration": "1h"}}, "duration"},
		{"ttl not a string", "ttl", 300, "ttl"},
		{"label value not a string", "labels", map[string]any{"count": 1}, "labels"},
		{"negative standby grace", "standbyGracePeriodSeconds", -1, "standbyGracePeriodSeconds"},
		{"fractional termination grace", "terminationGracePeriodSeconds", 1.5, "terminationGracePeriodSeconds"},
	} {
		t.Run(test.name, func(t *testing.T) {
			mcpServer, handler := newCreatorToolTestServer(nil)
			result := callCreateSandboxTool(t, mcpServer, map[string]any{"name": "invalid", test.field: test.value})
			if !result.IsError || !strings.Contains(toolResultText(result), test.wantInErr) {
				t.Fatalf("expected an error naming %q, got error=%v text=%q", test.wantInErr, result.IsError, toolResultText(result))
			}
			if len(handler.options) != 0 {
				t.Fatalf("malformed %s reached the handler", test.field)
			}
		})
	}
}

func TestCreateSandboxToolAddsNextStepToErrors(t *testing.T) {
	for _, test := range []struct {
		name     string
		err      string
		args     map[string]any
		wantHint []string
		notHint  []string
	}{
		{
			name:     "ttl below minimum",
			err:      "failed to create sandbox with status 400: Invalid sandbox configuration: ttl 1m is too short: the minimum is 5m",
			args:     map[string]any{"name": "short", "ttl": "1m"},
			wantHint: []string{"at least 5m", "at least 60s", "delete_sandbox"},
		},
		{
			name:     "bad duration format",
			err:      "failed to create sandbox with status 400: Invalid TTL format: 1y. Use a duration like '30m', '2h', '7d', or '2w'",
			args:     map[string]any{"name": "format", "ttl": "1y"},
			wantHint: []string{"s, m, h, d or w"},
		},
		{
			name:     "lifecycle 400 without details",
			err:      "invalid sandbox configuration",
			args:     map[string]any{"name": "masked", "ttl": "1m"},
			wantHint: []string{"at least 5m", "ISO 8601"},
		},
		{
			name:     "name conflict",
			err:      "sandbox with name 'taken' already exists",
			args:     map[string]any{"name": "taken"},
			wantHint: []string{"createIfNotExist"},
		},
		{
			name:     "grace period out of range",
			err:      "failed to create sandbox with status 400: Invalid sandbox configuration: runtime.standbyGracePeriodSeconds must be between 0 and 300, got 301",
			args:     map[string]any{"name": "grace", "ttl": "10m", "standbyGracePeriodSeconds": 301},
			wantHint: []string{"standbyGracePeriodSeconds", "0 to 300"},
			notHint:  []string{"ttl"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			mcpServer, _ := newCreatorToolTestServer(errors.New(test.err))
			result := callCreateSandboxTool(t, mcpServer, test.args)
			text := toolResultText(result)
			if !result.IsError || !strings.HasPrefix(text, test.err) || !strings.Contains(text, "\nNext step: ") {
				t.Fatalf("expected the API error followed by a next step, got error=%v text=%q", result.IsError, text)
			}
			for _, want := range test.wantHint {
				if !strings.Contains(text, want) {
					t.Errorf("hint %q does not mention %q", text, want)
				}
			}
			hint := text[strings.Index(text, "\nNext step: "):]
			for _, unwanted := range test.notHint {
				if strings.Contains(hint, unwanted) {
					t.Errorf("hint %q is about %q, which the error does not name", hint, unwanted)
				}
			}
		})
	}
}

func TestCreateSandboxToolLeavesUnrelatedErrorsAlone(t *testing.T) {
	mcpServer, _ := newCreatorToolTestServer(errors.New("failed to create sandbox with status 500"))
	result := callCreateSandboxTool(t, mcpServer, map[string]any{"name": "plain"})
	if text := toolResultText(result); text != "failed to create sandbox with status 500" {
		t.Fatalf("unexpected hint on an unrelated error: %q", text)
	}
}

func TestCreateSandboxNextSteps(t *testing.T) {
	join := func(steps []string) string { return strings.Join(steps, "\n") }

	withExpiry := join(CreateSandboxNextSteps(CreateSandboxOptions{
		Name:  "expiring",
		Ports: "3000,8080",
		TTL:   "24h",
		ExpirationPolicies: []ExpirationPolicy{
			{Type: ExpirationPolicyTTLIdle, Value: "1h"},
			{Type: ExpirationPolicyDate, Value: "2030-01-01T00:00:00Z"},
		},
		Volumes:          []VolumeAttachment{{Name: "data", MountPath: "/data"}, {Name: "tmp", MountPath: "/tmp", Type: VolumeTypeEphemeral, SizeMB: 10}},
		CreateIfNotExist: true,
	}))
	for _, want := range []string{
		`run_sandbox_command (name "expiring")`,
		"preview URL", "port 3000", "$HOST",
		"24h after creation (ttl)", "1h without activity (ttl-idle)", "at 2030-01-01T00:00:00Z (date)", "delete_sandbox",
		`volume "data"`,
		"already existed",
	} {
		if !strings.Contains(withExpiry, want) {
			t.Errorf("next steps do not mention %q:\n%s", want, withExpiry)
		}
	}
	if strings.Contains(withExpiry, `"tmp"`) {
		t.Errorf("next steps claim an ephemeral volume keeps its data:\n%s", withExpiry)
	}

	kept := join(CreateSandboxNextSteps(CreateSandboxOptions{Name: "kept"}))
	for _, want := range []string{"No ttl or expiration policy", "standby", "until you call delete_sandbox"} {
		if !strings.Contains(kept, want) {
			t.Errorf("next steps without expiry do not mention %q:\n%s", want, kept)
		}
	}
	if strings.Contains(kept, "preview") || strings.Contains(kept, "already existed") {
		t.Errorf("next steps mention settings that were not given:\n%s", kept)
	}
}

func TestCreateSandboxSchemaStatesLifecycleLimits(t *testing.T) {
	mcpServer, _ := newCreatorToolTestServer(nil)
	tool := mcpServer.GetTool("create_sandbox").Tool

	for _, want := range []string{"5m", "60s", "HTTP 400", "delete_sandbox", "standby", "createIfNotExist", "https://docs.blaxel.ai/Sandboxes/Lifecycle-recipe"} {
		if !strings.Contains(tool.Description, want) {
			t.Errorf("create_sandbox description does not mention %q", want)
		}
	}

	properties := tool.InputSchema.Properties
	ttl, _ := properties["ttl"].(map[string]any)
	if description, _ := ttl["description"].(string); !strings.Contains(description, "minimum 5m") || !strings.Contains(description, "ttl-idle") {
		t.Errorf("ttl description = %q, want the minimum and the ttl-idle alternative", description)
	}

	policies, _ := properties["expirationPolicies"].(map[string]any)
	policy, _ := policies["items"].(map[string]any)
	policyFields, _ := policy["properties"].(map[string]any)
	policyType, _ := policyFields["type"].(map[string]any)
	if !reflect.DeepEqual(policyType["enum"], []string{"ttl-idle", "ttl-max-age", "date"}) {
		t.Errorf("expirationPolicies type enum = %v", policyType["enum"])
	}
	if description, _ := policyType["description"].(string); !strings.Contains(description, "minimum 60s") || !strings.Contains(description, "minimum 5m") {
		t.Errorf("expirationPolicies type description = %q, want both minimums", description)
	}

	volumes, _ := properties["volumes"].(map[string]any)
	volume, _ := volumes["items"].(map[string]any)
	if !reflect.DeepEqual(volume["required"], []string{"name", "mountPath"}) {
		t.Errorf("volumes items required = %v, want name and mountPath", volume["required"])
	}

	for _, name := range []string{"standbyGracePeriodSeconds", "terminationGracePeriodSeconds"} {
		grace, _ := properties[name].(map[string]any)
		if grace["minimum"] != 0 || grace["maximum"] != maxGracePeriodSeconds {
			t.Errorf("%s range = %v to %v, want the API's 0 to 300", name, grace["minimum"], grace["maximum"])
		}
	}

	createIfNotExist, _ := properties["createIfNotExist"].(map[string]any)
	if createIfNotExist["type"] != "boolean" {
		t.Errorf("createIfNotExist type = %v, want boolean", createIfNotExist["type"])
	}
}
