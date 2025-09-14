package tool_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/gmail/v1"

	"github.com/hal9000y/gmail-mcp/internal/tool"
)

func newGetMessagesGmailSvc() *gmailSvcMock {
	return &gmailSvcMock{
		GetMessageFunc: func(_ context.Context, msgID string) (*gmail.Message, error) {
			if msgID == "error-msg" {
				return nil, fmt.Errorf("message not found: %s", msgID)
			}
			return &gmail.Message{
				Id:       msgID,
				ThreadId: "t-" + msgID,
				Snippet:  "test snippet " + msgID,
				Payload: &gmail.MessagePart{
					Headers: []*gmail.MessagePartHeader{
						{Name: "From", Value: fmt.Sprintf("Sender <%s@example.com>", msgID)},
						{Name: "To", Value: fmt.Sprintf("Receiver <receiver-%s@example.com>", msgID)},
						{Name: "Subject", Value: "Test subject " + msgID},
						{Name: "Date", Value: "2025-01-01 10:00:00"},
					},
					MimeType: "multipart/alternative",
					Parts: []*gmail.MessagePart{
						{
							MimeType: "text/plain",
							Body: &gmail.MessagePartBody{
								Data: "VGVzdCBwbGFpbiB0ZXh0IGJvZHkgZm9yIA==", // "Test plain text body for " base64
							},
						},
						{
							MimeType: "text/html",
							Body: &gmail.MessagePartBody{
								Data: "PGI+VGVzdCBIVE1MIGJvZHkgZm9yIDwvYj4=", // "<b>Test HTML body for </b>" base64
							},
						},
					},
				},
			}, nil
		},
	}
}

func TestGetMessages(t *testing.T) {
	cases := []struct {
		name        string
		req         tool.GetMessagesRequest
		expected    tool.GetMessagesResponse
		expectedErr error
	}{
		{
			name: "success with multiple messages",
			req: tool.GetMessagesRequest{
				MessageIDs: []string{"msg-001", "msg-002"},
			},
			expected: tool.GetMessagesResponse{
				Messages: []tool.MessageContent{
					{
						Summary: tool.MessageSummary{
							ID:        "msg-001",
							ThreadID:  "t-msg-001",
							Timestamp: "2025-01-01 10:00:00",
							From:      tool.EmailAddress{Name: "Sender", Email: "msg-001@example.com"},
							To:        []tool.EmailAddress{{Name: "Receiver", Email: "receiver-msg-001@example.com"}},
							Subject:   "Test subject msg-001",
							Snippet:   "test snippet msg-001",
						},
						BodyText: "Test plain text body for ",
					},
					{
						Summary: tool.MessageSummary{
							ID:        "msg-002",
							ThreadID:  "t-msg-002",
							Timestamp: "2025-01-01 10:00:00",
							From:      tool.EmailAddress{Name: "Sender", Email: "msg-002@example.com"},
							To:        []tool.EmailAddress{{Name: "Receiver", Email: "receiver-msg-002@example.com"}},
							Subject:   "Test subject msg-002",
							Snippet:   "test snippet msg-002",
						},
						BodyText: "Test plain text body for ",
					},
				},
			},
		},
		{
			name: "error case",
			req: tool.GetMessagesRequest{
				MessageIDs: []string{"error-msg"},
			},
			expectedErr: fmt.Errorf("message not found: error-msg"),
		},
	}

	gmailSvc := newGetMessagesGmailSvc()
	converter := &converterMock{
		HTML2MDFunc: func(raw []byte) (string, error) {
			return "**Converted from HTML**", nil
		},
	}

	server := tool.NewServer(gmailSvc, converter)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	defer serverSession.Close()
	require.NoError(t, err)

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	defer clientSession.Close()
	require.NoError(t, err)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name:      "get_messages",
				Arguments: tc.req,
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Content)

			if tc.expectedErr != nil {
				require.True(t, result.IsError, "Result should indicate error")
				errorText := result.Content[0].(*mcp.TextContent).Text
				assert.Contains(t, errorText, tc.expectedErr.Error())
				return
			}

			var response tool.GetMessagesResponse
			require.NoError(t,
				json.Unmarshal(
					[]byte(result.Content[0].(*mcp.TextContent).Text),
					&response,
				),
			)
			assert.Equal(t, tc.expected, response)
		})
	}
}