# mcp-po-compiler-go

MCP server written in Go that compiles gettext `.po` catalogs into `.mo` binaries (Poedit/msgfmt compatible), validates catalogs, and summarizes translation progress. Designed to run locally with Claude Desktop (or any MCP-aware client) without installing Poedit.

## Features
- Compile `.po` → `.mo` deterministically (little-endian, hash-stable ordering).
- Validate required headers and untranslated entries.
- Summarize language and progress counts.
- Single static binary (CGO disabled) with no external tools.

## Repository layout
- [cmd/mcp-po-server/main.go](cmd/mcp-po-server/main.go) — CLI entrypoint to run the MCP server.
- [internal/mcp/server.go](internal/mcp/server.go) — MCP wiring and tool handlers.
- [internal/po/service.go](internal/po/service.go) — PO parsing, validation, MO writer.
- [manifest.json](manifest.json) — MCP manifest declaring tools and schemas.
- [internal/po/service_test.go](internal/po/service_test.go) — integration tests for compile/validate.

## Build
```bash
# from repo root
go mod tidy
CGO_ENABLED=0 go build -o bin/mcp-po-server ./cmd/mcp-po-server
```

## Test
```bash
go test ./...
```

## Running the MCP server (standalone)
```bash
./bin/mcp-po-server
```
The server currently logs initialization and waits; Claude Desktop (or another MCP client) will connect to it.

## MCP tools exposed
- `compile_po`
  - Input: `po_content` (string, UTF-8). Optional `return` enum: `base64` (default) or `path`.
  - Output: base64-encoded `.mo` or path to a temp `.mo`, plus stats.
- `validate_po`
  - Input: `po_content` (string).
  - Output: list of warnings (missing headers, untranslated entries) and stats.
- `summarize_po`
  - Input: `po_content` (string).
  - Output: summary with language and counts.

## Configure in Claude Desktop
1) Build the server (see Build section) or place the binary somewhere on your machine.
2) Add a new MCP server entry in `claude_desktop_config.json` (or the UI, if available). Example:
```json
{
  "mcpServers": {
    "po-compiler": {
      "command": "/absolute/path/to/bin/mcp-po-server",
      "args": []
    }
  }
}
```
3) Restart Claude Desktop. The tools `compile_po`, `validate_po`, and `summarize_po` should appear and can be invoked by the assistant.

## Security and limits
- Rejects empty PO input; enforces deterministic output ordering.
- No filesystem access beyond temp file when `return=path`.
- Consider wrapping the process with OS-level limits (ulimit/container) for very large files.

## Notes
- Generated `.mo` should match msgfmt/Poedit for common WordPress catalogs; add your own fixtures under `testdata/` to compare hashes if needed.
- Module path: `github.com/scopweb/mcp-po-compiler-go`.
