# Gmail MCP Server

A simple read-only Gmail MCP (Model Context Protocol) server that provides Gmail API access through streamable HTTP transport.

## Features

- Read-only Gmail access via MCP tools
- Google OAuth2 authentication with automatic token management
- Streamable HTTP transport for MCP protocol
- Simple, focused implementation

## Prerequisites

- Go 1.21+
- Google Cloud project with Gmail API enabled
- OAuth2 credentials (Client ID and Secret)
- Document converters (for attachments):
  - `pandoc` - HTML to Markdown conversion
  - `pdftohtml` - PDF text extraction

## Setup

1. Create a Google Cloud project and enable Gmail API
2. Create OAuth2 credentials for a desktop application
3. Create `.env.local` file with your credentials:

```bash
OAUTH_GOOGLE_CLIENT_ID=your_client_id_here
OAUTH_GOOGLE_CLIENT_SECRET=your_client_secret_here
```

## Usage

Run the server:

```bash
go run ./cmd/gmail-mcp/main.go
```

The server will:
- Start on a random port on localhost, unless `http-addr` argument was provided
- Automatically open browser for OAuth authentication on first run
- Cache the token locally for subsequent runs
- Expose MCP endpoint at `/mcp`

### Available MCP Tools

- `search_messages` - Search Gmail messages using Gmail search syntax
- `get_messages` - Retrieve full message content with bodies converted to Markdown
- `preview_attachments` - Extract text content from email attachments (text, PDF)

## Architecture

- `/oauth` - Handles Google OAuth2 flow
- `/mcp` - MCP protocol endpoint (streamable HTTP)
- Token caching in `.__gmail-mcp-token.json` (auto-generated, gitignored)
- Dual transport support: HTTP (default) and stdio (for Claude Desktop)

## Development

### Running Tests
```bash
# Run all tests
go test ./internal/tool -v

# Run specific test
go test ./internal/tool -run TestSearchMessages -v
```

### Code Quality
```bash
# Run linters
golangci-lint run

# Generate mocks for testing
go generate ./...
```

## References

- [Gmail API - quickstart](https://developers.google.com/workspace/gmail/api/quickstart/go)
- [Gmail API - messages.list](https://developers.google.com/workspace/gmail/api/reference/rest/v1/users.messages/list) - Gmail search query syntax documentation
- [MCP Go SDK](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp) - Model Context Protocol SDK for Go
- [MCP server inspector](https://github.com/modelcontextprotocol/inspector)

## License

MIT
