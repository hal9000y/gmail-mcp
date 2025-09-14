package tool

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/api/gmail/v1"
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

type previewAttachmentsSvc interface {
	GetMessage(ctx context.Context, msgID string) (*gmail.Message, error)
	GetAttachment(ctx context.Context, msgID, attachmentID string) (*gmail.MessagePartBody, error)
}

type pdfConverter interface {
	PDF2MD(raw []byte) (string, error)
}

func NewPreviewAttachments(svc previewAttachmentsSvc, conv pdfConverter) *PreviewAttachments {
	return &PreviewAttachments{
		svc:  svc,
		conv: conv,
	}
}

type PreviewAttachments struct {
	svc  previewAttachmentsSvc
	conv pdfConverter
}

func (t *PreviewAttachments) PreviewAttachments(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input PreviewAttachmentsRequest,
) (*mcp.CallToolResult, PreviewAttachmentsResponse, error) {
	msg, err := t.svc.GetMessage(ctx, input.MessageID)
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

		attachment, err := t.svc.GetAttachment(ctx, input.MessageID, attachID)
		if err != nil {
			return nil, PreviewAttachmentsResponse{}, fmt.Errorf("get attachment %s failed: %w", attachID, err)
		}

		preview := AttachmentPreview{
			ID:       partID,
			Filename: fileName,
			MimeType: mimeType,
		}

		data, err := t.extractAttachmentContent(attachment.Data, preview.MimeType, preview.Filename)
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

func (t *PreviewAttachments) extractAttachmentContent(data, mimeType, filename string) (string, error) {
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
		return t.conv.PDF2MD(decodedData)

	case strings.HasSuffix(filename, ".txt") || strings.HasSuffix(filename, ".md"):
		return string(decodedData), nil

	case strings.HasSuffix(filename, ".csv"):
		return string(decodedData), nil

	default:
		return "", fmt.Errorf("unsupported file type: %s", mimeType)
	}
}
