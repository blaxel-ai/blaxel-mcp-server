package tools

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// The MCP tools specification requires a tool that declares an outputSchema
// to return structuredContent conforming to it, and structuredContent must be
// a JSON object. The helpers below turn the bytes a handler already returns
// into such an object while leaving the text content block exactly as it was,
// so clients that only read text see no change.
//
// Structured content is always derived from the same bytes as the text block.
// That keeps secret handling in one place: whatever a handler masks in its
// text response is masked in the structured response too, and nothing a
// handler omits can reappear.

// Keys used when a handler response cannot be returned as an object as-is.
const (
	// ItemsKey wraps a JSON array response.
	ItemsKey = "items"
	// CountKey accompanies ItemsKey with the number of wrapped items.
	CountKey = "count"
	// TextKey wraps a response that is not JSON, such as a formatted summary.
	TextKey = "text"
	// BodyKey wraps an opaque runtime response body of any JSON type.
	BodyKey = "body"
	// FormatKey reports whether BodyKey held JSON or plain text.
	FormatKey = "format"
)

// StructuredResult returns a tool result whose text block is the handler
// output unchanged and whose structuredContent is that output as an object.
// A JSON object is used directly, a JSON array becomes {"items": [...],
// "count": n}, and anything else becomes {"text": "..."}.
func StructuredResult(raw []byte) *mcp.CallToolResult {
	return mcp.NewToolResultStructured(StructureResponse(raw), string(raw))
}

// StructuredBodyResult returns a tool result for an opaque runtime response,
// such as an agent or model reply. The text block is unchanged, and
// structuredContent is {"format": "json", "body": <value>} when the response
// is JSON or {"format": "text", "body": "..."} when it is not.
func StructuredBodyResult(raw string) *mcp.CallToolResult {
	return mcp.NewToolResultStructured(StructureBody(raw), raw)
}

// StructureResponse converts handler output into a JSON object suitable for
// structuredContent. See StructuredResult for the rules.
func StructureResponse(raw []byte) map[string]any {
	value, ok := decodeJSON(raw)
	if !ok {
		return map[string]any{TextKey: string(raw)}
	}
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case []any:
		return map[string]any{ItemsKey: typed, CountKey: len(typed)}
	case nil:
		return map[string]any{ItemsKey: []any{}, CountKey: 0}
	default:
		return map[string]any{TextKey: string(raw)}
	}
}

// StructureBody converts an opaque response body into a JSON object suitable
// for structuredContent. See StructuredBodyResult for the rules.
func StructureBody(raw string) map[string]any {
	if value, ok := decodeJSON([]byte(raw)); ok {
		return map[string]any{FormatKey: "json", BodyKey: value}
	}
	return map[string]any{FormatKey: "text", BodyKey: raw}
}

func decodeJSON(raw []byte) (any, bool) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, false
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, false
	}
	// Reject trailing data so "{...} extra" is treated as text, not as JSON.
	if decoder.More() {
		return nil, false
	}
	return value, true
}

// OutputSchema returns a tool option that declares schema as the tool's
// outputSchema. It panics on a schema that cannot be marshalled, which can
// only happen through a programming error in a static schema literal.
func OutputSchema(schema map[string]any) mcp.ToolOption {
	data, err := json.Marshal(schema)
	if err != nil {
		panic(fmt.Sprintf("invalid output schema: %v", err))
	}
	return mcp.WithRawOutputSchema(data)
}

// Schema building blocks. The schemas describe the fields clients can rely
// on without pinning every field the API returns: every object allows
// additional properties, and every described property also accepts null,
// because the Blaxel API and the hosted control plane serialize absent values
// both by omitting them and as null. Only the top-level result object itself
// is never null.

