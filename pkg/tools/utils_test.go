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
