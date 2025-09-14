# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Gmail MCP (Model Context Protocol) server that provides Gmail API access through HTTP transport with Server-Sent Events (SSE). The server uses Google OAuth2 for authentication and implements MCP tools for email operations.

## Build and Run Commands

```bash
# Build the server
go build -o gmail-mcp ./cmd/gmail-mcp

# Run HTTP-only mode (logs to stdout, good for Docker/n8n)
go run ./cmd/gmail-mcp/main.go

# Run with stdio transport for Claude Desktop (discards logs)
go run ./cmd/gmail-mcp/main.go -stdio

# Run with stdio transport and file logging
go run ./cmd/gmail-mcp/main.go -stdio -log-file=gmail-mcp.log

# Run with custom parameters
go run ./cmd/gmail-mcp/main.go \
  -http-addr="127.0.0.1:8081" \
  -oauth-token-file=".__gmail-mcp-token.json" \
  -oauth-url="http://localhost:8081/oauth" \
  -env-file=".env.local" \
  -stdio \
  -log-file="gmail-mcp.log"

# Install dependencies
go mod download

# Update dependencies
go mod tidy
```

### CLI Flags

- `-http-addr` - HTTP server listen address (default: "localhost:0", auto-assigns port)
- `-oauth-token-file` - Path to cache OAuth token (default: ".__gmail-mcp-token.json")
- `-oauth-url` - OAuth redirect URL (default: auto-generated from http-addr)
- `-env-file` - Path to env file (default: ".env.local")
- `-stdio` - Enable stdio transport for MCP (default: false)
- `-log-file` - Log file path when stdio is enabled (default: "", discards logs)

## Required Environment Variables

Set these in `.env.local` or as environment variables:
- `OAUTH_GOOGLE_CLIENT_ID` - Google OAuth2 client ID
- `OAUTH_GOOGLE_CLIENT_SECRET` - Google OAuth2 client secret

## Architecture

### Core Components

**Main Server (`cmd/gmail-mcp/main.go`)**
- HTTP server with dual functionality: OAuth flow and MCP endpoint
- Routes: `/oauth` for Google authentication, `/mcp` for MCP protocol
- Auto-opens browser for OAuth flow on first run if token not cached
- Graceful shutdown with signal handling

**Authentication (`internal/auth/`)**
- `token.go`: OAuth2 token management with file persistence
- `http_handler.go`: HTTP handler for OAuth callback flow
- Token caching in `.__gmail-mcp-token.json` (gitignored)

**Gmail Integration (`internal/gservice/gmail.go`)**
- Facade pattern for Gmail API operations
- Implements minimal interfaces required by each tool
- Methods: `ListMessages`, `GetMessageMetadata`, `GetMessage`, `GetAttachment`
- Handles token refresh automatically

**MCP Tools (`internal/tool/`)**
- Each tool is a separate struct with dependency injection
- `search_messages.go`: SearchMessages - finds messages by query
- `get_messages.go`: GetMessages - retrieves full message content
- `preview_attachments.go`: PreviewAttachments - extracts text from attachments
- `common_model.go`: Shared types (EmailAddress, MessageSummary)
- `server.go`: MCP server setup and tool registration

**Format Converters (`internal/format/`)**
- `converter.go`: HTML to Markdown and PDF to Markdown conversion
- Uses external tools: `pandoc` for HTML→MD, `pdftohtml` for PDF→HTML

### Transport Modes

The server supports two transport modes:
- **HTTP Transport** (always enabled): Used with n8n, web clients, and OAuth flow
  - Logs to stdout (ideal for Docker/docker-compose)
  - OAuth endpoint: `/oauth`
  - MCP endpoint: `/mcp`
- **Stdio Transport** (optional via `-stdio` flag): Used with Claude Desktop
  - Logs are discarded or written to file to avoid protocol interference
  - Cannot log to stdout/stderr as it would break the MCP protocol

### Current MCP Implementation

- Uses `github.com/modelcontextprotocol/go-sdk` v0.4.0
- Dual transport support:
  - HTTP transport (always enabled): Streamable HTTP for n8n, web clients
  - Stdio transport (optional): For Claude Desktop integration
- Transport handler: `mcp.NewStreamableHTTPHandler` for HTTP
- Compatible with Claude Desktop, n8n, LangChain agents, and web-based MCP clients

## Testing Strategy

### Interface-Based Testing
- Each tool uses minimal interfaces for dependencies
- Mock implementations can be created for Gmail service and converters
- Example interfaces:
  - `searchMessagesSvc`: `ListMessages`, `GetMessageMetadata`
  - `getMessagesSvc`: `GetMessage`
  - `previewAttachmentsSvc`: `GetMessage`, `GetAttachment`
  - `htmlConverter`: `HTML2MD`
  - `pdfConverter`: `PDF2MD`

### Testing Approach
- **Unit tests implemented** using MCP in-memory transport for full protocol testing
- Mock generation via `go:generate moq` directives in `server.go`
- Test pattern:
  - Minimal interface mocking (only required methods)
  - Dynamic data generation using IDs in mock responses
  - Direct type assertion for `*mcp.TextContent` to access results
  - Error handling via `IsError` flag, not RPC failures

### Test Files
- `search_messages_test.go` - Tests message search with pagination
- `get_messages_test.go` - Tests full message retrieval with body extraction
- `preview_attachments_test.go` - Tests attachment content extraction

## Development Notes

- Token files (`.__*`) are gitignored for security
- `.env.local` is gitignored - create from `.env` template
- Gmail API scope: `gmail.readonly` for read-only access
- External dependencies: `pandoc` and `pdftohtml` for document conversion

## Code Style Guidelines

- Avoid unnecessary comments - use clear function names instead
- Comments should explain "why" not "what"
- Extract logic into helper functions with descriptive names
- Use structured types (e.g., `EmailAddress`) for better data organization
- Embed common structs to avoid duplication (e.g., `MessageSummary` in `MessageContent`)
- Use Gmail API `Fields` parameter to minimize API calls
- Follow Go conventions for error naming (e.g., `ErrTokenNotSet`)
- Use defer for cleanup operations with error logging
- Prefer switch statements over if-else chains for type checking

## Code Quality

### Linting
The project passes all default `golangci-lint` checks including:
- `revive`: Go code style checker
- `errcheck`: Ensures all errors are handled
- `staticcheck`: Go static analysis
- `cyclop`: Cyclomatic complexity checker

Run linters:
```bash
golangci-lint run
```

### Testing
Run all tests:
```bash
go test ./internal/tool -v
```

Generate mocks (requires moq):
```bash
go generate ./...

### Error Handling
- All errors are handled or explicitly ignored with `_`
- Cleanup errors (file close, remove) are logged but don't fail operations
- Error messages follow format: `functionName failed: %w`