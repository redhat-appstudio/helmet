package e2e

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	o "github.com/onsi/gomega"
)

func TestToolResult_Text(t *testing.T) {
	g := o.NewWithT(t)

	r := ToolResult{mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: "hello world"},
		},
	}}
	g.Expect(r.Text()).To(o.Equal("hello world"))
}

func TestToolResult_TextMultiple(t *testing.T) {
	g := o.NewWithT(t)

	r := ToolResult{mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: "hello "},
			mcp.TextContent{Type: "text", Text: "world"},
		},
	}}
	g.Expect(r.Text()).To(o.Equal("hello world"))
}

func TestToolResult_TextEmpty(t *testing.T) {
	g := o.NewWithT(t)

	r := ToolResult{mcp.CallToolResult{}}
	g.Expect(r.Text()).To(o.BeEmpty())
}

func TestToolResult_TextNonText(t *testing.T) {
	g := o.NewWithT(t)

	r := ToolResult{mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.ImageContent{Type: "image", Data: "base64data", MIMEType: "image/png"},
		},
	}}
	g.Expect(r.Text()).To(o.BeEmpty())
}
