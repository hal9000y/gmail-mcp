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

### Running with Go

```bash
go run -v ./cmd/gmail-mcp/main.go --env-file ./.env.local
```

The server will:
- Start on a random port on localhost, unless `http-addr` argument was provided
- Automatically open browser for OAuth authentication on first run
- Cache the token locally for subsequent runs
- Expose MCP endpoint at `/mcp`

### Running from Binary

Build and run:
```bash
# Build the binary
go build -o gmail-mcp ./cmd/gmail-mcp

# Create config directory and data folder
mkdir -p ~/.config/gmail-mcp/data

# Create .env.local file (remove quotes from values)
cat > ~/.config/gmail-mcp/.env.local << EOF
OAUTH_GOOGLE_CLIENT_ID=your_client_id_here
OAUTH_GOOGLE_CLIENT_SECRET=your_client_secret_here
EOF

# Run the binary
./gmail-mcp -stdio \
  -env-file ~/.config/gmail-mcp/.env.local \
  -oauth-token-file ~/.config/gmail-mcp/data/gmail-mcp-token.json \
  -log-file ~/.config/gmail-mcp/data/gmail-mcp.log
```

### Using with Claude Code

Install the gmail-mcp binary as an MCP server in Claude Code:

```bash
# Build the binary first
go build -o gmail-mcp ./cmd/gmail-mcp

# Add to Claude Code as MCP server
claude mcp add gmail-mcp -- gmail-mcp -stdio \
  -env-file ~/.config/gmail-mcp/.env.local \
  -oauth-token-file ~/.config/gmail-mcp/data/gmail-mcp-token.json \
  -log-file ~/.config/gmail-mcp/data/gmail-mcp.log
```

This command:
- Registers `gmail-mcp` as an MCP server in Claude Code
- Uses stdio transport (`-stdio`) for Claude Desktop compatibility
- Stores configuration in `~/.config/gmail-mcp/.env.local`
- Caches OAuth token in `~/.config/gmail-mcp/data/gmail-mcp-token.json`
- Logs to `~/.config/gmail-mcp/data/gmail-mcp.log` (required when using stdio)

### Running with Docker

Setup and build:
```bash
# Create config directory and data folder
mkdir -p ~/.config/gmail-mcp/data

# Create .env.local file (remove quotes from values)
cat > ~/.config/gmail-mcp/.env.local << EOF
OAUTH_GOOGLE_CLIENT_ID=your_client_id_here
OAUTH_GOOGLE_CLIENT_SECRET=your_client_secret_here
EOF

# Build the Docker image
docker build -t gmail-mcp:latest .
```

Run the container:
```bash
docker run -it --rm \
  -v ~/.config/gmail-mcp/data:/data \
  --env-file ~/.config/gmail-mcp/.env.local \
  -p 127.0.0.1:3000:3000 \
  gmail-mcp:latest
```

**IMPORTANT**: When running with Docker, you must manually open the OAuth URL in your browser:
```
http://127.0.0.1:3000/oauth?redirect=1
```

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
