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

**Gmail Integration (`internal/gservice/service.go`)**
- Creates Gmail API client using OAuth2 token
- Handles token refresh automatically

**MCP Tools (`internal/tool/`)**
- `gmail.go`: Implements Gmail-related MCP tools
- Tool registration with MCP server
- Structured request/response types with JSON and jsonschema tags
- Tools being implemented:
  - `search_messages` - Returns minimal metadata (ID, thread, date, from, subject, snippet)
  - `get_messages` - Returns full message with markdown-converted body
  - `preview_attachments` - Extracts text content from attachments

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

## Development Notes

- Token files (`.__*`) are gitignored for security
- `.env.local` is gitignored - create from `.env` template
- No tests currently exist - implement as needed
- Gmail API scope: `gmail.readonly` for read-only access

## Code Style Guidelines

- Avoid unnecessary comments - use clear function names instead
- Comments should explain "why" not "what"
- Extract logic into helper functions with descriptive names
- Use structured types (e.g., `EmailAddress`) for better data organization
- Embed common structs to avoid duplication (e.g., `MessageSummary` in `MessageContent`)
- Use Gmail API `Fields` parameter to minimize API calls