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

func newPreviewAttachmentsGmailSvc() *gmailSvcMock {
	return &gmailSvcMock{
		GetMessageFunc: func(_ context.Context, msgID string) (*gmail.Message, error) {
			if msgID == "error-msg" {
				return nil, fmt.Errorf("message not found: %s", msgID)
			}
			return &gmail.Message{
				Id: msgID,
				Payload: &gmail.MessagePart{
					Parts: []*gmail.MessagePart{
						{
							PartId:   "1",
							Filename: "document.txt",
							MimeType: "text/plain",
							Body: &gmail.MessagePartBody{
								AttachmentId: "attach-txt-" + msgID,
								Size:         100,
							},
						},
						{
							PartId:   "2",
							Filename: "report.pdf",
							MimeType: "application/pdf",
							Body: &gmail.MessagePartBody{
								AttachmentId: "attach-pdf-" + msgID,
								Size:         200,
							},
						},
					},
				},
			}, nil
		},
		GetAttachmentFunc: func(_ context.Context, msgID, attachmentID string) (*gmail.MessagePartBody, error) {
			switch attachmentID {
			case "attach-txt-" + msgID:
				// "Text content for " + msgID in base64
				return &gmail.MessagePartBody{
					Data: "VGV4dCBjb250ZW50IGZvciA=",
				}, nil
			case "attach-pdf-" + msgID:
				// Simulate PDF binary data (just placeholder)
				return &gmail.MessagePartBody{
					Data: "UERGIGNvbnRlbnQgZm9yIA==",
				}, nil
			default:
				return nil, fmt.Errorf("attachment not found: %s", attachmentID)
			}
		},
	}
}

func TestPreviewAttachments(t *testing.T) {
	cases := []struct {
		name        string
		req         tool.PreviewAttachmentsRequest
		expected    tool.PreviewAttachmentsResponse
		expectedErr error
	}{
		{
			name: "success with text and pdf attachments",
			req: tool.PreviewAttachmentsRequest{
				MessageID:     "msg-001",
				AttachmentIDs: []string{"1", "2"},
			},
			expected: tool.PreviewAttachmentsResponse{
				Attachments: []tool.AttachmentPreview{
					{
						ID:       "1",
						Filename: "document.txt",
						MimeType: "text/plain",
						Content:  "Text content for ",
					},
					{
						ID:       "2",
						Filename: "report.pdf",
						MimeType: "application/pdf",
						Content:  "PDF content as plain text",
					},
				},
			},
		},
		{
			name: "error case - message not found",
			req: tool.PreviewAttachmentsRequest{
				MessageID:     "error-msg",
				AttachmentIDs: []string{"1"},
			},
			expectedErr: fmt.Errorf("message not found: error-msg"),
		},
	}

	gmailSvc := newPreviewAttachmentsGmailSvc()
	converter := &converterMock{
		PDF2TextFunc: func(_ []byte) (string, error) {
			return "PDF content as plain text", nil
		},
	}

	server := tool.NewServer(gmailSvc, converter)
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
		t.Run(tc.name, func(t *testing.T) {
			result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name:      "preview_attachments",
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

			var response tool.PreviewAttachmentsResponse
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
