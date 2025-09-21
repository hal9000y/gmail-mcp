// Package auth handles OAuth2 token management and persistence.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"sync"
	"time"

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
	stateStore  map[string]time.Time
}

// NewToken creates a Token manager, loading from disk if path provided.
func NewToken(cfg *oauth2.Config, persistPath string) (*Token, error) {
	t := &Token{
		cfg:         cfg,
		persistPath: persistPath,
		stateStore:  make(map[string]time.Time),
	}
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

// RedirectURL generates the OAuth2 authorization URL with a secure random state.
func (t *Token) RedirectURL() string {
	state := t.generateState()
	return t.cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// generateState creates a cryptographically secure random state value.
func (t *Token) generateState() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Printf("Failed to generate random state: %v", err)
		return ""
	}
	state := base64.URLEncoding.EncodeToString(b)

	t.mu.Lock()
	defer t.mu.Unlock()

	// Store state with expiration (5 minutes)
	t.stateStore[state] = time.Now().Add(5 * time.Minute)

	// Clean up expired states
	now := time.Now()
	for s, exp := range t.stateStore {
		if exp.Before(now) {
			delete(t.stateStore, s)
		}
	}

	return state
}

// ValidateState checks if the provided state is valid and not expired.
func (t *Token) ValidateState(state string) bool {
	if state == "" {
		return false
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	expiry, exists := t.stateStore[state]
	if !exists {
		return false
	}

	if time.Now().After(expiry) {
		delete(t.stateStore, state)
		return false
	}

	// State is valid, remove it (one-time use)
	delete(t.stateStore, state)
	return true
}

// AuthorizeCode exchanges an authorization code for an access token after validating state.
func (t *Token) AuthorizeCode(ctx context.Context, code string, state string) error {
	if !t.ValidateState(state) {
		return errors.New("invalid or expired state parameter")
	}

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
