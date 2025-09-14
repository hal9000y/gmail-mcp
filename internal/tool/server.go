package tool

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type gmailSvc interface {
	getMessagesSvc
	searchMessagesSvc
	previewAttachmentsSvc
}

type cnv interface {
	htmlConverter
	pdfConverter
}

// NewServer creates an MCP server with Gmail tools.
func NewServer(svc gmailSvc, cnv cnv) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "gmail-helper", Version: "v1.0.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_messages",
		Description: "Search Gmail messages using Gmail search syntax",
	}, NewSearchMessages(svc).SearchMessages)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_messages",
		Description: "Get full message content for specified message IDs",
	}, NewGetMessages(svc, cnv).GetMessages)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "preview_attachments",
		Description: "Extract text content from attachments (PDFs, text files, etc)",
	}, NewPreviewAttachments(svc, cnv).PreviewAttachments)

	return server
}
