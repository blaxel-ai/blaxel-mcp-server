package sandboxes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/blaxel-ai/blaxel-mcp-server/pkg/client"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/config"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/formatter"
	"github.com/blaxel-ai/blaxel-mcp-server/pkg/tools"
	"github.com/blaxel-ai/toolkit/sdk"
)

// SDKHandler implements SandboxHandler using the SDK client
type SDKHandler struct {
	sdkClient *sdk.ClientWithResponses
	readOnly  bool
}

// NewSDKHandler creates a new SDK-based sandbox handler
func NewSDKHandler(cfg *config.Config) (SandboxHandler, error) {
	sdkClient, err := client.NewSDKClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SDK client: %w", err)
	}

	return &SDKHandler{
		sdkClient: sdkClient,
		readOnly:  cfg.ReadOnly,
	}, nil
}

// resolveHandler returns a handler for the given workspace override.
func resolveHandler(defaultHandler SandboxHandler, cfg *config.Config, workspace string) (SandboxHandler, error) {
	overriddenCfg := cfg.WithWorkspace(workspace)
	if overriddenCfg == cfg {
		return defaultHandler, nil
	}
	return NewSDKHandler(overriddenCfg)
}

// ListSandboxes implements SandboxHandler.ListSandboxes
func (h *SDKHandler) ListSandboxes(ctx context.Context, filter string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := h.sdkClient.ListSandboxesWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sandboxes: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("list sandboxes failed with status %d", resp.StatusCode())
	}

	sandboxes := []sdk.Sandbox{}
	if resp.JSON200 != nil {
		sandboxes = *resp.JSON200
	}

	// Apply filter if requested
	if filter != "" {
		var filtered []sdk.Sandbox
		for _, sandbox := range sandboxes {
			if sandbox.Metadata != nil && sandbox.Metadata.Name != nil &&
				tools.ContainsString(*sandbox.Metadata.Name, filter) {
				filtered = append(filtered, sandbox)
			}
		}
		sandboxes = filtered
	}

	// Convert SDK sandboxes to simple models
	sandboxModels := make([]formatter.SandboxModel, len(sandboxes))
	for i, sandbox := range sandboxes {
		sandboxModels[i] = convertToSandboxModel(sandbox)
	}

	// Format the sandboxes using the formatter
	formatted := formatter.FormatSandboxes(sandboxModels)
	return []byte(formatted), nil
}

// GetSandbox implements SandboxHandler.GetSandbox
func (h *SDKHandler) GetSandbox(ctx context.Context, name string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	sandbox, err := h.sdkClient.GetSandboxWithResponse(ctx, name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox: %w", err)
	}

	if sandbox.JSON200 == nil {
		return nil, fmt.Errorf("no sandbox found")
	}

	// Convert to JSON for better formatting
	jsonData, err := json.MarshalIndent(*sandbox.JSON200, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format sandbox data: %w", err)
	}

	return jsonData, nil
}

// CreateSandbox implements SandboxHandler.CreateSandbox
func (h *SDKHandler) CreateSandbox(ctx context.Context, name, image string, memory float64, ports, env string) ([]byte, error) {
	return h.CreateSandboxWithOptions(ctx, CreateSandboxOptions{Name: name, Image: image, Memory: memory, Ports: ports, Env: env})
}

// CreateSandboxInRegion forwards explicit placement to the sandbox API.
func (h *SDKHandler) CreateSandboxInRegion(ctx context.Context, name, image string, memory float64, ports, env, region string) ([]byte, error) {
	return h.CreateSandboxWithOptions(ctx, CreateSandboxOptions{Name: name, Image: image, Memory: memory, Ports: ports, Env: env, Region: region})
}

// createSandboxBody is the create-sandbox request body. It is written out
// here because the toolkit SDK types predate the lifecycle, volume and grace
// period fields.
type createSandboxBody struct {
	Metadata createSandboxMetadata `json:"metadata"`
	Spec     createSandboxSpec     `json:"spec"`
}

