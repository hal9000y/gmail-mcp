package tool

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/oauth2"

	"github.com/hal9000y/gmail-mcp/internal/auth"
	"github.com/hal9000y/gmail-mcp/internal/gservice"
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
		Name:        "search_emails",
		Description: "Search emails according to specified criteria",
	}, h.SearchEmail)

	return server
}

type SearchEmailRequest struct {
	Query      string
	PageToken  string
	MaxResults int64
}

type SearchEmailResponse struct {
	Messages      []EmailHeader
	NextPageToken string
}

type EmailAddress struct {
	Name  string
	Email string
}

type EmailHeader struct {
	ID        string
	ThreadID  string
	Timestamp string
	Snippet   string
	From      EmailAddress
	To        []EmailAddress
	Subject   string
}

func (h *GmailHandler) SearchEmail(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SearchEmailRequest,
) (*mcp.CallToolResult, SearchEmailResponse, error) {
	srv, err := gservice.NewGmail(ctx, h.cfg, h.tok)
	if err != nil {
		return nil, SearchEmailResponse{}, fmt.Errorf("gservice.NewGmail failed: %w", err)
	}

	call := srv.Users.Messages.List(gmailUserID).
		Q(input.Query).
		PageToken(input.PageToken).
		MaxResults(input.MaxResults)

	result, err := call.Do()
	if err != nil {
		return nil, SearchEmailResponse{}, fmt.Errorf("call.Do failed: %w", err)
	}

	messages := make([]EmailHeader, 0, len(result.Messages))

	for _, m := range result.Messages {
		messages = append(messages, EmailHeader{
			ID:       m.Id,
			ThreadID: m.ThreadId,
			Snippet:  m.Snippet,
		})
	}

	return nil, SearchEmailResponse{
		Messages:      messages,
		NextPageToken: result.NextPageToken,
	}, nil
}
