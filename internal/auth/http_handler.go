package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

type tok interface {
	AuthorizeCode(context.Context, string, string) error
	OAuthToken() (*oauth2.Token, error)
	RedirectURL() string
}

// HTTPHandler handles OAuth2 authentication flow via HTTP.
type HTTPHandler struct {
	tok tok
}

// NewHTTPHandler creates an HTTP handler for OAuth2 flow.
func NewHTTPHandler(tok tok) *HTTPHandler {
	return &HTTPHandler{tok: tok}
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("redirect") != "" {
		http.Redirect(w, r, h.tok.RedirectURL(), http.StatusMovedPermanently)
		return
	}

	if code := r.URL.Query().Get("code"); code != "" {
		state := r.URL.Query().Get("state")
		if err := h.tok.AuthorizeCode(r.Context(), code, state); err != nil {
			log.Println("h.tok.AuthorizeCode failed", err)
			http.Error(w, "Unable to authorize provided code", http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, r.URL.EscapedPath(), http.StatusFound)
		return
	}

	t, err := h.tok.OAuthToken()
	if errors.Is(err, ErrTokenNotSet) {
		http.Error(w, "Token not found", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "Token: %s, expires: %s", maskLeft(t.AccessToken), t.Expiry.Format(time.RFC3339))
}

func maskLeft(s string) string {
	rs := []rune(s)
	for i := 0; i < len(rs)-4; i++ {
		rs[i] = 'X'
	}
	return string(rs)
}
