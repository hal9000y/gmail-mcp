package tool

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/api/gmail/v1"
)

// GetMessagesRequest contains message IDs to retrieve.
type GetMessagesRequest struct {
	MessageIDs []string `json:"message_ids" jsonschema:"array of message IDs to retrieve"`
}

// GetMessagesResponse contains full message contents.
type GetMessagesResponse struct {
	Messages []MessageContent `json:"messages" jsonschema:"array of full message contents"`
}

// MessageContent contains complete message data with body and attachments.
type MessageContent struct {
	Summary     MessageSummary `json:"summary" jsonschema:"summary"`
	BodyText    string         `json:"body_text,omitempty" jsonschema:"text body"`
	Attachments []Attachment   `json:"attachments,omitempty" jsonschema:"list of attachments"`
}

// Attachment represents email attachment metadata.
type Attachment struct {
	ID       string `json:"id" jsonschema:"attachment ID"`
	Filename string `json:"filename" jsonschema:"original filename"`
	MimeType string `json:"mime_type" jsonschema:"MIME type"`
	Size     int64  `json:"size" jsonschema:"size in bytes"`
}

type getMessagesSvc interface {
	GetMessage(ctx context.Context, msgID string) (*gmail.Message, error)
}

type htmlConverter interface {
	HTML2MD(raw []byte) (string, error)
}

// NewGetMessages creates a new GetMessages tool.
func NewGetMessages(svc getMessagesSvc, conv htmlConverter) *GetMessages {
	return &GetMessages{
		svc:  svc,
		conv: conv,
	}
}

// GetMessages retrieves full message content with converted bodies.
type GetMessages struct {
	svc  getMessagesSvc
	conv htmlConverter
}

// GetMessages retrieves complete messages by their IDs.
func (t *GetMessages) GetMessages(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetMessagesRequest,
) (*mcp.CallToolResult, GetMessagesResponse, error) {
	messages := make([]MessageContent, 0, len(input.MessageIDs))

	for _, msgID := range input.MessageIDs {
		msg, err := t.svc.GetMessage(ctx, msgID)
		if err != nil {
			return nil, GetMessagesResponse{}, fmt.Errorf("get message %s failed: %w", msgID, err)
		}

		content := MessageContent{
			Summary: extractMessageSummary(msg),
		}

		if msg.Payload != nil {
			content.Attachments = extractAttachments(msg.Payload)

			textBody, htmlBody := extractMessageBodies(msg.Payload)
			content.BodyText, err = t.previewText(textBody, htmlBody)
			if err != nil {
				return nil, GetMessagesResponse{}, fmt.Errorf("previewText failed: %w", err)
			}
		}

		messages = append(messages, content)
	}

	return nil, GetMessagesResponse{
		Messages: messages,
	}, nil
}

func (t *GetMessages) previewText(textBody, htmlBody string) (string, error) {
	if textBody != "" {
		return textBody, nil
	}
	if htmlBody == "" {
		return "", nil
	}

	converted, err := t.conv.HTML2MD([]byte(htmlBody))
	if err != nil {
		return "", fmt.Errorf("conv.HTML2MD failed: %w", err)
	}

	return converted, nil
}

func extractMessageBodies(payload *gmail.MessagePart) (textBody, htmlBody string) {
	textBody, htmlBody = extractBodyFromPart(payload)

	for _, part := range payload.Parts {
		partText, partHTML := extractBodyFromPart(part)

		if textBody == "" {
			textBody = partText
		}
		if htmlBody == "" {
			htmlBody = partHTML
		}

		if len(part.Parts) > 0 {
			nestedText, nestedHTML := extractMessageBodies(part)
			if textBody == "" {
				textBody = nestedText
			}
			if htmlBody == "" {
				htmlBody = nestedHTML
			}
		}
	}

	return textBody, htmlBody
}

func extractBodyFromPart(part *gmail.MessagePart) (textBody, htmlBody string) {
	if part.Body == nil || part.Body.Data == "" {
		return "", ""
	}

	switch part.MimeType {
	case "text/plain":
		return decodeBase64URL(part.Body.Data), ""
	case "text/html":
		return "", decodeBase64URL(part.Body.Data)
	default:
		return "", ""
	}
}

func decodeBase64URL(data string) string {
	decoded, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		decoded, err = base64.RawURLEncoding.DecodeString(data)
		if err != nil {
			return data
		}
	}
	return string(decoded)
}

func extractAttachments(payload *gmail.MessagePart) []Attachment {
	var attachments []Attachment

	if payload.Body != nil && payload.Body.AttachmentId != "" {
		attachments = append(attachments, Attachment{
			ID:       payload.Body.AttachmentId,
			Filename: payload.Filename,
			MimeType: payload.MimeType,
			Size:     payload.Body.Size,
		})
	}

	for _, part := range payload.Parts {
		if part.Body != nil && part.Body.AttachmentId != "" {
			attachments = append(attachments, Attachment{
				ID:       part.PartId,
				Filename: part.Filename,
				MimeType: part.MimeType,
				Size:     part.Body.Size,
			})
		}

		if len(part.Parts) > 0 {
			attachments = append(attachments, extractAttachments(part)...)
		}
	}

	return attachments
}
