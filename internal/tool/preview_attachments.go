package tool

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/api/gmail/v1"

	"github.com/hal9000y/gmail-mcp/internal/gservice"
)

type PreviewAttachmentsRequest struct {
	MessageID     string   `json:"message_id" jsonschema:"message ID containing attachments"`
	AttachmentIDs []string `json:"attachment_ids" jsonschema:"array of attachment IDs (Part IDs)"`
}

type PreviewAttachmentsResponse struct {
	Attachments []AttachmentPreview `json:"attachments" jsonschema:"array of attachment previews"`
}

type AttachmentPreview struct {
	ID       string `json:"id" jsonschema:"attachment ID (Part ID)"`
	Filename string `json:"filename" jsonschema:"original filename"`
	MimeType string `json:"mime_type" jsonschema:"MIME type"`
	Content  string `json:"content,omitempty" jsonschema:"extracted text content"`
	Error    string `json:"error,omitempty" jsonschema:"error if extraction failed"`
}

func (h *GmailHandler) PreviewAttachments(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input PreviewAttachmentsRequest,
) (*mcp.CallToolResult, PreviewAttachmentsResponse, error) {
	srv, err := gservice.NewGmail(ctx, h.cfg, h.tok)
	if err != nil {
		return nil, PreviewAttachmentsResponse{}, fmt.Errorf("gservice.NewGmail failed: %w", err)
	}

	msg, err := srv.Users.Messages.Get(gmailUserID, input.MessageID).Do()
	if err != nil {
		return nil, PreviewAttachmentsResponse{}, fmt.Errorf("get message failed: %w", err)
	}

	previews := make([]AttachmentPreview, 0, len(input.AttachmentIDs))

	for _, partID := range input.AttachmentIDs {
		content := findAttachmentMetadata(msg.Payload, partID)

		if content.Body == nil || content.Body.AttachmentId == "" {
			return nil, PreviewAttachmentsResponse{}, fmt.Errorf("No attachmentID found for %s/%s", input.MessageID, partID)
		}
		attachID := content.Body.AttachmentId
		fileName := content.Filename
		mimeType := content.MimeType

		attachment, err := srv.Users.Messages.Attachments.Get(gmailUserID, input.MessageID, attachID).Do()
		if err != nil {
			return nil, PreviewAttachmentsResponse{}, fmt.Errorf("get attachment %s failed: %w", attachID, err)
		}

		preview := AttachmentPreview{
			ID:       partID,
			Filename: fileName,
			MimeType: mimeType,
		}

		data, err := h.extractAttachmentContent(attachment.Data, preview.MimeType, preview.Filename)
		if err != nil {
			preview.Error = err.Error()
		} else {
			preview.Content = data
		}

		previews = append(previews, preview)
	}

	return nil, PreviewAttachmentsResponse{
		Attachments: previews,
	}, nil
}

func findAttachmentMetadata(payload *gmail.MessagePart, partID string) *gmail.MessagePart {
	if payload.Body != nil && payload.PartId == partID {
		return payload
	}

	for _, part := range payload.Parts {
		if found := findAttachmentMetadata(part, partID); found != nil {
			return found
		}
	}

	return nil
}

func (h *GmailHandler) extractAttachmentContent(data, mimeType, filename string) (string, error) {
	decodedData, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		decodedData, err = base64.RawURLEncoding.DecodeString(data)
		if err != nil {
			return "", fmt.Errorf("failed to decode attachment: %w", err)
		}
	}

	switch {
	case strings.HasPrefix(mimeType, "text/"):
		return string(decodedData), nil

	case mimeType == "application/pdf":
		return h.conv.PDF2MD(decodedData)

	case strings.HasSuffix(filename, ".txt") || strings.HasSuffix(filename, ".md"):
		return string(decodedData), nil

	case strings.HasSuffix(filename, ".csv"):
		return string(decodedData), nil

	default:
		return "", fmt.Errorf("unsupported file type: %s", mimeType)
	}
}
