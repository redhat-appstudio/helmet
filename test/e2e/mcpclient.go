package e2e

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
)

// MCPClient communicates with a helmet-ex mcp-server subprocess via
// JSON-RPC 2.0 over STDIO. Created by Runner.StartMCPServer.
type MCPClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader
	nextID int64
	mu     sync.Mutex
}

// notify sends a JSON-RPC 2.0 notification (no id, no response expected).
func (c *MCPClient) notify(method string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	n := jsonRPCNotification{JSONRPC: "2.0", Method: method}
	data, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON-RPC notification: %w", err)
	}
	if _, err := fmt.Fprintf(c.stdin, "%s\n", data); err != nil {
		return fmt.Errorf("failed to write notification to MCP server stdin: %w", err)
	}
	return nil
}

// send marshals and writes a JSON-RPC request, then reads the response.
// The mu mutex serializes concurrent calls.
func (c *MCPClient) send(
	_ context.Context,
	method string,
	params any,
) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.nextID
	c.nextID++

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON-RPC request: %w", err)
	}

	// Write JSON + newline to stdin.
	if _, err := fmt.Fprintf(c.stdin, "%s\n", data); err != nil {
		return nil, fmt.Errorf("failed to write to MCP server stdin: %w", err)
	}

	// Read one line from stdout.
	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read from MCP server stdout: %w", err)
	}

	var resp jsonRPCResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal JSON-RPC response: %w\nraw: %s", err, line)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf(
			"JSON-RPC error (code %d): %s", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result, nil
}

// Initialize performs the MCP initialize handshake.
// Must be called before any tool calls.
func (c *MCPClient) Initialize(ctx context.Context) error {
	_, err := c.send(ctx, "initialize", initializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo: clientInfo{
			Name:    "helmet-e2e-test",
			Version: "1.0.0",
		},
	})
	if err != nil {
		return fmt.Errorf("MCP initialize handshake failed: %w", err)
	}

	// Send initialized notification (fire-and-forget, no id, no response).
	if err := c.notify("notifications/initialized"); err != nil {
		return fmt.Errorf("MCP initialized notification failed: %w", err)
	}

	return nil
}

// ListTools calls tools/list and returns the tool names.
func (c *MCPClient) ListTools(ctx context.Context) ([]string, error) {
	raw, err := c.send(ctx, "tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("tools/list failed: %w", err)
	}

	var result mcp.ListToolsResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ListToolsResult: %w", err)
	}

	names := make([]string, len(result.Tools))
	for i, t := range result.Tools {
		names[i] = t.Name
	}
	return names, nil
}

// CallTool invokes a tool by name with optional arguments.
// Tool errors arrive as ToolResult with IsError=true, not as Go errors.
// Go errors indicate protocol-level failures only; they are reported via
// Gomega Expect to fail the test immediately.
func (c *MCPClient) CallTool(
	ctx context.Context,
	name string,
	args map[string]any,
) ToolResult {
	raw, err := c.send(ctx, "tools/call", callToolParams{
		Name: name, Arguments: args,
	})
	if err != nil {
		// Protocol error — fail the test immediately via panic so callers
		// don't need to check error returns on every call.
		panic(fmt.Sprintf(
			"JSON-RPC protocol error calling tool %q: %v", name, err))
	}

	var result mcp.CallToolResult
	if err := json.Unmarshal(raw, &result); err != nil {
		panic(fmt.Sprintf(
			"failed to unmarshal CallToolResult for tool %q: %v", name, err))
	}

	return ToolResult{result}
}

// Shutdown sends a clean shutdown and waits for the subprocess to exit.
func (c *MCPClient) Shutdown() error {
	c.stdin.Close()
	return c.cmd.Wait()
}

// NewMCPClient instantiates an MCPClient.
func NewMCPClient(
	cmd *exec.Cmd,
	stdin io.WriteCloser,
	reader *bufio.Reader,
	nextID int64,
) *MCPClient {
	return &MCPClient{cmd: cmd, stdin: stdin, reader: reader, nextID: nextID}
}