type createSandboxMetadata struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`
}

type createSandboxSpec struct {
	Region    string                  `json:"region,omitempty"`
	Runtime   createSandboxRuntime    `json:"runtime"`
	Lifecycle *createSandboxLifecycle `json:"lifecycle,omitempty"`
	Volumes   []VolumeAttachment      `json:"volumes,omitempty"`
}

type createSandboxRuntime struct {
	Image                         string        `json:"image,omitempty"`
	Memory                        int           `json:"memory,omitempty"`
	Ports                         []sdk.Port    `json:"ports,omitempty"`
	Envs                          []interface{} `json:"envs,omitempty"`
	TTL                           string        `json:"ttl,omitempty"`
	StandbyGracePeriodSeconds     *int          `json:"standbyGracePeriodSeconds,omitempty"`
	TerminationGracePeriodSeconds *int          `json:"terminationGracePeriodSeconds,omitempty"`
}

type createSandboxLifecycle struct {
	ExpirationPolicies  []ExpirationPolicy `json:"expirationPolicies,omitempty"`
	TerminatedRetention string             `json:"terminatedRetention,omitempty"`
}

func buildCreateSandboxBody(opts CreateSandboxOptions) (createSandboxBody, error) {
	if err := opts.Validate(); err != nil {
		return createSandboxBody{}, err
	}
	parsedPorts, err := parseSandboxPorts(opts.Ports)
	if err != nil {
		return createSandboxBody{}, err
	}

	body := createSandboxBody{
		Metadata: createSandboxMetadata{Name: opts.Name, Labels: opts.Labels},
		Spec: createSandboxSpec{
			Region:  opts.Region,
			Volumes: opts.Volumes,
			Runtime: createSandboxRuntime{
				Image:                         opts.Image,
				Memory:                        int(opts.Memory),
				TTL:                           opts.TTL,
				StandbyGracePeriodSeconds:     opts.StandbyGracePeriodSeconds,
				TerminationGracePeriodSeconds: opts.TerminationGracePeriodSeconds,
			},
		},
	}

	defaultProtocol := "TCP"
	for i := range parsedPorts {
		body.Spec.Runtime.Ports = append(body.Spec.Runtime.Ports, sdk.Port{
			Target:   &parsedPorts[i],
			Protocol: &defaultProtocol,
		})
	}

	envs, err := tools.SetRuntimeEnv(opts.Env)
	if err != nil {
		return createSandboxBody{}, err
	}
	if envs != nil {
		body.Spec.Runtime.Envs = *envs
	}

	if len(opts.ExpirationPolicies) > 0 || opts.TerminatedRetention != "" {
		lifecycle := &createSandboxLifecycle{TerminatedRetention: opts.TerminatedRetention}
		for _, policy := range opts.ExpirationPolicies {
			if policy.Action == "" {
				policy.Action = ExpirationActionDelete
			}
			lifecycle.ExpirationPolicies = append(lifecycle.ExpirationPolicies, policy)
		}
		body.Spec.Lifecycle = lifecycle
	}
	return body, nil
}

// CreateSandboxWithOptions implements SandboxCreator.
func (h *SDKHandler) CreateSandboxWithOptions(ctx context.Context, opts CreateSandboxOptions) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}
	body, err := buildCreateSandboxBody(opts)
	if err != nil {
		return nil, err
	}
	requestBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode sandbox request: %w", err)
	}

	var editors []sdk.RequestEditorFn
	if opts.CreateIfNotExist {
		editors = append(editors, func(_ context.Context, req *http.Request) error {
			query := req.URL.Query()
			query.Set("createIfNotExist", "true")
			req.URL.RawQuery = query.Encode()
			return nil
		})
	}

	sandbox, err := h.sdkClient.CreateSandboxWithBodyWithResponse(ctx, "application/json", bytes.NewReader(requestBody), editors...)
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox: %w", err)
	}
	if sandbox.JSON200 == nil {
		return nil, createSandboxStatusError(opts.Name, sandbox.StatusCode(), sandbox.Body)
	}

	created := map[string]interface{}{
		"name": opts.Name,
	}
	if sandbox.JSON200.Status != nil {
		created["status"] = *sandbox.JSON200.Status
	}
	result := map[string]interface{}{
		"success":   true,
		"message":   fmt.Sprintf("Sandbox '%s' created successfully", opts.Name),
		"sandbox":   created,
		"nextSteps": CreateSandboxNextSteps(opts),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return jsonData, nil
}

// maxAPIErrorMessage bounds how much of an API error message is relayed.
const maxAPIErrorMessage = 500

// createSandboxStatusError reports a failed create. For a client error it
// relays the API's own message, which says what to change, such as
// "ttl 1m is too short: the minimum is 5m". Server error bodies are not
// relayed. A conflict is a name clash unless the API names another cause,
// such as a volume attached elsewhere or a deletion still in progress.
func createSandboxStatusError(name string, status int, body []byte) error {
	if status >= 400 && status < 500 {
		message := apiErrorMessage(body)
		if status == http.StatusConflict && (message == "" || strings.Contains(strings.ToLower(message), "already exist")) {
			return fmt.Errorf("sandbox with name '%s' already exists", name)
		}
		if message != "" {
			return fmt.Errorf("failed to create sandbox with status %d: %s", status, message)
		}
	}
	return fmt.Errorf("failed to create sandbox with status %d", status)
}

// apiErrorMessage reads the message of a Blaxel API error body, which is
// {"message": ...} for sandbox creation and {"error": ...} elsewhere.
func apiErrorMessage(body []byte) string {
	var decoded struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return ""
	}
	message := strings.TrimSpace(decoded.Message)
	if message == "" {
		message = strings.TrimSpace(decoded.Error)
	}
	if runes := []rune(message); len(runes) > maxAPIErrorMessage {
		message = string(runes[:maxAPIErrorMessage]) + "..."
	}
	return message
}

// DeleteSandbox implements SandboxHandler.DeleteSandbox
func (h *SDKHandler) DeleteSandbox(ctx context.Context, name string) ([]byte, error) {
	if h.sdkClient == nil {
		return nil, fmt.Errorf("SDK client not initialized")
	}

	resp, err := h.sdkClient.DeleteSandboxWithResponse(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete sandbox: %w", err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("sandbox '%s' not found", name)
	}
	if resp.StatusCode() < http.StatusOK || resp.StatusCode() >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("failed to delete sandbox '%s': status %d", name, resp.StatusCode())
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Sandbox '%s' deleted successfully", name),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}

	return jsonData, nil
}

// IsReadOnly implements SandboxHandlerWithReadOnly.IsReadOnly
func (h *SDKHandler) IsReadOnly() bool {
	return h.readOnly
}

// convertToSandboxModel converts an SDK sandbox to a simple sandbox model
func convertToSandboxModel(sandbox sdk.Sandbox) formatter.SandboxModel {
	model := formatter.SandboxModel{
		Name:   "",
		Status: "",
		Labels: make(map[string]string),
	}

	// Extract name
	if sandbox.Metadata != nil && sandbox.Metadata.Name != nil {
		model.Name = *sandbox.Metadata.Name
	}

	// Extract status
	if sandbox.Status != nil {
		model.Status = *sandbox.Status
	}

	// Extract labels
	if sandbox.Metadata != nil && sandbox.Metadata.Labels != nil {
		model.Labels = *sandbox.Metadata.Labels
	}

	// Extract runtime spec
	if sandbox.Spec != nil && sandbox.Spec.Runtime != nil {
		if sandbox.Spec.Runtime.Image != nil {
			model.Image = sandbox.Spec.Runtime.Image
		}
		if sandbox.Spec.Runtime.Generation != nil {
			model.Generation = sandbox.Spec.Runtime.Generation
		}
		if sandbox.Spec.Runtime.Memory != nil {
			model.Memory = sandbox.Spec.Runtime.Memory
		}
		if sandbox.Spec.Runtime.Ttl != nil {
			model.TTL = sandbox.Spec.Runtime.Ttl
		}
		if sandbox.Spec.Runtime.Expires != nil {
			// Parse the time string to time.Time
			if expires, err := time.Parse(time.RFC3339, *sandbox.Spec.Runtime.Expires); err == nil {
				model.Expires = &expires
			}
		}
		// Note: Ports conversion skipped due to SDK type complexity
		// if sandbox.Spec.Runtime.Ports != nil {
		// 	model.Ports = convertPorts(*sandbox.Spec.Runtime.Ports)
		// }
	}

	// Extract creation time
	if sandbox.Metadata != nil && sandbox.Metadata.CreatedAt != nil {
		// Parse the time string to time.Time
		if createdAt, err := time.Parse(time.RFC3339, *sandbox.Metadata.CreatedAt); err == nil {
			model.CreatedAt = &createdAt
		}
	}

	return model
}
