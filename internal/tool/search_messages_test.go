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

func newSearchMessagesGmailSvc(byQuery map[string]*gmail.ListMessagesResponse) *gmailSvcMock {
	return &gmailSvcMock{
		ListMessagesFunc: func(
			_ context.Context,
			Q, _ string,
			_ int64) (*gmail.ListMessagesResponse, error) {
			res, ok := byQuery[Q]
			if !ok {
				return nil, fmt.Errorf("simulated error: %s", Q)
			}
			return res, nil
		},
		GetMessageMetadataFunc: func(_ context.Context, msgID string) (*gmail.Message, error) {
			return &gmail.Message{
				Id:       msgID,
				ThreadId: "t-" + msgID,
				Snippet:  "test summary " + msgID,
				Payload: &gmail.MessagePart{
					Headers: []*gmail.MessagePartHeader{
						{Name: "From", Value: fmt.Sprintf("Test User <test+%s@test.com>", msgID)},
						{Name: "To", Value: fmt.Sprintf("My Name <me+%s@test.com>", msgID)},
						{Name: "Subject", Value: "Super important email " + msgID},
						{Name: "Date", Value: "2025-09-14 12:12:32"},
					},
				},
			}, nil
		},
	}
}

func TestSearchMessages(t *testing.T) {
	cases := []struct {
		req         tool.SearchMessagesRequest
		expected    tool.SearchMessagesResponse
		expectedErr error
	}{
		{
			req: tool.SearchMessagesRequest{Query: "test@test.com", MaxResults: 2},
			expected: tool.SearchMessagesResponse{
				TotalResults:  2,
				NextPageToken: "next-page-token-1",
				Messages: []tool.MessageSummary{
					{
						ID:        "m-001",
						ThreadID:  "t-m-001",
						Timestamp: "2025-09-14 12:12:32",
						From:      tool.EmailAddress{Name: "Test User", Email: "test+m-001@test.com"},
						To:        []tool.EmailAddress{{Name: "My Name", Email: "me+m-001@test.com"}},
						Subject:   "Super important email m-001",
						Snippet:   "test summary m-001",
					},
					{
						ID:        "m-002",
						ThreadID:  "t-m-002",
						Timestamp: "2025-09-14 12:12:32",
						From:      tool.EmailAddress{Name: "Test User", Email: "test+m-002@test.com"},
						To:        []tool.EmailAddress{{Name: "My Name", Email: "me+m-002@test.com"}},
						Subject:   "Super important email m-002",
						Snippet:   "test summary m-002",
					},
				},
			},
		},
		{
			req:         tool.SearchMessagesRequest{Query: "undefined@undefined"},
			expectedErr: fmt.Errorf("simulated error: undefined@undefined"),
		},
	}

	gmailSvc := newSearchMessagesGmailSvc(map[string]*gmail.ListMessagesResponse{
		"test@test.com": {
			Messages: []*gmail.Message{
				{Id: "m-001"},
				{Id: "m-002"},
			},
			NextPageToken: "next-page-token-1",
		},
	})

	server := tool.NewServer(gmailSvc, &converterMock{})
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer serverSession.Close()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer clientSession.Close()

	for _, tc := range cases {
		t.Run(tc.req.Query, func(t *testing.T) {
			result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name:      "search_messages",
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

			var response tool.SearchMessagesResponse

			require.NoError(
				t,
				json.Unmarshal(
					[]byte(result.Content[0].(*mcp.TextContent).Text),
					&response,
				),
			)
			assert.Equal(t, tc.expected, response)
		})
	}
}
