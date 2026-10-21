package sandboxes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Expiration policy types and the only action the API accepts.
const (
	ExpirationPolicyTTLIdle   = "ttl-idle"
	ExpirationPolicyTTLMaxAge = "ttl-max-age"
	ExpirationPolicyDate      = "date"
	ExpirationActionDelete    = "delete"
)

// Volume attachment types. An empty type means persistent.
const (
	VolumeTypePersistent = "persistent"
	VolumeTypeEphemeral  = "ephemeral"
)

// ExpirationPolicy is one spec.lifecycle.expirationPolicies entry of the
// create-sandbox request.
type ExpirationPolicy struct {
	Type   string `json:"type"`
	Value  string `json:"value"`
	Action string `json:"action,omitempty"`
}

// VolumeAttachment is one spec.volumes entry of the create-sandbox request.
type VolumeAttachment struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
	Type      string `json:"type,omitempty"`
	SizeMB    int    `json:"sizeMb,omitempty"`
}

// CreateSandboxOptions holds every create_sandbox argument. Zero values mean
// the argument was omitted and the API default applies.
type CreateSandboxOptions struct {
	Name   string
	Image  string
	Memory float64
	Region string
	// Ports is the comma-separated list of ports to expose.
	Ports string
	// Env is the comma-separated list of NAME=VALUE entries.
	Env    string
	Labels map[string]string

	TTL                           string
	ExpirationPolicies            []ExpirationPolicy
	TerminatedRetention           string
	Volumes                       []VolumeAttachment
	StandbyGracePeriodSeconds     *int
	TerminationGracePeriodSeconds *int

	// CreateIfNotExist returns a live sandbox with the same name, unchanged,
	// instead of failing with a conflict.
	CreateIfNotExist bool
}

// SandboxCreator is implemented by sandbox handlers that accept every
// create_sandbox argument. create_sandbox uses it when available. A handler
// that only implements the legacy CreateSandbox method cannot receive
// lifecycle, volume or label settings, so create_sandbox rejects those
// arguments for it instead of silently dropping them.
type SandboxCreator interface {
	CreateSandboxWithOptions(ctx context.Context, opts CreateSandboxOptions) ([]byte, error)
}

type createSandboxArguments struct {
	Name                          string            `json:"name"`
	Image                         string            `json:"image"`
	Memory                        *float64          `json:"memory"`
	Ports                         string            `json:"ports"`
	Env                           string            `json:"env"`
	Workspace                     string            `json:"workspace"`
	Labels                        map[string]string `json:"labels"`
	TTL                           string            `json:"ttl"`
	ExpirationPolicies            json.RawMessage   `json:"expirationPolicies"`
	TerminatedRetention           string            `json:"terminatedRetention"`
	Volumes                       json.RawMessage   `json:"volumes"`
	StandbyGracePeriodSeconds     *int              `json:"standbyGracePeriodSeconds"`
	TerminationGracePeriodSeconds *int              `json:"terminationGracePeriodSeconds"`
	CreateIfNotExist              bool              `json:"createIfNotExist"`
}

