package mcp

import (
	"context"
	"log"

	"github.com/scopweb/mcp-po-compiler-go/internal/po"
)

// Server wires MCP tool handlers to the PO service.
type Server struct {
	po *po.Service
}

// NewServer builds a Server with default dependencies.
func NewServer() *Server {
	return &Server{po: po.NewService()}
}

// Serve starts the MCP server loop (placeholder for transport integration).
func (s *Server) Serve(ctx context.Context) error {
	log.Println("MCP server initialized (transport pending)")
	<-ctx.Done()
	return ctx.Err()
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
