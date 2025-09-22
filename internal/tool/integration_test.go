package tool_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"

	"github.com/hal9000y/gmail-mcp/internal/auth"
	"github.com/hal9000y/gmail-mcp/internal/format"
	"github.com/hal9000y/gmail-mcp/internal/gservice"
	"github.com/hal9000y/gmail-mcp/internal/tool"
)

func TestIntegrationGmailMCP(t *testing.T) {
	tokenFile := os.Getenv("GMAIL_TOKEN_FILE")
	searchQuery := os.Getenv("GMAIL_SEARCH_QUERY")
	envFile := os.Getenv("ENV_FILE")

	if tokenFile == "" || searchQuery == "" {
		t.Skip("Skipping integration test: GMAIL_TOKEN_FILE and GMAIL_SEARCH_QUERY env vars must be set")
	}

	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			t.Logf("Warning: could not load env file %s: %v", envFile, err)
		}
	}

	clientID := os.Getenv("OAUTH_GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		t.Skip("Skipping integration test: OAUTH_GOOGLE_CLIENT_ID and OAUTH_GOOGLE_CLIENT_SECRET must be set")
	}

	session := setupMCPSession(t, clientID, clientSecret, tokenFile)
	defer session.Close()

	messages := searchMessages(session.ctx, t, session.client, searchQuery)
	t.Logf("Found %d messages", len(messages))

	totalTokens := 0
	messageStats := make([]messageAnalysis, 0, len(messages))

	for i, msg := range messages {
		t.Logf("\n=== MESSAGE %d/%d ===", i+1, len(messages))
		stats := analyzeMessage(session.ctx, t, session.client, msg)
		messageStats = append(messageStats, stats)
		totalTokens += stats.EstimatedTokens + stats.AttachmentTokens
		printMessageStats(t, stats)
	}

	printSummary(t, messageStats, totalTokens)
}

type mcpSession struct {
	ctx    context.Context
	client *mcp.ClientSession
	server *mcp.ServerSession
}

func (s *mcpSession) Close() {
	s.client.Close()
	s.server.Close()
}

func setupMCPSession(t *testing.T, clientID, clientSecret, tokenFile string) *mcpSession {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:8080/oauth",
		Scopes:       []string{gmail.GmailReadonlyScope},
	}

	tok, err := auth.NewToken(config, tokenFile)
	require.NoError(t, err, "Failed to create token")

	_, err = tok.OAuthToken()
	require.NoError(t, err, "Token not set - please authenticate first")

	gmailSvc := gservice.NewGmail(config, tok)
	converter := &format.Converter{}
	server := tool.NewServer(gmailSvc, converter)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)

	return &mcpSession{
		ctx:    ctx,
		client: clientSession,
		server: serverSession,
	}
}