// ParseCreateSandboxArguments turns raw create_sandbox tool arguments into
// options and rejects malformed input before any API call. Duration formats,
// minimums and quotas are left to the API, whose error says what to change.
func ParseCreateSandboxArguments(args map[string]any) (CreateSandboxOptions, error) {
	// JSON encoding rejects non-finite floats before it can name the field.
	if rawMemory, ok := args["memory"].(float64); ok {
		if err := validateSandboxMemory(rawMemory); err != nil {
			return CreateSandboxOptions{}, err
		}
	}
	region, err := ParseRegionArgument(args)
	if err != nil {
		return CreateSandboxOptions{}, err
	}

	data, err := json.Marshal(args)
	if err != nil {
		return CreateSandboxOptions{}, fmt.Errorf("invalid create_sandbox arguments: %v", err)
	}
	var raw createSandboxArguments
	if err := json.Unmarshal(data, &raw); err != nil {
		return CreateSandboxOptions{}, fmt.Errorf("invalid create_sandbox arguments: %v", err)
	}
	if raw.Memory != nil {
		if err := validateSandboxMemory(*raw.Memory); err != nil {
			return CreateSandboxOptions{}, err
		}
	}

	opts := CreateSandboxOptions{
		Name:                          raw.Name,
		Image:                         raw.Image,
		Region:                        region,
		Ports:                         raw.Ports,
		Env:                           raw.Env,
		Labels:                        raw.Labels,
		TTL:                           raw.TTL,
		TerminatedRetention:           raw.TerminatedRetention,
		StandbyGracePeriodSeconds:     raw.StandbyGracePeriodSeconds,
		TerminationGracePeriodSeconds: raw.TerminationGracePeriodSeconds,
		CreateIfNotExist:              raw.CreateIfNotExist,
	}
	if raw.Memory != nil {
		opts.Memory = *raw.Memory
	}
	if err := decodeStrict("expirationPolicies", raw.ExpirationPolicies, &opts.ExpirationPolicies); err != nil {
		return CreateSandboxOptions{}, err
	}
	for i := range opts.ExpirationPolicies {
		if opts.ExpirationPolicies[i].Action == "" {
			opts.ExpirationPolicies[i].Action = ExpirationActionDelete
		}
	}
	if err := decodeStrict("volumes", raw.Volumes, &opts.Volumes); err != nil {
		return CreateSandboxOptions{}, err
	}

	if err := opts.Validate(); err != nil {
		return CreateSandboxOptions{}, err
	}
	return opts, nil
}

// decodeStrict decodes a nested argument and rejects unknown fields, so a
// misspelled key such as mount_path fails instead of being ignored.
func decodeStrict(field string, data json.RawMessage, target any) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid %s: %v", field, err)
	}
	return nil
}

// Validate checks the shape of the options. It does not repeat the API's
// duration and quota rules.
func (o CreateSandboxOptions) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("sandbox name is required")
	}
	if o.Memory != 0 {
		if err := validateSandboxMemory(o.Memory); err != nil {
			return err
		}
	}
	if o.Region != strings.TrimSpace(o.Region) {
		return fmt.Errorf("region must be a non-empty region identifier")
	}
	if _, err := parseSandboxPorts(o.Ports); err != nil {
		return err
	}
	for i, policy := range o.ExpirationPolicies {
		if policy.Type == "" {
			return fmt.Errorf("expirationPolicies[%d].type is required: use %s, %s or %s", i, ExpirationPolicyTTLIdle, ExpirationPolicyTTLMaxAge, ExpirationPolicyDate)
		}
		if policy.Value == "" {
			return fmt.Errorf("expirationPolicies[%d].value is required: a duration such as 24h, or an ISO 8601 time for type date", i)
		}
	}
	for i, volume := range o.Volumes {
		if volume.Name == "" {
			return fmt.Errorf("volumes[%d].name is required", i)
		}
		if volume.MountPath == "" {
			return fmt.Errorf("volumes[%d].mountPath is required, for example /data", i)
		}
		if volume.Type == VolumeTypeEphemeral && volume.SizeMB <= 0 {
			return fmt.Errorf("volumes[%d].sizeMb must be greater than 0 for an ephemeral volume", i)
		}
	}
	if o.StandbyGracePeriodSeconds != nil && *o.StandbyGracePeriodSeconds < 0 {
		return fmt.Errorf("standbyGracePeriodSeconds must be 0 or more")
	}
	if o.TerminationGracePeriodSeconds != nil && *o.TerminationGracePeriodSeconds < 0 {
		return fmt.Errorf("terminationGracePeriodSeconds must be 0 or more")
	}
	return nil
}

