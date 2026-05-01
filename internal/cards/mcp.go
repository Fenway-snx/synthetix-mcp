package cards

import (
	"encoding/json"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Prepends a rendered card while preserving structured JSON unchanged.
// Returning a copy keeps wrapping safe for shared tool results.
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

// Builds a fresh tool result with card text plus JSON structured content.
// Tools with an existing result should use the text-only wrapper.
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
