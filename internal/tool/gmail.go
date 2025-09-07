package tool

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"

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
		Name:        "search_messages",
		Description: "Search Gmail messages using Gmail search syntax",
	}, h.SearchMessages)

	return server
}

// Common types
type EmailAddress struct {
	Name  string `json:"name,omitempty" jsonschema:"the display name"`
	Email string `json:"email" jsonschema:"the email address"`
}

// SearchMessages - Returns minimal message metadata to preserve context
type SearchMessagesRequest struct {
	Query      string `json:"query" jsonschema:"the Gmail search query"`
	MaxResults int64  `json:"max_results,omitempty" jsonschema:"max results per page"`
	PageToken  string `json:"page_token,omitempty" jsonschema:"token for pagination"`
}

type SearchMessagesResponse struct {
	Messages      []MessageSummary `json:"messages" jsonschema:"array of message summaries"`
	NextPageToken string           `json:"next_page_token,omitempty" jsonschema:"token for next page"`
	TotalResults  int              `json:"total_results" jsonschema:"number of messages returned"`
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

// GetMessages - Returns full message content with bodies converted to markdown
type GetMessagesRequest struct {
	MessageIDs []string `json:"message_ids" jsonschema:"array of message IDs to retrieve"`
}

type GetMessagesResponse struct {
	Messages []MessageContent `json:"messages" jsonschema:"array of full message contents"`
}

type MessageContent struct {
	MessageSummary
	Body        string       `json:"body" jsonschema:"message body in markdown"`
	Attachments []Attachment `json:"attachments,omitempty" jsonschema:"list of attachments"`
}

type Attachment struct {
	ID       string `json:"id" jsonschema:"attachment ID"`
	Filename string `json:"filename" jsonschema:"original filename"`
	MimeType string `json:"mime_type" jsonschema:"MIME type"`
	Size     int64  `json:"size" jsonschema:"size in bytes"`
}

// PreviewAttachments - Extracts text content from attachments when possible
type PreviewAttachmentsRequest struct {
	MessageID     string   `json:"message_id" jsonschema:"message ID containing attachments"`
	AttachmentIDs []string `json:"attachment_ids" jsonschema:"array of attachment IDs"`
}

type PreviewAttachmentsResponse struct {
	Attachments []AttachmentPreview `json:"attachments" jsonschema:"array of attachment previews"`
}

type AttachmentPreview struct {
	ID       string `json:"id" jsonschema:"attachment ID"`
	Filename string `json:"filename" jsonschema:"original filename"`
	MimeType string `json:"mime_type" jsonschema:"MIME type"`
	Content  string `json:"content,omitempty" jsonschema:"extracted text content"`
	Error    string `json:"error,omitempty" jsonschema:"error if extraction failed"`
}

func (h *GmailHandler) SearchMessages(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SearchMessagesRequest,
) (*mcp.CallToolResult, SearchMessagesResponse, error) {
	srv, err := gservice.NewGmail(ctx, h.cfg, h.tok)
	if err != nil {
		return nil, SearchMessagesResponse{}, fmt.Errorf("gservice.NewGmail failed: %w", err)
	}

	input.MaxResults = normalizeMaxResults(input.MaxResults)

	call := srv.Users.Messages.List(gmailUserID).
		Q(input.Query).
		PageToken(input.PageToken).
		MaxResults(input.MaxResults)

	result, err := call.Do()
	if err != nil {
		return nil, SearchMessagesResponse{}, fmt.Errorf("list messages failed: %w", err)
	}

	messages := make([]MessageSummary, 0, len(result.Messages))

	for _, m := range result.Messages {
		summary, err := getMessageSummary(srv, m.Id)
		if err != nil {
			return nil, SearchMessagesResponse{}, fmt.Errorf("getMessageSummary failed: %w", err)
		}

		messages = append(messages, summary)
	}

	return nil, SearchMessagesResponse{
		Messages:      messages,
		NextPageToken: result.NextPageToken,
		TotalResults:  len(messages),
	}, nil
}

func getMessageSummary(srv *gmail.Service, ID string) (MessageSummary, error) {
	msg, err := srv.Users.Messages.Get(gmailUserID, ID).
		Format("METADATA").
		MetadataHeaders("From", "To", "Cc", "Subject", "Date").
		Do()
	if err != nil {
		return MessageSummary{}, fmt.Errorf("get message summary failed: %w", err)
	}

	summary := MessageSummary{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Snippet:  msg.Snippet,
	}

	if msg.Payload != nil && msg.Payload.Headers != nil {
		extractHeadersToSummary(msg.Payload.Headers, &summary)
	}

	return summary, nil
}

func normalizeMaxResults(maxResults int64) int64 {
	if maxResults == 0 {
		return 10
	}
	if maxResults > 50 {
		return 50
	}
	return maxResults
}

func extractHeadersToSummary(headers []*gmail.MessagePartHeader, summary *MessageSummary) {
	for _, header := range headers {
		switch header.Name {
		case "From":
			summary.From = parseEmailAddress(header.Value)
		case "To":
			summary.To = parseEmailAddressList(header.Value)
		case "Cc":
			summary.CC = parseEmailAddressList(header.Value)
		case "Subject":
			summary.Subject = header.Value
		case "Date":
			summary.Timestamp = header.Value
		}
	}
}

func parseEmailAddress(from string) EmailAddress {
	addr := EmailAddress{}

	if idx := strings.Index(from, "<"); idx != -1 {
		addr.Name = strings.TrimSpace(from[:idx])
		if endIdx := strings.Index(from[idx:], ">"); endIdx != -1 {
			addr.Email = strings.TrimSpace(from[idx+1 : idx+endIdx])
		}
	} else {
		addr.Email = strings.TrimSpace(from)
	}

	addr.Name = strings.Trim(addr.Name, "\"")

	return addr
}

func parseEmailAddressList(addresses string) []EmailAddress {
	if addresses == "" {
		return nil
	}

	parts := strings.Split(addresses, ",")
	result := make([]EmailAddress, 0, len(parts))

	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, parseEmailAddress(trimmed))
		}
	}

	return result
}
