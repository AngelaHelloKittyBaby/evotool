// Package server exposes a small MCP-compatible JSON-RPC surface.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/service"
)

type Server struct {
	Memory *service.CapabilityMemoryService
}
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}
type response struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Handle converts one MCP JSON-RPC request into one response. It supports the
// stable read-only tools first; execution remains available through the Go API.
func (s Server) Handle(ctx context.Context, input []byte) []byte {
	var req request
	if err := json.Unmarshal(input, &req); err != nil {
		return marshal(response{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: err.Error()}})
	}
	out := response{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		out.Result = map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]string{"name": "evotool", "version": "mvp"}}
	case "tools/list":
		out.Result = map[string]any{"tools": []map[string]any{{"name": "evotool.search_tools", "description": "Search reusable EvoTool capabilities", "inputSchema": map[string]any{"type": "object", "required": []string{"intent"}, "properties": map[string]any{"intent": map[string]string{"type": "string"}}}}, {"name": "evotool.get_tool", "description": "Get a stored capability", "inputSchema": map[string]any{"type": "object", "required": []string{"id"}, "properties": map[string]any{"id": map[string]string{"type": "string"}}}}}}
	case "tools/call":
		out.Result = s.call(ctx, req.Params)
	default:
		out.Error = &rpcError{Code: -32601, Message: "method not found"}
	}
	return marshal(out)
}

func (s Server) call(ctx context.Context, raw json.RawMessage) any {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return map[string]any{"isError": true, "content": []any{map[string]string{"type": "text", "text": err.Error()}}}
	}
	switch p.Name {
	case "evotool.get_tool":
		id, _ := p.Arguments["id"].(string)
		if s.Memory == nil {
			return errorContent("memory unavailable")
		}
		tool, err := s.Memory.GetTool(ctx, id)
		if err != nil {
			return errorContent(err.Error())
		}
		return textContent(tool)
	case "evotool.search_tools":
		intent, _ := p.Arguments["intent"].(string)
		if s.Memory == nil {
			return errorContent("memory unavailable")
		}
		r, err := s.Memory.SearchTools(ctx, domain.TaskSpec{Intent: intent})
		if err != nil {
			return errorContent(err.Error())
		}
		return textContent(r)
	default:
		return errorContent(fmt.Sprintf("unknown tool %q", p.Name))
	}
}
func textContent(v any) any {
	b, _ := json.Marshal(v)
	return map[string]any{"content": []any{map[string]string{"type": "text", "text": string(b)}}}
}
func errorContent(m string) any {
	return map[string]any{"isError": true, "content": []any{map[string]string{"type": "text", "text": m}}}
}
func marshal(v any) []byte { b, _ := json.Marshal(v); return b }
