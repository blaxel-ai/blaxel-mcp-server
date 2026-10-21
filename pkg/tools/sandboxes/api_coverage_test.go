package sandboxes

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	blaxel "github.com/blaxel-ai/sdk-go"
)

// The create-sandbox request fields come from the Go SDK, which is generated
// from the Blaxel OpenAPI spec. When a dependency update brings a new field,
// TestCreateSandboxCoversAPIFields fails until the field is exposed as a
// create_sandbox parameter or listed below with the reason it is left out.

// createSandboxAPIFields maps each exposed request field to the create_sandbox
// parameter that sets it. A dotted parameter reaches into array items.
var createSandboxAPIFields = map[string]string{
	"query.createIfNotExist":                     "createIfNotExist",
	"metadata.name":                              "name",
	"metadata.labels":                            "labels",
	"spec.region":                                "region",
	"spec.runtime.image":                         "image",
	"spec.runtime.memory":                        "memory",
	"spec.runtime.ports":                         "ports",
	"spec.runtime.envs":                          "env",
	"spec.runtime.ttl":                           "ttl",
	"spec.runtime.terminationGracePeriodSeconds": "terminationGracePeriodSeconds",
	"spec.lifecycle.terminatedRetention":         "terminatedRetention",
	"spec.lifecycle.expirationPolicies.type":     "expirationPolicies.type",
	"spec.lifecycle.expirationPolicies.value":    "expirationPolicies.value",
	"spec.lifecycle.expirationPolicies.action":   "expirationPolicies.action",
	"spec.volumes.name":                          "volumes.name",
	"spec.volumes.mountPath":                     "volumes.mountPath",
	"spec.volumes.readOnly":                      "volumes.readOnly",
	"spec.volumes.type":                          "volumes.type",
	"spec.volumes.sizeMb":                        "volumes.sizeMb",
}

// createSandboxFieldsLeftOut lists request fields create_sandbox deliberately
// does not expose.
var createSandboxFieldsLeftOut = map[string]string{
	"metadata.displayName":   "the sandbox name is shown instead",
	"metadata.externalId":    "no sandbox tool looks a sandbox up by external ID",
	"spec.enabled":           "a disabled sandbox cannot run commands",
	"spec.vpc":               "network placement is configured with the SDK, CLI or console",
	"spec.network":           "proxy, firewall and egress rules can carry secrets and are configured with the SDK, CLI or console",
	"spec.runtime.expires":   "expirationPolicies with type date sets the same deletion time",
	"spec.runtime.extraArgs": "kernel variants depend on the infrastructure generation and are rarely needed",
}

// isLifecycleField reports whether a request field controls when a sandbox is
// deleted or what storage it keeps. These must always be exposed.
func isLifecycleField(path string) bool {
	return path == "spec.runtime.ttl" || path == "query.createIfNotExist" ||
		strings.HasPrefix(path, "spec.lifecycle.") || strings.HasPrefix(path, "spec.volumes.")
}

const sdkParamPackage = "github.com/blaxel-ai/sdk-go/packages/param"

// createSandboxRequestFields walks the SDK's create-sandbox parameters and
// returns every field path. It does not descend into listed fields.
func createSandboxRequestFields() []string {
	listed := func(path string) bool {
		_, exposed := createSandboxAPIFields[path]
		_, leftOut := createSandboxFieldsLeftOut[path]
		return exposed || leftOut
	}
	var paths []string
	var walk func(t reflect.Type, prefix string)
	walk = func(t reflect.Type, prefix string) {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			var name string
			if query := strings.Split(field.Tag.Get("query"), ",")[0]; query != "" {
				name = "query." + query
			} else if jsonName := strings.Split(field.Tag.Get("json"), ",")[0]; jsonName != "" && jsonName != "-" {
				name = prefix + jsonName
			} else {
				continue
			}
			inner := field.Type
			for inner.Kind() == reflect.Slice || inner.Kind() == reflect.Pointer {
				inner = inner.Elem()
			}
			if !listed(name) && inner.Kind() == reflect.Struct && inner.PkgPath() != sdkParamPackage {
				walk(inner, name+".")
				continue
			}
			paths = append(paths, name)
		}
	}
	walk(reflect.TypeOf(blaxel.SandboxNewParams{}), "")
	walk(reflect.TypeOf(blaxel.SandboxParam{}), "")
	sort.Strings(paths)
	return paths
}

func TestCreateSandboxCoversAPIFields(t *testing.T) {
	fields := createSandboxRequestFields()
	if len(fields) < 10 {
		t.Fatalf("found only %d create-sandbox request fields in the SDK: %v", len(fields), fields)
	}

	seen := map[string]bool{}
	for _, path := range fields {
		seen[path] = true
		_, exposed := createSandboxAPIFields[path]
		_, leftOut := createSandboxFieldsLeftOut[path]
		switch {
		case isLifecycleField(path) && !exposed:
			t.Errorf("create-sandbox lifecycle field %s must be a create_sandbox parameter", path)
		case !exposed && !leftOut:
			t.Errorf("create-sandbox field %s is not a create_sandbox parameter: expose it, or add it to createSandboxFieldsLeftOut with the reason", path)
		}
	}
	for _, listing := range []map[string]string{createSandboxAPIFields, createSandboxFieldsLeftOut} {
		for path := range listing {
			if !seen[path] {
				t.Errorf("%s is listed but is no longer a create-sandbox request field", path)
			}
		}
	}

	mcpServer, _ := newCreatorToolTestServer(nil)
	properties := mcpServer.GetTool("create_sandbox").Tool.InputSchema.Properties
	for path, parameter := range createSandboxAPIFields {
		if !schemaHasParameter(properties, parameter) {
			t.Errorf("create_sandbox has no %s parameter for API field %s", parameter, path)
		}
	}
}

func schemaHasParameter(properties map[string]any, parameter string) bool {
	name, rest, nested := strings.Cut(parameter, ".")
	property, ok := properties[name].(map[string]any)
	if !ok {
		return false
	}
	if !nested {
		return true
	}
	items, _ := property["items"].(map[string]any)
	itemProperties, _ := items["properties"].(map[string]any)
	return schemaHasParameter(itemProperties, rest)
}

// The request the SDK handler sends must carry each exposed field under the
// API's own name.
func TestCreateSandboxRequestBodyUsesAPIFieldNames(t *testing.T) {
	opts, err := ParseCreateSandboxArguments(lifecycleArguments())
	if err != nil {
		t.Fatal(err)
	}
	body, err := buildCreateSandboxBody(opts)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	var request map[string]any
	if err := json.Unmarshal(encoded, &request); err != nil {
		t.Fatal(err)
	}

	for path := range createSandboxAPIFields {
		if strings.HasPrefix(path, "query.") {
			continue
		}
		if !hasJSONField(request, strings.Split(path, ".")) {
			t.Errorf("request body has no %s: %s", path, encoded)
		}
	}
}

// hasJSONField reports whether value has a non-null field at path, looking
// through every item of the arrays on the way.
func hasJSONField(value any, path []string) bool {
	if len(path) == 0 {
		return value != nil
	}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if hasJSONField(item, path) {
				return true
			}
		}
	case map[string]any:
		return hasJSONField(typed[path[0]], path[1:])
	}
	return false
}
