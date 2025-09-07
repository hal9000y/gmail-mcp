package gservice

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/hal9000y/gmail-mcp/internal/auth"
)

func NewGmail(ctx context.Context, cfg *oauth2.Config, tok *auth.Token) (*gmail.Service, error) {
	t, err := tok.OAuthToken()
	if err != nil {
		return nil, fmt.Errorf("tok.OAuthToken failed: %w", err)
	}

	clt := cfg.Client(ctx, t)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(clt))
	if err != nil {
		return nil, fmt.Errorf("gmail.NewService failed: %w", err)
	}

	return srv, nil
}
