package tool

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/api/gmail/v1"

	"github.com/hal9000y/gmail-mcp/internal/gservice"
)

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
		msg, err := srv.Users.Messages.Get(gmailUserID, m.Id).
			Format("METADATA").
			MetadataHeaders("From", "To", "Cc", "Subject", "Date").
			Do()
		if err != nil {
			return nil, SearchMessagesResponse{}, fmt.Errorf("get message %s failed: %w", m.Id, err)
		}

		summary := extractMessageSummary(msg)
		messages = append(messages, summary)
	}

	return nil, SearchMessagesResponse{
		Messages:      messages,
		NextPageToken: result.NextPageToken,
		TotalResults:  len(messages),
	}, nil
}

func extractMessageSummary(msg *gmail.Message) MessageSummary {
	summary := MessageSummary{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Snippet:  msg.Snippet,
	}

	if msg.Payload != nil && msg.Payload.Headers != nil {
		extractHeadersToSummary(msg.Payload.Headers, &summary)
	}

	return summary
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
