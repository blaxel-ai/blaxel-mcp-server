package tools

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestStructureResponse(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want string
	}{
		{"object passes through", `{"a":1}`, `{"a":1}`},
		{"array is wrapped", `[{"a":1},{"a":2}]`, `{"count":2,"items":[{"a":1},{"a":2}]}`},
		{"null becomes an empty list", `null`, `{"count":0,"items":[]}`},
		{"text is wrapped", "Found 1 agent(s):\n", `{"text":"Found 1 agent(s):\n"}`},
		{"scalar is wrapped as text", `"ok"`, `{"text":"\"ok\""}`},
		{"trailing data is text", `{"a":1} extra`, `{"text":"{\"a\":1} extra"}`},
		{"empty is text", ``, `{"text":""}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertJSONEqual(t, StructureResponse([]byte(tc.raw)), tc.want)
		})
	}
}

func TestStructureBody(t *testing.T) {
	assertJSONEqual(t, StructureBody(`{"output":"hi"}`), `{"format":"json","body":{"output":"hi"}}`)
	assertJSONEqual(t, StructureBody(`["a"]`), `{"format":"json","body":["a"]}`)
	assertJSONEqual(t, StructureBody("hello"), `{"format":"text","body":"hello"}`)
}

func TestStructuredResultKeepsTextUnchanged(t *testing.T) {
	raw := "[\n  {\"a\": 1}\n]"
	result := StructuredResult([]byte(raw))
	if len(result.Content) != 1 {
		t.Fatalf("got %d content blocks, want 1", len(result.Content))
	}
	block, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content is %T, want text", result.Content[0])
	}
	if block.Text != raw {
		t.Fatalf("text = %q, want %q", block.Text, raw)
	}
}

func assertJSONEqual(t *testing.T, got any, want string) {
	t.Helper()
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var gotValue, wantValue any
	if err := json.Unmarshal(encoded, &gotValue); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("got %s, want %s", encoded, want)
	}
}
