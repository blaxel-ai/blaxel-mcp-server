package tools

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

var legacyUserEmail string

// The legacy tool tests make real provider and workspace-user mutations. Keep
// direct `go test ./...` fail-closed; the strict e2e/live suite is the supported
// acceptance path.
func TestMain(m *testing.M) {
	if os.Getenv("BLAXEL_LEGACY_E2E_ALLOW_UNSAFE") != "true" {
		fmt.Fprintln(os.Stderr, "legacy tool e2e safety gate: BLAXEL_LEGACY_E2E_ALLOW_UNSAFE must equal true")
		os.Exit(2)
	}
	workspace := strings.TrimSpace(os.Getenv("BL_WORKSPACE"))
	dedicated := strings.TrimSpace(os.Getenv("BLAXEL_E2E_WORKSPACE"))
	if workspace == "" || workspace != dedicated {
		fmt.Fprintln(os.Stderr, "legacy tool e2e safety gate: BL_WORKSPACE must equal non-empty BLAXEL_E2E_WORKSPACE")
		os.Exit(2)
	}
	allowed := false
	for _, candidate := range strings.Split(os.Getenv("BLAXEL_E2E_WORKSPACE_ALLOWLIST"), ",") {
		if strings.TrimSpace(candidate) == workspace {
			allowed = true
			break
		}
	}
	if prefix := strings.TrimSpace(os.Getenv("BLAXEL_E2E_WORKSPACE_PREFIX")); prefix != "" && strings.HasPrefix(workspace, prefix) {
		allowed = true
	}
	if !allowed {
		fmt.Fprintln(os.Stderr, "legacy tool e2e safety gate: workspace must match BLAXEL_E2E_WORKSPACE_ALLOWLIST or BLAXEL_E2E_WORKSPACE_PREFIX")
		os.Exit(2)
	}
	if os.Getenv("BLAXEL_LEGACY_E2E_ALLOW_PROVIDER_MUTATIONS") != "true" {
		fmt.Fprintln(os.Stderr, "legacy tool e2e safety gate: BLAXEL_LEGACY_E2E_ALLOW_PROVIDER_MUTATIONS must equal true")
		os.Exit(2)
	}
	if os.Getenv("BLAXEL_LEGACY_E2E_ALLOW_USER_MUTATIONS") != "true" {
		fmt.Fprintln(os.Stderr, "legacy tool e2e safety gate: BLAXEL_LEGACY_E2E_ALLOW_USER_MUTATIONS must equal true")
		os.Exit(2)
	}
	legacyUserEmail = strings.TrimSpace(os.Getenv("BLAXEL_LEGACY_E2E_USER_EMAIL"))
	if legacyUserEmail == "" {
		fmt.Fprintln(os.Stderr, "legacy tool e2e safety gate: BLAXEL_LEGACY_E2E_USER_EMAIL must identify an approved disposable no-email test user")
		os.Exit(2)
	}
	os.Exit(m.Run())
}
