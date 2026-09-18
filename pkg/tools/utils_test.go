package tools

import "testing"

func TestMaskSecret(t *testing.T) {
	for _, test := range []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty remains empty"},
		{name: "short secret", value: "abc", want: "[REDACTED]"},
		{name: "long secret", value: "blx_live-client-secret", want: "[REDACTED]"},
		{name: "already masked", value: "abc***xyz", want: "[REDACTED]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := MaskSecret(test.value); got != test.want {
				t.Fatalf("MaskSecret(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestMaskSecretMapReturnsRedactedCopy(t *testing.T) {
	original := map[string]string{"token": "live-token", "client_secret": "another-secret"}

	masked := MaskSecretMap(original)

	for key := range original {
		if masked[key] != "[REDACTED]" {
			t.Fatalf("masked[%q] = %q, want [REDACTED]", key, masked[key])
		}
	}
	if original["token"] != "live-token" {
		t.Fatal("MaskSecretMap modified its input")
	}
	if MaskSecretMap(nil) != nil {
		t.Fatal("MaskSecretMap(nil) must return nil")
	}
}

// SetRuntimeEnv used to index parts[1] unconditionally, so an entry without
// "=" panicked. Under the default stdio transport that killed the process, so
// one malformed tool argument ended the session.
func TestSetRuntimeEnvRejectsMalformedEntriesWithoutPanicking(t *testing.T) {
	for name, env := range map[string]string{
		"missing separator":      "API_KEY",
		"one good one malformed": "DEBUG=1,API_KEY",
		"empty name":             "=value",
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("SetRuntimeEnv(%q) panicked: %v", env, r)
				}
			}()
			got, err := SetRuntimeEnv(env)
			if err == nil {
				t.Fatalf("SetRuntimeEnv(%q) error = nil, want a rejection", env)
			}
			if got != nil {
				t.Fatalf("SetRuntimeEnv(%q) returned envs alongside an error", env)
			}
		})
	}
}

func TestSetRuntimeEnvAcceptsWellFormedEntries(t *testing.T) {
	got, err := SetRuntimeEnv(" DEBUG=1, TOKEN=a=b ")
	if err != nil {
		t.Fatalf("SetRuntimeEnv() error = %v", err)
	}
	if got == nil || len(*got) != 2 {
		t.Fatalf("SetRuntimeEnv() = %v, want 2 entries", got)
	}
	first, ok := (*got)[0].(map[string]interface{})
	if !ok {
		t.Fatalf("entry 0 has unexpected type %T", (*got)[0])
	}
	if first["name"] != "DEBUG" || first["value"] != "1" {
		t.Fatalf("entry 0 = %v, want DEBUG=1", first)
	}
	// A value may legitimately contain "=".
	second := (*got)[1].(map[string]interface{})
	if second["name"] != "TOKEN" || second["value"] != "a=b" {
		t.Fatalf("entry 1 = %v, want TOKEN=a=b", second)
	}

	empty, err := SetRuntimeEnv("")
	if err != nil {
		t.Fatalf("SetRuntimeEnv(\"\") error = %v", err)
	}
	if empty != nil {
		t.Fatal("SetRuntimeEnv(\"\") must return nil so an absent env block stays absent")
	}
}
