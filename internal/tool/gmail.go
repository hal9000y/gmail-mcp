package tool

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/oauth2"

	"github.com/hal9000y/gmail-mcp/internal/auth"
)

const gmailUserID = "me"

type GmailHandler struct {
	cfg *oauth2.Config
	tok *auth.Token
}

func NewGmailHandler(cfg *oauth2.Config, tok *auth.Token) *GmailHandler {
	return &GmailHandler{cfg: cfg, tok: tok}
}

func NewGmailToolSet(h *GmailHandler) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "gmail-helper", Version: "v1.0.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_messages",
		Description: "Search Gmail messages using Gmail search syntax",
	}, h.SearchMessages)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_messages",
		Description: "Get full message content for specified message IDs",
	}, h.GetMessages)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "preview_attachments",
		Description: "Extract text content from attachments (PDFs, text files, etc)",
	}, h.PreviewAttachments)

	return server
}

type EmailAddress struct {
	Name  string `json:"name,omitempty" jsonschema:"the display name"`
	Email string `json:"email" jsonschema:"the email address"`
}

type MessageSummary struct {
	ID        string         `json:"id" jsonschema:"message ID"`
	ThreadID  string         `json:"thread_id" jsonschema:"thread ID"`
	Timestamp string         `json:"timestamp" jsonschema:"message timestamp"`
	From      EmailAddress   `json:"from" jsonschema:"sender information"`
	To        []EmailAddress `json:"to,omitempty" jsonschema:"recipients"`
	CC        []EmailAddress `json:"cc,omitempty" jsonschema:"CC recipients"`
	Subject   string         `json:"subject" jsonschema:"email subject"`
	Snippet   string         `json:"snippet" jsonschema:"message preview"`
}
