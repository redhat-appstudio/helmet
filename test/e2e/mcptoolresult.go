package e2e

import (
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// ToolResult wraps a CallToolResult with convenience accessors.
type ToolResult struct {
	mcp.CallToolResult
}

// Text extracts the concatenated text from all TextContent entries.
// Returns empty string if no text content is present.
func (r ToolResult) Text() string {
	var sb strings.Builder
	for _, c := range r.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String()
}
