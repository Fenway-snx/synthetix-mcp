package cards

import (
	"encoding/json"
	"testing"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Confirms card text prepends without mutating structured content.
func TestAttachTextPrependsCardPreservingStructured(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	structured := json.RawMessage(`{"ok":true}`)
	original := &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: "json-body"}},
		StructuredContent: structured,
	}
	out := AttachText(original, "RENDERED CARD\n")
	if out == original {
		t.Fatal("AttachText must return a new result, not mutate the input")
	}
	if len(out.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(out.Content))
	}
	if first, ok := out.Content[0].(*mcp.TextContent); !ok || first.Text != "RENDERED CARD\n" {
		t.Fatalf("first content block must be the card; got %#v", out.Content[0])
	}
	if second, ok := out.Content[1].(*mcp.TextContent); !ok || second.Text != "json-body" {
		t.Fatalf("second block must be the original content; got %#v", out.Content[1])
	}
	gotStructured, ok := out.StructuredContent.(json.RawMessage)
	if !ok || string(gotStructured) != string(structured) {
		t.Fatalf("StructuredContent must survive byte-for-byte; got %v", out.StructuredContent)
	}
}

// Confirms the env kill switch returns results unchanged.
func TestAttachTextNoOpsWhenCardsDisabled(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "false")

	original := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "json-body"}},
	}
	out := AttachText(original, "RENDERED CARD\n")
	if out != original {
		t.Fatal("disabled cards must return the original result unchanged")
	}
}

// Protects callers that only sometimes have card text to show.
func TestAttachTextNoOpsOnEmptyCard(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	original := &mcp.CallToolResult{}
	if out := AttachText(original, ""); out != original {
		t.Fatal("empty card must return the original result unchanged")
	}
}

// Keeps nil results safe when only card text is available.
func TestAttachTextHandlesNilResult(t *testing.T) {
	t.Setenv("SNXMCP_CARDS_ENABLED", "true")

	out := AttachText(nil, "RENDERED CARD\n")
	if out == nil {
		t.Fatal("AttachText(nil, rendered) must return a new result")
	}
	if len(out.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(out.Content))
	}
}
