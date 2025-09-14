package gservice

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/hal9000y/gmail-mcp/internal/auth"
)

const gmailUserID = "me"

func NewGmail(cfg *oauth2.Config, tok *auth.Token) *GMail {
	return &GMail{
		cfg: cfg,
		tok: tok,
	}
}

type GMail struct {
	cfg *oauth2.Config
	tok *auth.Token
}

func (m *GMail) ListMessages(ctx context.Context, Q, pageToken string, maxResults int64) (*gmail.ListMessagesResponse, error) {
	svc, err := m.newSvc(ctx)
	if err != nil {
		return nil, fmt.Errorf("newSvc failed: %w", err)
	}

	call := svc.Users.Messages.List(gmailUserID).
		Q(Q).
		PageToken(pageToken).
		MaxResults(maxResults)

	result, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("messages.List failed: %w", err)
	}

	return result, nil
}

func (m *GMail) GetMessageMetadata(ctx context.Context, msgID string) (*gmail.Message, error) {
	svc, err := m.newSvc(ctx)
	if err != nil {
		return nil, fmt.Errorf("newSvc failed: %w", err)
	}

	msg, err := svc.Users.Messages.Get(gmailUserID, msgID).
		Format("METADATA").
		MetadataHeaders("From", "To", "Cc", "Subject", "Date").
		Do()
	if err != nil {
		return nil, fmt.Errorf("messages.Get failed: %w", err)
	}

	return msg, nil
}

func (m *GMail) GetMessage(ctx context.Context, msgID string) (*gmail.Message, error) {
	svc, err := m.newSvc(ctx)
	if err != nil {
		return nil, fmt.Errorf("newSvc failed: %w", err)
	}

	msg, err := svc.Users.Messages.Get(gmailUserID, msgID).Do()
	if err != nil {
		return nil, fmt.Errorf("messages.Get failed: %w", err)
	}

	return msg, nil
}

func (m *GMail) GetAttachment(ctx context.Context, msgID, attachmentID string) (*gmail.MessagePartBody, error) {
	svc, err := m.newSvc(ctx)
	if err != nil {
		return nil, fmt.Errorf("newSvc failed: %w", err)
	}

	attachment, err := svc.Users.Messages.Attachments.Get(gmailUserID, msgID, attachmentID).Do()
	if err != nil {
		return nil, fmt.Errorf("attachments.Get failed: %w", err)
	}

	return attachment, nil
}

func (m *GMail) newSvc(ctx context.Context) (*gmail.Service, error) {
	t, err := m.tok.OAuthToken()
	if err != nil {
		return nil, fmt.Errorf("tok.OAuthToken failed: %w", err)
	}

	clt := m.cfg.Client(ctx, t)

	svc, err := gmail.NewService(ctx, option.WithHTTPClient(clt))
	if err != nil {
		return nil, fmt.Errorf("gmail.NewService failed: %w", err)
	}

	return svc, nil
}
