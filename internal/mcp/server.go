package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/scopweb/mcp-po-compiler-go/internal/po"
)

// JSON-RPC 2.0 structures
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      any         `json:"id,omitempty"`
	Result  any         `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCP protocol structures
type initializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

type initializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

type toolsListResult struct {
	Tools []toolDefinition `json:"tools"`
}

type toolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type callToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type callToolResult struct {
	Content []contentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Server wires MCP tool handlers to the PO service.
type Server struct {
	po     *po.Service
	mu     sync.Mutex
	writer io.Writer
}

// NewServer builds a Server with default dependencies.
func NewServer() *Server {
	return &Server{
		po:     po.NewService(),
		writer: os.Stdout,
	}
}

// Serve starts the MCP server loop over stdio.
func (s *Server) Serve(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read error: %w", err)
		}

		if len(line) == 0 || (len(line) == 1 && line[0] == '\n') {
			continue
		}

		var req jsonRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, -32700, "Parse error")
			continue
		}

		s.handleRequest(ctx, &req)
	}
}

func (s *Server) handleRequest(ctx context.Context, req *jsonRPCRequest) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "initialized":
		// Notification, no response needed
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(ctx, req)
	case "ping":
		s.sendResult(req.ID, map[string]any{})
	default:
		s.sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (s *Server) handleInitialize(req *jsonRPCRequest) {
	result := initializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: map[string]any{
			"tools": map[string]any{},
		},
	}
	result.ServerInfo.Name = "mcp-po-compiler"
	result.ServerInfo.Version = "1.0.2"
	s.sendResult(req.ID, result)
}

func (s *Server) handleToolsList(req *jsonRPCRequest) {
	tools := []toolDefinition{
		{
			Name:        "compile_po",
			Description: "Compile a PO file content to MO binary format",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"po_content": map[string]any{
						"type":        "string",
						"description": "The content of the PO file to compile",
					},
					"return": map[string]any{
						"type":        "string",
						"enum":        []string{"base64", "path"},
						"default":     "base64",
						"description": "Return format: base64-encoded MO data or path to temp file",
					},
				},
				"required": []string{"po_content"},
			},
		},
		{
			Name:        "validate_po",
			Description: "Validate a PO file content and report warnings",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"po_content": map[string]any{
						"type":        "string",
						"description": "The content of the PO file to validate",
					},
				},
				"required": []string{"po_content"},
			},
		},
		{
			Name:        "summarize_po",
			Description: "Summarize translation progress of a PO file",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"po_content": map[string]any{
						"type":        "string",
						"description": "The content of the PO file to summarize",
					},
				},
				"required": []string{"po_content"},
			},
		},
	}
	s.sendResult(req.ID, toolsListResult{Tools: tools})
}

func (s *Server) handleToolsCall(ctx context.Context, req *jsonRPCRequest) {
	var params callToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}

	var resultText string
	var isError bool

	switch params.Name {
	case "compile_po":
		poContent, _ := params.Arguments["po_content"].(string)
		returnMode, _ := params.Arguments["return"].(string)
		if returnMode == "" {
			returnMode = "base64"
		}
		result, err := s.po.Compile(ctx, poContent, returnMode)
		if err != nil {
			resultText = fmt.Sprintf("Error: %v", err)
			isError = true
		} else {
			jsonBytes, _ := json.Marshal(result)
			resultText = string(jsonBytes)
		}

	case "validate_po":
		poContent, _ := params.Arguments["po_content"].(string)
		warnings, summary, err := s.po.Validate(ctx, poContent)
		if err != nil {
			resultText = fmt.Sprintf("Error: %v", err)
			isError = true
		} else {
			jsonBytes, _ := json.Marshal(map[string]any{
				"warnings": warnings,
				"summary":  summary,
			})
			resultText = string(jsonBytes)
		}

	case "summarize_po":
		poContent, _ := params.Arguments["po_content"].(string)
		summary, err := s.po.Summarize(ctx, poContent)
		if err != nil {
			resultText = fmt.Sprintf("Error: %v", err)
			isError = true
		} else {
			jsonBytes, _ := json.Marshal(summary)
			resultText = string(jsonBytes)
		}

	default:
		resultText = fmt.Sprintf("Unknown tool: %s", params.Name)
		isError = true
	}

	s.sendResult(req.ID, callToolResult{
		Content: []contentBlock{{Type: "text", Text: resultText}},
		IsError: isError,
	})
}

func (s *Server) sendResult(id any, result any) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.send(resp)
}

func (s *Server) sendError(id any, code int, message string) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message},
	}
	s.send(resp)
}

func (s *Server) send(resp jsonRPCResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, _ := json.Marshal(resp)
	fmt.Fprintf(s.writer, "%s\n", data)
}

// CompilePO dispatches the compile_po tool.
func (s *Server) CompilePO(ctx context.Context, poContent, returnMode string) (*po.CompileResult, error) {
	return s.po.Compile(ctx, poContent, returnMode)
}

// ValidatePO dispatches the validate_po tool.
func (s *Server) ValidatePO(ctx context.Context, poContent string) ([]string, po.Summary, error) {
	return s.po.Validate(ctx, poContent)
}

// SummarizePO dispatches the summarize_po tool.
func (s *Server) SummarizePO(ctx context.Context, poContent string) (po.Summary, error) {
	return s.po.Summarize(ctx, poContent)
}