// extendedSettings names the options the legacy CreateSandbox and
// CreateSandboxInRegion methods cannot carry.
func (o CreateSandboxOptions) extendedSettings() []string {
	var names []string
	add := func(set bool, name string) {
		if set {
			names = append(names, name)
		}
	}
	add(len(o.Labels) > 0, "labels")
	add(o.TTL != "", "ttl")
	add(len(o.ExpirationPolicies) > 0, "expirationPolicies")
	add(o.TerminatedRetention != "", "terminatedRetention")
	add(len(o.Volumes) > 0, "volumes")
	add(o.StandbyGracePeriodSeconds != nil, "standbyGracePeriodSeconds")
	add(o.TerminationGracePeriodSeconds != nil, "terminationGracePeriodSeconds")
	add(o.CreateIfNotExist, "createIfNotExist")
	return names
}

func (o CreateSandboxOptions) hasExpiration() bool {
	return o.TTL != "" || len(o.ExpirationPolicies) > 0
}

// createSandbox sends opts to the richest create method the handler has.
func createSandbox(ctx context.Context, handler SandboxHandler, opts CreateSandboxOptions) ([]byte, error) {
	if creator, ok := handler.(SandboxCreator); ok {
		return creator.CreateSandboxWithOptions(ctx, opts)
	}
	if unsupported := opts.extendedSettings(); len(unsupported) > 0 {
		return nil, fmt.Errorf("this server cannot apply %s when creating a sandbox; remove them and try again", strings.Join(unsupported, ", "))
	}
	if opts.Region != "" {
		regional, ok := handler.(RegionalSandboxHandler)
		if !ok {
			return nil, fmt.Errorf("explicit region is not supported by this sandbox handler")
		}
		return regional.CreateSandboxInRegion(ctx, opts.Name, opts.Image, opts.Memory, opts.Ports, opts.Env, opts.Region)
	}
	return handler.CreateSandbox(ctx, opts.Name, opts.Image, opts.Memory, opts.Ports, opts.Env)
}

// CreateSandboxNextSteps returns short instructions for an agent that just
// created a sandbox with opts: how to use it, how to reach its ports, and when
// it is deleted.
func CreateSandboxNextSteps(opts CreateSandboxOptions) []string {
	steps := []string{
		fmt.Sprintf("Run commands with run_sandbox_command (name %q) once its status is DEPLOYED; check with get_sandbox.", opts.Name),
	}
	if ports, err := parseSandboxPorts(opts.Ports); err == nil && len(ports) > 0 {
		steps = append(steps, fmt.Sprintf("To reach port %d from a browser, create a preview URL for it with the Blaxel SDK or API (https://docs.blaxel.ai/Sandboxes/Preview-url), and have the server listen on $HOST.", ports[0]))
	}
	if opts.hasExpiration() {
		var rules []string
		if opts.TTL != "" {
			rules = append(rules, fmt.Sprintf("%s after creation (ttl)", opts.TTL))
		}
		for _, policy := range opts.ExpirationPolicies {
			switch policy.Type {
			case ExpirationPolicyTTLIdle:
				rules = append(rules, fmt.Sprintf("%s without activity (ttl-idle)", policy.Value))
			case ExpirationPolicyTTLMaxAge:
				rules = append(rules, fmt.Sprintf("%s after creation (ttl-max-age)", policy.Value))
			case ExpirationPolicyDate:
				rules = append(rules, fmt.Sprintf("at %s (date)", policy.Value))
			default:
				rules = append(rules, fmt.Sprintf("%s %s", policy.Type, policy.Value))
			}
		}
		steps = append(steps, fmt.Sprintf("The sandbox is deleted automatically at the first of: %s. Call delete_sandbox to remove it sooner.", strings.Join(rules, "; ")))
	} else {
		steps = append(steps, "No ttl or expiration policy is set: when idle the sandbox goes to standby and is kept until you call delete_sandbox, unless your plan tier enforces a maximum lifetime.")
	}
	for _, volume := range opts.Volumes {
		if volume.Type != VolumeTypeEphemeral {
			steps = append(steps, fmt.Sprintf("Files under %s are stored on volume %q and stay there after the sandbox is deleted.", volume.MountPath, volume.Name))
		}
	}
	if opts.CreateIfNotExist {
		steps = append(steps, fmt.Sprintf("If a sandbox named %q already existed, it was returned unchanged and the settings in this call were not applied; check them with get_sandbox.", opts.Name))
	}
	return steps
}

