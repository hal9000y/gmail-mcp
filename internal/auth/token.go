// Package auth handles OAuth2 token management and persistence.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"sync"

	"golang.org/x/oauth2"
)

// ErrTokenNotSet indicates no OAuth token is available.
var ErrTokenNotSet = errors.New("no token defined")

// Token manages OAuth2 tokens with thread-safe operations.
type Token struct {
	mu          sync.RWMutex
	cfg         *oauth2.Config
	token       *oauth2.Token
	persistPath string
}

// NewToken creates a Token manager, loading from disk if path provided.
func NewToken(cfg *oauth2.Config, persistPath string) (*Token, error) {
	t := &Token{cfg: cfg, persistPath: persistPath}
	if persistPath == "" {
		return t, nil
	}

	f, err := os.Open(persistPath)
	defer func() { _ = f.Close() }()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Printf("File %s doesn't exist, but will be created at the end", persistPath)

			return t, nil
		}

		return nil, fmt.Errorf("os.Open failed: %w", err)
	}

	token := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(token); err != nil {
		return nil, fmt.Errorf("json.NewDecoder.Decode failed: %w", err)
	}
	t.token = token

	return t, nil
}

// RedirectURL generates the OAuth2 authorization URL.
func (t *Token) RedirectURL() string {
	return t.cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

// AuthorizeCode exchanges an authorization code for an access token.
func (t *Token) AuthorizeCode(ctx context.Context, code string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	tok, err := t.cfg.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("cfg.Exchange failed: %w", err)
	}

	t.token = tok

	return nil
}

// OAuthToken returns the current OAuth2 token.
func (t *Token) OAuthToken() (*oauth2.Token, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.token == nil {
		return nil, ErrTokenNotSet
	}

	return t.token, nil
}

// Persist saves the token to disk.
func (t *Token) Persist() error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.persistPath == "" || t.token == nil {
		return nil
	}

	f, err := os.OpenFile(t.persistPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer func() { _ = f.Close() }()
	if err != nil {
		return fmt.Errorf("os.OpenFile failed: %w", err)
	}

	if err := json.NewEncoder(f).Encode(t.token); err != nil {
		return fmt.Errorf("json.NewEncoder.Encode failed: %w", err)
	}

	return nil
}
