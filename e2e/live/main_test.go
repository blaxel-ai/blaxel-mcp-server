package live

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

var liveConfig struct {
	Workspace string
	Prefix    string
	Fixtures  fixtureManifest
}

type disposableRuntimeFixture struct {
	Name            string `json:"name"`
	DeleteAfterTest bool   `json:"delete_after_test"`
}

type fixtureManifest struct {
	Agent disposableRuntimeFixture `json:"agent"`
	Job   disposableRuntimeFixture `json:"job"`
}

func TestMain(m *testing.M) {
	root, err := repoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "live e2e manifest invalid: repository root not found")
		os.Exit(2)
	}

	// Load credentials without printing either the file contents or values.
	_ = godotenv.Load(filepath.Join(root, ".env.live"))

	if err := validateBaseManifest(); err != nil {
		fmt.Fprintf(os.Stderr, "live e2e base manifest invalid: %v\n", err)
		os.Exit(2)
	}
	code := m.Run()
	if code == 0 && fullSuiteRunRequested() {
		if err := verifyMappedWorkflowsExecuted(); err != nil {
			fmt.Fprintf(os.Stderr, "live e2e executable coverage gate: %v\n", err)
			code = 1
		}
	}
	os.Exit(code)
}

func fullSuiteRunRequested() bool {
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-test.run") {
			return false
		}
	}
	return true
}

func validateBaseManifest() error {
	workspace := strings.TrimSpace(os.Getenv("BL_WORKSPACE"))
	dedicatedWorkspace := strings.TrimSpace(os.Getenv("BLAXEL_E2E_WORKSPACE"))
	if workspace == "" || dedicatedWorkspace == "" {
		return fmt.Errorf("BL_WORKSPACE and BLAXEL_E2E_WORKSPACE are required")
	}
	if workspace != dedicatedWorkspace {
		return fmt.Errorf("BL_WORKSPACE must equal BLAXEL_E2E_WORKSPACE")
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
		return fmt.Errorf("BLAXEL_E2E_WORKSPACE must match BLAXEL_E2E_WORKSPACE_ALLOWLIST or BLAXEL_E2E_WORKSPACE_PREFIX")
	}
	if os.Getenv("BLAXEL_E2E_ALLOW_DESTRUCTIVE") != "true" {
		return fmt.Errorf("BLAXEL_E2E_ALLOW_DESTRUCTIVE must equal true")
	}
	if os.Getenv("BL_DEBUG") == "true" {
		return fmt.Errorf("BL_DEBUG must not be true because live output may contain credentials")
	}
	if strings.TrimSpace(os.Getenv("BL_API_KEY")) == "" {
		return fmt.Errorf("BL_API_KEY is required")
	}

	prefix := strings.TrimSpace(os.Getenv("BLAXEL_E2E_RUN_PREFIX"))
	if !regexp.MustCompile(`^mcp-live-[a-z0-9]+-[a-z0-9]+$`).MatchString(prefix) || len(prefix) > 48 {
		return fmt.Errorf("BLAXEL_E2E_RUN_PREFIX must be unique, <=48 characters, and match mcp-live-<run>-<nonce>")
	}

	liveConfig.Workspace = workspace
	liveConfig.Prefix = prefix
	return nil
}

// loadRuntimeFixtures is lane-specific so focused contract, PM-2609, and
// ENG-3885 runs do not require optional agent/job fixtures.
func loadRuntimeFixtures() (fixtureManifest, error) {
	root, err := repoRoot()
	if err != nil {
		return fixtureManifest{}, err
	}
	manifestPath := strings.TrimSpace(os.Getenv("BLAXEL_E2E_FIXTURE_MANIFEST"))
	if manifestPath == "" {
		return fixtureManifest{}, fmt.Errorf("runtime-fixture lane requires BLAXEL_E2E_FIXTURE_MANIFEST with agent.name and job.name")
	}
	if !filepath.IsAbs(manifestPath) {
		manifestPath = filepath.Join(root, manifestPath)
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fixtureManifest{}, fmt.Errorf("runtime-fixture lane cannot read BLAXEL_E2E_FIXTURE_MANIFEST")
	}
	var fixtures fixtureManifest
	if err := json.Unmarshal(data, &fixtures); err != nil {
		return fixtureManifest{}, fmt.Errorf("runtime-fixture BLAXEL_E2E_FIXTURE_MANIFEST is not valid JSON")
	}
	if fixtures.Agent.Name == "" || fixtures.Job.Name == "" {
		return fixtureManifest{}, fmt.Errorf("runtime-fixture manifest requires agent.name and job.name")
	}
	for kind, fixture := range map[string]disposableRuntimeFixture{"agent": fixtures.Agent, "job": fixtures.Job} {
		if !fixture.DeleteAfterTest {
			return fixtureManifest{}, fmt.Errorf("runtime-fixture manifest requires %s.delete_after_test=true", kind)
		}
		if !strings.HasPrefix(fixture.Name, "mcp-live-") {
			return fixtureManifest{}, fmt.Errorf("runtime-fixture manifest requires %s.name to start with mcp-live- before deletion is allowed", kind)
		}
	}
	return fixtures, nil
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		if filepath.Dir(dir) == dir {
			return "", fmt.Errorf("go.mod not found")
		}
	}
}