// CreateSandboxErrorHint returns what to change after a failed create, or ""
// when the error gives no lifecycle, quota, volume or conflict clue.
func CreateSandboxErrorHint(message string, opts CreateSandboxOptions) string {
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "too short"):
		return "Set ttl and ttl-max-age to at least 5m and ttl-idle to at least 60s. To remove a sandbox sooner, call delete_sandbox when you are done with it."
	case strings.Contains(lower, "quota exceeded") || strings.Contains(lower, "greater than default expiration"):
		return quotaErrorHint(lower)
	case strings.Contains(lower, "must be in the future") || strings.Contains(lower, "date format"):
		return "Give date policies a future ISO 8601 time with a time zone, for example 2030-01-01T00:00:00Z."
	case strings.Contains(lower, "can only appear once"):
		return "List each expirationPolicies type at most once."
	case strings.Contains(lower, "format") && (strings.Contains(lower, "ttl") || strings.Contains(lower, "duration") || strings.Contains(lower, "retention")):
		return "Write durations as a number and a unit (s, m, h, d or w), for example 30m, 24h or 7d."
	case strings.Contains(lower, "deletion is still in progress"):
		return "A sandbox with this name is still being deleted: retry in a few seconds or choose another name."
	case strings.Contains(lower, "already exists"):
		return "Choose another name, or pass createIfNotExist: true if the existing sandbox is the one you want. Delete it with delete_sandbox only if you created it in this session and no longer need it; otherwise ask the user first."
	case strings.Contains(lower, "volume"):
		return "Attach one volume that already exists in the same workspace and region and is not attached to another sandbox. To free a volume, delete the sandbox using it only if you created that sandbox in this session and no longer need it; otherwise ask the user first. Ephemeral volumes also need sizeMb."
	case opts.hasExpiration() && (strings.Contains(lower, "400") || strings.Contains(lower, "invalid")):
		return "If the lifecycle settings caused this: ttl and ttl-max-age need at least 5m, ttl-idle at least 60s, durations use s, m, h, d or w, and date needs a future ISO 8601 time."
	}
	return ""
}

// quotaErrorHint returns the next step for a quota error. Only the plan's
// lifetime cap ("TTL quota exceeded") is fixed by a shorter ttl; the sandbox
// count quota ("Sandbox quota exceeded", "Quota exceeded: 5/5 sandboxes")
// needs a free slot. Other quotas get no hint beyond the API message.
func quotaErrorHint(lower string) string {
	switch {
	case strings.Contains(lower, "ttl quota exceeded") || strings.Contains(lower, "greater than default expiration"):
		return "Your plan tier caps sandbox lifetime: use a shorter ttl or an earlier date, or omit them to get the plan default."
	case strings.Contains(lower, "sandbox quota exceeded") || strings.Contains(lower, " sandboxes"):
		return "Your account has reached its sandbox limit: delete a sandbox you created in this session and no longer need with delete_sandbox, or raise the quota at https://app.blaxel.ai/account/plan-quotas."
	}
	return ""
}

// withCreateSandboxErrorHint appends the next step to an error message.
func withCreateSandboxErrorHint(message string, opts CreateSandboxOptions) string {
	if hint := CreateSandboxErrorHint(message, opts); hint != "" {
		return message + "\nNext step: " + hint
	}
	return message
}
