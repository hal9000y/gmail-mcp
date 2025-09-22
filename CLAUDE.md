# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Gmail MCP (Model Context Protocol) server that provides Gmail API access through HTTP transport. The server uses Google OAuth2 for authentication and implements MCP tools for email operations.

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
  -oauth-token-file="./data/gmail-mcp-token.json" \
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
- `-oauth-token-file` - Path to cache OAuth token (default: "./data/gmail-mcp-token.json")
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
- Token caching in `./data/gmail-mcp-token.json` (gitignored)

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
- `converter.go`: HTML to Markdown and PDF to text conversion
- `html_simplifier.go`: Simplifies HTML by unwrapping table layouts
- Uses external tools: `pandoc` for HTML→MD, `pdftotext` for PDF→Text

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
  - `pdfConverter`: `PDF2Text`

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

- Token files in `data/` directory are gitignored for security
- `.env.local` is gitignored - create from `.env` template
- Gmail API scope: `gmail.readonly` for read-only access
- External dependencies: `pandoc` and `pdftotext` for document conversion

## Code Style Guidelines

### Clean Code Principles

- **No excessive comments**: Code should be self-documenting through clear naming
  - Extract complex logic into well-named functions instead of adding comments
  - Comments should only explain "why" when the business reason isn't obvious
  - Remove comments that describe "what" the code does

- **Early returns and guard clauses**: Return errors as soon as possible
  - Use guard clauses to handle exceptional cases at the beginning of functions
  - Avoid deeply nested if-else chains
  - Fail fast principle: validate inputs early and return errors immediately

- **YAGNI (You Aren't Gonna Need It)**: Keep code minimalistic
  - Don't add functionality until it's actually needed
  - Avoid over-engineering solutions for hypothetical future requirements
  - Remove unused code, variables, and imports immediately

- **Boy Scout Rule**: Leave code cleaner than you found it
  - When touching a file, improve its overall quality
  - Fix formatting issues, remove dead code, improve naming
  - Refactor complex functions into smaller, focused ones

### SOLID Principles

- **Single Responsibility**: Each function/struct should have one reason to change
  - Functions should do one thing well
  - Break down complex functions into smaller, composable units
  - Avoid "god" functions that handle multiple concerns

- **Open/Closed**: Code should be open for extension, closed for modification
  - Use interfaces to allow new implementations without changing existing code
  - Prefer composition over inheritance

- **Dependency Inversion**: Depend on abstractions, not concretions
  - Use minimal interfaces for dependencies
  - Mock interfaces for testing, not concrete implementations

### Go-Specific Guidelines

- Use structured types for better data organization (e.g., `EmailAddress` struct)
- Embed common structs to avoid duplication (e.g., `MessageSummary` in `MessageContent`)
- Follow Go conventions for error naming (e.g., `ErrTokenNotSet`)
- Use defer for cleanup operations with proper error logging
- Prefer switch statements over if-else chains for type checking
- Use Gmail API `Fields` parameter to minimize API calls
- Return early from functions to reduce nesting
- Keep interfaces small and focused (interface segregation)

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
```

### Error Handling
- All errors are handled or explicitly ignored with `_`
- Cleanup errors (file close, remove) are logged but don't fail operations
- Error messages follow format: `functionName failed: %w`

## Refactoring Guidelines

When refactoring code in this project:

1. **Simplify complex functions**: Break down functions longer than 30 lines
2. **Extract nested logic**: Convert deeply nested conditions to early returns or separate functions
3. **Remove unnecessary state**: Eliminate state management that adds complexity without clear benefit
4. **Consolidate duplicated code**: Create shared functions for repeated patterns
5. **Improve testability**: Extract dependencies as interfaces for easier mocking

### Priority Areas for Refactoring

- Main function orchestration: Extract setup logic into focused functions
- Recursive message parsing: Simplify nested email body extraction
- Platform-specific code: Isolate OS-dependent logic in separate packages
- Configuration management: Centralize OAuth and environment config