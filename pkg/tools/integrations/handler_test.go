package integrations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/toolkit/sdk"
)

const testIntegrationSecret = "dummy-test-token-do-not-use"

func integrationWithSecret(name string) sdk.IntegrationConnection {
	integrationType := "github"
	secret := map[string]string{"token": testIntegrationSecret}
	return sdk.IntegrationConnection{
		Metadata: &sdk.Metadata{Name: &name},
		Spec: &sdk.IntegrationConnectionSpec{
			Integration: &integrationType,
			Secret:      &secret,
		},
	}
}

func TestMaskIntegrationSecretsReturnsRedactedCopy(t *testing.T) {
	integration := integrationWithSecret("test-integration")

	masked := maskIntegrationSecrets(integration)

	if got := (*masked.Spec.Secret)["token"]; got != "[REDACTED]" {
		t.Fatalf("masked secret = %q, want [REDACTED]", got)
	}
	if got := (*integration.Spec.Secret)["token"]; got != testIntegrationSecret {
		t.Fatalf("maskIntegrationSecrets modified its input: %q", got)
	}
}

func TestGetIntegrationRedactsUpstreamSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"metadata":{"name":"test-integration"},"spec":{"integration":"github","secret":{"token":"dummy-test-token-do-not-use"}}}`))
	}))
	t.Cleanup(server.Close)

	handler, err := NewSDKHandler(&config.Config{
		APIEndpoint: server.URL,
		RunEndpoint: server.URL,
		Workspace:   "test-workspace",
		Credentials: sdk.Credentials{APIKey: "test-api-key"},
	})
	if err != nil {
		t.Fatalf("failed to create integration handler: %v", err)
	}

	result, err := handler.GetIntegration(context.Background(), "test-integration")
	if err != nil {
		t.Fatalf("get integration failed: %v", err)
	}
	if strings.Contains(string(result), testIntegrationSecret) {
		t.Fatal("get_integration response contains the plaintext secret")
	}
	if !strings.Contains(string(result), "[REDACTED]") {
		t.Fatalf("get_integration response does not contain a redaction marker: %s", result)
	}
}

func TestListIntegrationModelRedactsSecrets(t *testing.T) {
	model := convertToIntegrationModel(integrationWithSecret("test-integration"))
	if got := model.Secrets["token"]; got != "[REDACTED]" {
		t.Fatalf("formatted secret = %q, want [REDACTED]", got)
	}
}