// ObjectSchema returns an open object schema with the given properties. Use it
// for a tool's top-level result; use NestedObjectSchema for properties.
func ObjectSchema(description string, properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": true,
	}
	if description != "" {
		schema["description"] = description
	}
	if properties != nil {
		schema["properties"] = properties
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// NestedObjectSchema returns an open object property schema that may be null.
func NestedObjectSchema(description string, properties map[string]any) map[string]any {
	schema := ObjectSchema(description, properties)
	schema["type"] = []string{"object", "null"}
	return schema
}

func nullableScalar(jsonType, description string) map[string]any {
	schema := map[string]any{"type": []string{jsonType, "null"}}
	if description != "" {
		schema["description"] = description
	}
	return schema
}

// StringSchema returns a string property schema that may be null.
func StringSchema(description string) map[string]any {
	return nullableScalar("string", description)
}

// BooleanSchema returns a boolean property schema that may be null.
func BooleanSchema(description string) map[string]any {
	return nullableScalar("boolean", description)
}

// IntegerSchema returns an integer property schema that may be null.
func IntegerSchema(description string) map[string]any {
	return nullableScalar("integer", description)
}

// NumberSchema returns a number property schema that may be null.
func NumberSchema(description string) map[string]any {
	return nullableScalar("number", description)
}

// ArraySchema returns an array property schema that may be null.
func ArraySchema(description string, items map[string]any) map[string]any {
	schema := nullableScalar("array", description)
	schema["items"] = items
	return schema
}

// AnySchema returns a schema that accepts any JSON value.
func AnySchema(description string) map[string]any {
	return map[string]any{"description": description}
}

// StringMapSchema returns an object property schema whose values are strings.
func StringMapSchema(description string) map[string]any {
	schema := nullableScalar("object", description)
	schema["additionalProperties"] = map[string]any{"type": []string{"string", "null"}}
	return schema
}

// MetadataSchema describes the metadata block shared by Blaxel resources.
func MetadataSchema() map[string]any {
	return NestedObjectSchema("Resource metadata", map[string]any{
		"name":        StringSchema("Unique resource name within the workspace"),
		"displayName": StringSchema("Human-readable display name"),
		"workspace":   StringSchema("Workspace that owns the resource"),
		"labels":      StringMapSchema("Resource labels"),
		"url":         StringSchema("Resource URL, when the resource is reachable over HTTP"),
		"createdAt":   StringSchema("Creation timestamp"),
		"updatedAt":   StringSchema("Last update timestamp"),
		"createdBy":   StringSchema("Creator identifier"),
		"updatedBy":   StringSchema("Last updater identifier"),
	})
}

// ResourceProperties returns the metadata, spec, status and events properties
// shared by Blaxel resources. specProperties lists the spec fields worth
// describing; the spec itself stays open to every field the API returns.
func ResourceProperties(specProperties map[string]any) map[string]any {
	return map[string]any{
		"metadata": MetadataSchema(),
		"spec":     NestedObjectSchema("Resource specification", specProperties),
		"status":   StringSchema("Deployment status, for example DEPLOYED, DEPLOYING, FAILED or TERMINATED"),
		"events":   ArraySchema("Recent lifecycle events", NestedObjectSchema("Lifecycle event", nil)),
	}
}

// ResourceSchema returns a top-level schema for one Blaxel resource.
func ResourceSchema(description string, specProperties map[string]any) map[string]any {
	return ObjectSchema(description, ResourceProperties(specProperties))
}

// NestedResourceSchema returns a nullable property schema for one resource.
func NestedResourceSchema(description string, specProperties map[string]any) map[string]any {
	return NestedObjectSchema(description, ResourceProperties(specProperties))
}

// RuntimeSpecProperties describes the runtime block of deployable resources.
func RuntimeSpecProperties() map[string]any {
	return map[string]any{
		"enabled":     BooleanSchema("Whether the resource is enabled"),
		"description": StringSchema("Resource description"),
		"runtime": NestedObjectSchema("Runtime configuration", map[string]any{
			"image":      StringSchema("Container image"),
			"memory":     IntegerSchema("Memory in MB"),
			"generation": StringSchema("Infrastructure generation"),
			"type":       StringSchema("Runtime type"),
			"envs":       ArraySchema("Environment variables; secret values are masked", NestedObjectSchema("Environment variable", nil)),
		}),
		"integrationConnections": ArraySchema("Integration connections used by the resource", StringSchema("Integration connection name")),
	}
}

// ListSchema describes a list result. Depending on the server, a list comes
// back as items (a JSON array response), as a readable summary in text, or as
// a readable summary in content with a count. extra adds tool-specific
// properties.
func ListSchema(description string, item map[string]any, extra map[string]any) map[string]any {
	properties := map[string]any{
		ItemsKey:  ArraySchema("Resources in the workspace, after any filter is applied", item),
		CountKey:  IntegerSchema("Number of resources returned"),
		TextKey:   StringSchema("Readable summary, returned when the server formats the list as text"),
		"content": StringSchema("Readable summary, returned with count when the server formats the list as text"),
	}
	for key, value := range extra {
		properties[key] = value
	}
	return ObjectSchema(description, properties)
}

// MutationSchema describes the {success, message, ...} object that create,
// update and delete tools return. extra adds tool-specific properties.
func MutationSchema(description string, extra map[string]any) map[string]any {
	properties := map[string]any{
		"success": map[string]any{"type": "boolean", "description": "Whether the operation succeeded"},
		"message": map[string]any{"type": "string", "description": "Human-readable outcome of the operation"},
	}
	for key, value := range extra {
		properties[key] = value
	}
	return ObjectSchema(description, properties, "success", "message")
}

// BodySchema describes an opaque runtime response wrapped by StructureBody.
func BodySchema(description string) map[string]any {
	return ObjectSchema(description, map[string]any{
		FormatKey: map[string]any{
			"type":        "string",
			"enum":        []string{"json", "text"},
			"description": "json when body holds the parsed JSON response, text when the response was not JSON",
		},
		BodyKey: AnySchema("Response body returned by the invoked resource: the parsed JSON value when format is json, otherwise the raw text"),
	}, FormatKey, BodyKey)
}
