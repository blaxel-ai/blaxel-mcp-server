package tools_test

import (
	"context"
	"testing"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/integrations"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/mcpservers"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/modelapis"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/runtime"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/sandboxes"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/serviceaccounts"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools/users"
	"github.com/mark3labs/mcp-go/server"
)

func TestCreateAndInviteAnnotationsAreDestructive(t *testing.T) {
	s := newAnnotationTestServer()

	for _, name := range []string{
		"create_integration",
		"create_mcp_server",
		"create_model_api",
		"create_sandbox",
		"create_service_account",
		"invite_workspace_user",
	} {
		t.Run(name, func(t *testing.T) {
			assertMutationAnnotations(t, s, name, true)
		})
	}
}

func TestDestructiveMutationAnnotationsRemainDestructive(t *testing.T) {
	s := newAnnotationTestServer()

	for _, name := range []string{
		"delete_integration",
		"delete_mcp_server",
		"delete_model_api",
		"delete_sandbox",
		"delete_service_account",
		"update_service_account",
		"update_workspace_user_role",
		"remove_workspace_user",
		"run_agent",
		"run_job",
		"run_model",
		"run_sandbox_command",
		"stop_sandbox_process",
		"kill_sandbox_process",
	} {
		t.Run(name, func(t *testing.T) {
			assertMutationAnnotations(t, s, name, true)
		})
	}
}

func assertMutationAnnotations(t *testing.T, s *server.MCPServer, name string, destructive bool) {
	t.Helper()

	registered := s.GetTool(name)
	if registered == nil {
		t.Fatalf("tool %q is not registered", name)
	}
	annotations := registered.Tool.Annotations
	if annotations.ReadOnlyHint == nil {
		t.Fatalf("tool %q readOnlyHint is nil", name)
	}
	if *annotations.ReadOnlyHint {
		t.Errorf("tool %q readOnlyHint = true, want false", name)
	}
	if annotations.DestructiveHint == nil {
		t.Fatalf("tool %q destructiveHint is nil", name)
	}
	if got := *annotations.DestructiveHint; got != destructive {
		t.Errorf("tool %q destructiveHint = %t, want %t", name, got, destructive)
	}
}

func newAnnotationTestServer() *server.MCPServer {
	s := server.NewMCPServer("annotation-test", "1.0.0")
	handler := annotationHandler{}
	cfg := &config.Config{}

	integrations.RegisterIntegrationTools(s, handler, cfg)
	mcpservers.RegisterMCPServerTools(s, handler, cfg)
	modelapis.RegisterModelAPITools(s, handler, cfg)
	sandboxes.RegisterSandboxTools(s, handler, cfg)
	serviceaccounts.RegisterServiceAccountTools(s, handler, cfg)
	users.RegisterUserTools(s, handler, cfg)
	runtime.RegisterRuntimeTools(s, handler, cfg)

	return s
}

type annotationHandler struct{}

func (annotationHandler) ListIntegrations(context.Context, string) ([]byte, error) { return nil, nil }
func (annotationHandler) GetIntegration(context.Context, string) ([]byte, error)   { return nil, nil }
func (annotationHandler) CreateIntegration(context.Context, string, string, map[string]string, map[string]string) ([]byte, error) {
	return nil, nil
}
func (annotationHandler) DeleteIntegration(context.Context, string) ([]byte, error) { return nil, nil }

func (annotationHandler) ListMCPServers(context.Context, string) ([]byte, error) { return nil, nil }
func (annotationHandler) GetMCPServer(context.Context, string) ([]byte, error)   { return nil, nil }
func (annotationHandler) CreateMCPServer(context.Context, string, string, string, string, map[string]string, map[string]string) ([]byte, error) {
	return nil, nil
}
func (annotationHandler) DeleteMCPServer(context.Context, string, string) ([]byte, error) {
	return nil, nil
}

func (annotationHandler) ListModelAPIs(context.Context, string) ([]byte, error) { return nil, nil }
func (annotationHandler) GetModelAPI(context.Context, string) ([]byte, error)   { return nil, nil }
func (annotationHandler) CreateModelAPI(context.Context, string, string, string, string, string, string, string, map[string]interface{}) ([]byte, error) {
	return nil, nil
}
func (annotationHandler) DeleteModelAPI(context.Context, string, string) ([]byte, error) {
	return nil, nil
}

func (annotationHandler) ListSandboxes(context.Context, string) ([]byte, error) { return nil, nil }
func (annotationHandler) GetSandbox(context.Context, string) ([]byte, error)    { return nil, nil }
func (annotationHandler) CreateSandbox(context.Context, string, string, float64, string, string) ([]byte, error) {
	return nil, nil
}
func (annotationHandler) DeleteSandbox(context.Context, string) ([]byte, error) { return nil, nil }

func (annotationHandler) ListServiceAccounts(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (annotationHandler) GetServiceAccount(context.Context, string) ([]byte, error) { return nil, nil }
func (annotationHandler) CreateServiceAccount(context.Context, string, bool) ([]byte, error) {
	return nil, nil
}
func (annotationHandler) DeleteServiceAccount(context.Context, string) ([]byte, error) {
	return nil, nil
}
func (annotationHandler) UpdateServiceAccount(context.Context, string, string) ([]byte, error) {
	return nil, nil
}

func (annotationHandler) ListUsers(context.Context, string) ([]byte, error) { return nil, nil }
func (annotationHandler) GetUser(context.Context, string) ([]byte, error)   { return nil, nil }
func (annotationHandler) InviteUser(context.Context, string, string) ([]byte, error) {
	return nil, nil
}
func (annotationHandler) UpdateUserRole(context.Context, string, string) ([]byte, error) {
	return nil, nil
}
func (annotationHandler) RemoveUser(context.Context, string) ([]byte, error) { return nil, nil }

func (annotationHandler) RunAgent(context.Context, string, string, string) (string, error) {
	return "", nil
}
func (annotationHandler) RunJob(context.Context, string, string) (string, error) { return "", nil }
func (annotationHandler) RunModel(context.Context, string, string, string, string) (string, error) {
	return "", nil
}
func (annotationHandler) RunSandbox(context.Context, string, string, string, string) (string, error) {
	return "", nil
}
