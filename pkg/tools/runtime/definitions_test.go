package runtime

import "testing"

func TestBuildSandboxCommandBodyIncludesFalseBooleans(t *testing.T) {
	body := buildSandboxCommandBody(runSandboxCommandArguments{
		Command:           "echo hello",
		WaitForCompletion: false,
		RestartOnFailure:  false,
		KeepAlive:         false,
	})

	for _, key := range []string{"waitForCompletion", "restartOnFailure", "keepAlive"} {
		value, ok := body[key]
		if !ok {
			t.Fatalf("expected %q to be included even when false; body=%v", key, body)
		}
		boolValue, ok := value.(bool)
		if !ok {
			t.Fatalf("expected %q to be a bool, got %T", key, value)
		}
		if boolValue {
			t.Fatalf("expected %q to be false, got true", key)
		}
	}
}

func TestBuildSandboxCommandBodyIncludesOptionalFields(t *testing.T) {
	body := buildSandboxCommandBody(runSandboxCommandArguments{
		Command:           "npm start",
		ProcessName:       "dev",
		WorkingDir:        "/app",
		Env:               map[string]string{"NODE_ENV": "test"},
		WaitForCompletion: true,
		Timeout:           30,
		WaitForPorts:      []int{3000},
		RestartOnFailure:  true,
		MaxRestarts:       2,
		KeepAlive:         true,
	})

	expected := map[string]interface{}{
		"command":           "npm start",
		"name":              "dev",
		"workingDir":        "/app",
		"env":               map[string]string{"NODE_ENV": "test"},
		"waitForCompletion": true,
		"timeout":           30,
		"waitForPorts":      []int{3000},
		"restartOnFailure":  true,
		"maxRestarts":       2,
		"keepAlive":         true,
	}

	for key, expectedValue := range expected {
		value, ok := body[key]
		if !ok {
			t.Fatalf("expected %q to be included; body=%v", key, body)
		}
		switch want := expectedValue.(type) {
		case map[string]string:
			got, ok := value.(map[string]string)
			if !ok || got["NODE_ENV"] != want["NODE_ENV"] {
				t.Fatalf("expected %q=%v, got %T %v", key, want, value, value)
			}
		case []int:
			got, ok := value.([]int)
			if !ok || len(got) != len(want) || got[0] != want[0] {
				t.Fatalf("expected %q=%v, got %T %v", key, want, value, value)
			}
		default:
			if value != expectedValue {
				t.Fatalf("expected %q=%v, got %v", key, expectedValue, value)
			}
		}
	}
}
