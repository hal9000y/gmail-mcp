#!/bin/bash

# Usage: ./test_integration.sh [search_query]
# Example: ./test_integration.sh "has:attachment newer_than:7d"
# Example: ./test_integration.sh "from:someone@example.com"
# Example: ./test_integration.sh "is:unread"

# Get the directory where this script is located (project root)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Path to your Gmail OAuth token file (from previous authentication)
export GMAIL_TOKEN_FILE="${GMAIL_TOKEN_FILE:-$SCRIPT_DIR/data/gmail-mcp-token.json}"

# Gmail search query from command line argument or default
if [ $# -eq 0 ]; then
    export GMAIL_SEARCH_QUERY="has:attachment newer_than:7d"
else
    export GMAIL_SEARCH_QUERY="$1"
fi

# Optional: path to .env file with OAuth credentials
export ENV_FILE="${ENV_FILE:-$SCRIPT_DIR/.env.local}"

# Make sure OAuth credentials are set (from ENV_FILE or environment)
# export OAUTH_GOOGLE_CLIENT_ID="your-client-id"
# export OAUTH_GOOGLE_CLIENT_SECRET="your-client-secret"

echo "Running Gmail MCP integration test..."
echo "Token file: $GMAIL_TOKEN_FILE"
echo "Search query: $GMAIL_SEARCH_QUERY"
echo "Env file: $ENV_FILE"
echo ""

go test -v -run TestIntegrationGmailMCP ./internal/tool
