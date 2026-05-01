package cards

import (
	"encoding/json"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// AttachText returns a copy of result with the rendered card
// prepended as a TextContent block. The structured JSON payload
// is preserved byte-for-byte — card emission is strictly additive.
//
// Returning a copy (rather than mutating) keeps this helper safe
// to call on a shared *CallToolResult. When cards are globally
// disabled or the rendered card is empty, the original result is
// returned unchanged so callers can unconditionally wrap.
//
// Downstream PRs will wire this into each card-emitting tool. PR 0
// ships the helper but calls no tool code, so the surface stays
// invisible to agents until a per-tool PR lands.
func AttachText(result *mcp.CallToolResult, rendered string) *mcp.CallToolResult {
	if !Enabled() || rendered == "" {
		return result
	}

	textBlock := &mcp.TextContent{Text: rendered}
	if result == nil {
		return &mcp.CallToolResult{Content: []mcp.Content{textBlock}}
	}

	content := make([]mcp.Content, 0, len(result.Content)+1)
	content = append(content, textBlock)
	content = append(content, result.Content...)

	return &mcp.CallToolResult{
		Content:           content,
		IsError:           result.IsError,
		Meta:              result.Meta,
		StructuredContent: result.StructuredContent,
	}
}

// Attach is AttachText's companion for tools that return a
// StructuredContent value instead of a pre-built *CallToolResult.
// It produces a fresh result with the card text and a JSON-encoded
// copy of structured as the second content block, matching how the
// MCP SDK itself serializes structured outputs. Downstream tools
// that already build their own *CallToolResult should use
// AttachText; Attach is the convenience for the majority of tools
// that currently rely on the SDK's default structured encoding.
func Attach(rendered string, structured any) (*mcp.CallToolResult, error) {
	if !Enabled() || rendered == "" {
		// No card — leave the structured-only path to the SDK.
		return nil, nil
	}
	payload, err := json.Marshal(structured)
	if err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: rendered},
			&mcp.TextContent{Text: string(payload)},
		},
		StructuredContent: json.RawMessage(payload),
	}, nil
}