func searchMessages(ctx context.Context, t *testing.T, client *mcp.ClientSession, query string) []tool.MessageSummary {
	t.Logf("\n=== SEARCHING MESSAGES ===")
	t.Logf("Query: %s", query)

	result, err := client.CallTool(ctx, &mcp.CallToolParams{
		Name: "search_messages",
		Arguments: tool.SearchMessagesRequest{
			Query:      query,
			MaxResults: 10,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.IsError, "Search failed: %v", result.Content)

	var response tool.SearchMessagesResponse
	require.NoError(t, json.Unmarshal(
		[]byte(result.Content[0].(*mcp.TextContent).Text),
		&response,
	))

	return response.Messages
}

func analyzeMessage(ctx context.Context, t *testing.T, client *mcp.ClientSession, msg tool.MessageSummary) messageAnalysis {
	stats := messageAnalysis{
		ID:      msg.ID,
		From:    formatEmails([]tool.EmailAddress{msg.From}),
		To:      formatEmails(msg.To),
		Subject: msg.Subject,
		Date:    msg.Timestamp,
	}

	fullMsg := getFullMessage(ctx, t, client, msg.ID)
	if fullMsg == nil {
		return stats
	}

	stats.BodyPreview = truncateString(fullMsg.BodyText, 200)
	stats.BodySize = len(fullMsg.BodyText)
	stats.EstimatedTokens = estimateTokens(fullMsg.BodyText)

	if len(fullMsg.Attachments) > 0 {
		analyzeAttachments(ctx, t, client, msg.ID, fullMsg.Attachments, &stats)
	}

	return stats
}

func getFullMessage(ctx context.Context, t *testing.T, client *mcp.ClientSession, messageID string) *tool.MessageContent {
	result, err := client.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_messages",
		Arguments: tool.GetMessagesRequest{
			MessageIDs: []string{messageID},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.IsError, "Get message failed: %v", result.Content)

	var response tool.GetMessagesResponse
	require.NoError(t, json.Unmarshal(
		[]byte(result.Content[0].(*mcp.TextContent).Text),
		&response,
	))

	if len(response.Messages) > 0 {
		return &response.Messages[0]
	}
	return nil
}

func analyzeAttachments(ctx context.Context, t *testing.T, client *mcp.ClientSession, messageID string, attachments []tool.Attachment, stats *messageAnalysis) {
	stats.AttachmentCount = len(attachments)

	attachmentIDs := make([]string, 0, len(attachments))
	for _, att := range attachments {
		attachmentIDs = append(attachmentIDs, att.ID)
		stats.AttachmentNames = append(stats.AttachmentNames, att.Filename)
	}

	previews := getAttachmentPreviews(ctx, t, client, messageID, attachmentIDs)
	for _, preview := range previews {
		if preview.Error == "" && preview.Content != "" {
			attTokens := estimateTokens(preview.Content)
			stats.AttachmentTokens += attTokens
			stats.AttachmentPreviews = append(stats.AttachmentPreviews, attachmentPreview{
				Filename: preview.Filename,
				MimeType: preview.MimeType,
				Size:     len(preview.Content),
				Tokens:   attTokens,
				Preview:  truncateString(preview.Content, 100),
			})
		}
	}
}

func getAttachmentPreviews(ctx context.Context, _ *testing.T, client *mcp.ClientSession, messageID string, attachmentIDs []string) []tool.AttachmentPreview {
	result, err := client.CallTool(ctx, &mcp.CallToolParams{
		Name: "preview_attachments",
		Arguments: tool.PreviewAttachmentsRequest{
			MessageID:     messageID,
			AttachmentIDs: attachmentIDs,
		},
	})

	if err != nil || result == nil || result.IsError {
		return nil
	}

	var response tool.PreviewAttachmentsResponse
	if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &response); err != nil {
		return nil
	}

	return response.Attachments
}

func printSummary(t *testing.T, messageStats []messageAnalysis, totalTokens int) {
	t.Logf("\n=== SUMMARY ===")
	t.Logf("Total messages analyzed: %d", len(messageStats))
	t.Logf("Total estimated tokens: %d", totalTokens)
	t.Logf("Average tokens per message: %d", totalTokens/maxInt(len(messageStats), 1))

	t.Logf("\n=== TOKEN DISTRIBUTION ===")
	for _, stats := range messageStats {
		t.Logf("Message %s: %d tokens (body: %d, attachments: %d)",
			truncateString(stats.Subject, 30),
			stats.EstimatedTokens+stats.AttachmentTokens,
			stats.EstimatedTokens,
			stats.AttachmentTokens,
		)
	}
}

type messageAnalysis struct {
	ID                 string
	From               string
	To                 string
	Subject            string
	Date               string
	BodyPreview        string
	BodySize           int
	EstimatedTokens    int
	AttachmentCount    int
	AttachmentNames    []string
	AttachmentTokens   int
	AttachmentPreviews []attachmentPreview
}

type attachmentPreview struct {
	Filename string
	MimeType string
	Size     int
	Tokens   int
	Preview  string
}

func printMessageStats(t *testing.T, stats messageAnalysis) {
	t.Logf("ID: %s", stats.ID)
	t.Logf("From: %s", stats.From)
	t.Logf("To: %s", stats.To)
	t.Logf("Subject: %s", stats.Subject)
	t.Logf("Date: %s", stats.Date)
	t.Logf("Body: %d bytes (~%d tokens)", stats.BodySize, stats.EstimatedTokens)
	t.Logf("Body preview: %s", stats.BodyPreview)

	if stats.AttachmentCount > 0 {
		t.Logf("Attachments: %d", stats.AttachmentCount)
		for i, name := range stats.AttachmentNames {
			t.Logf("  - %s", name)
			if i < len(stats.AttachmentPreviews) {
				preview := stats.AttachmentPreviews[i]
				t.Logf("    Type: %s, Size: %d bytes (~%d tokens)",
					preview.MimeType, preview.Size, preview.Tokens)
				t.Logf("    Preview: %s", preview.Preview)
			}
		}
		t.Logf("Total attachment tokens: %d", stats.AttachmentTokens)
	}
	t.Logf("Total tokens for message: %d", stats.EstimatedTokens+stats.AttachmentTokens)
}

func formatEmails(emails []tool.EmailAddress) string {
	if len(emails) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(emails))
	for _, email := range emails {
		if email.Name != "" {
			parts = append(parts, fmt.Sprintf("%s <%s>", email.Name, email.Email))
		} else {
			parts = append(parts, email.Email)
		}
	}
	return strings.Join(parts, ", ")
}

func truncateString(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func estimateTokens(text string) int {
	return len(text) / 4
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
