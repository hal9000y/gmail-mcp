# Gmail MCP Server

A simple read-only Gmail MCP (Model Context Protocol) server that provides Gmail API access through streamable HTTP transport.

## Features

- Read-only Gmail access via MCP tools
- Google OAuth2 authentication with automatic token management
- Streamable HTTP transport for MCP protocol
- Simple, focused implementation

## Prerequisites

- Go 1.25.0 or later
- Google Cloud project with Gmail API enabled
- OAuth2 credentials (Client ID and Secret)

## Setup

1. Create a Google Cloud project and enable Gmail API
2. Create OAuth2 credentials for a web application
3. Set redirect URI to `http://localhost:8081/oauth`
4. Create `.env.local` file with your credentials:

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
- Start on `http://localhost:8081`
- Automatically open browser for OAuth authentication on first run
- Cache the token locally for subsequent runs
- Expose MCP endpoint at `/mcp`

### Available MCP Tools

- `search_emails` - Search Gmail messages with query parameters

## Architecture

- `/oauth` - Handles Google OAuth2 flow
- `/mcp` - MCP protocol endpoint (streamable HTTP)
- Token caching in `.__gmail-mcp-token.json` (auto-generated, gitignored)

## References

- [Gmail API - messages.list](https://developers.google.com/workspace/gmail/api/reference/rest/v1/users.messages/list) - Gmail search query syntax documentation
- [MCP Go SDK](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp) - Model Context Protocol SDK for Go

## License

MIT