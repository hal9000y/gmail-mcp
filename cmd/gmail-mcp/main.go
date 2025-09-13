package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"

	"github.com/hal9000y/gmail-mcp/internal/auth"
	"github.com/hal9000y/gmail-mcp/internal/format"
	"github.com/hal9000y/gmail-mcp/internal/tool"
)

func main() {
	httpAddr := flag.String("http-addr", "localhost:0", "HTTP SERVER listen addr")
	oauthTokenFile := flag.String("oauth-token-file", ".__gmail-mcp-token.json", "Path to cache google oauth token, empty to avoid storing")
	oauthURLParam := flag.String("oauth-url", "", "OAuth URL")
	envFileParam := flag.String("env-file", ".env.local", "Path to env file")

	flag.Parse()

	if httpAddr == nil || oauthTokenFile == nil {
		panic("incomplete parameters provided")
	}

	if envFileParam != nil && *envFileParam != "" {
		if err := godotenv.Load(*envFileParam); err != nil {
			panic(fmt.Errorf("godotenv.Load failed: %w", err))
		}
	}

	oauthClientID := os.Getenv("OAUTH_GOOGLE_CLIENT_ID")
	oauthClientSec := os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET")

	if oauthClientID == "" || oauthClientSec == "" {
		panic("Env variables OAUTH_GOOGLE_CLIENT_ID and OAUTH_GOOGLE_CLIENT_SECRET must be set")
	}

	ln, err := net.Listen("tcp", *httpAddr)
	if err != nil {
		panic(fmt.Errorf("net.Listen failed: %w", err))
	}

	oauthURL := fmt.Sprintf("http://%s/oauth", ln.Addr().String())
	if oauthURLParam != nil && *oauthURLParam != "" {
		oauthURL = *oauthURLParam
	}

	config := &oauth2.Config{
		ClientID:     oauthClientID,
		ClientSecret: oauthClientSec,
		RedirectURL:  oauthURL,
		Scopes:       []string{gmail.GmailReadonlyScope},
		Endpoint:     google.Endpoint,
	}

	tok, err := auth.NewToken(config, *oauthTokenFile)
	if err != nil {
		panic(fmt.Errorf("auth.NewToken failed: %w", err))
	}

	defer func() {
		log.Println("Persisting token if exists")
		if err := tok.Persist(); err != nil {
			log.Println(fmt.Errorf("tok.Persist failed: %w", err))
		}
	}()

	authHTTP := auth.NewHTTPHandler(tok)

	mux := http.NewServeMux()
	mux.Handle("/oauth", authHTTP)

	conv := format.Converter{}
	gmailH := tool.NewGmailHandler(conv, config, tok)
	gmailT := tool.NewGmailToolSet(gmailH)
	mcpHTTP := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server { return gmailT }, nil)

	mux.Handle("/mcp", loggingHandler(mcpHTTP))

	srv := http.Server{
		Handler: mux,
	}

	shutdown := make(chan os.Signal, 1)

	signal.Notify(shutdown, syscall.SIGTERM, syscall.SIGINT)

	if _, err := tok.OAuthToken(); errors.Is(err, auth.TokenNotSet) {
		openBrowser(oauthURL)
	}

	errHTTPCh := make(chan error, 1)
	go func() {
		log.Println("Starting http server on", ln.Addr().String())

		err := srv.Serve(ln)
		if err != nil {
			err = fmt.Errorf("srv.ListenAndServe failed: %w", err)
		}
		if !errors.Is(err, http.ErrServerClosed) {
			log.Println("HTTP server closed with error", err)
		}

		errHTTPCh <- err
	}()

	stdioCtx, stdioCancel := context.WithCancel(context.Background())
	defer stdioCancel()
	errStdioCh := make(chan error, 1)
	go func() {
		log.Println("Starting stdio transport")
		err := gmailT.Run(stdioCtx, &mcp.StdioTransport{})
		if err != nil {
			err = fmt.Errorf("gmailT.Run failed: %w", err)
			log.Println("gmailT.Run failed", err)
		}

		errStdioCh <- err
	}()

	select {
	case err := <-errHTTPCh:
		log.Println("Error http server", err)
	case err := <-errStdioCh:
		log.Println("Error stdio", err)
	case <-shutdown:
		log.Println("Shutdown signal received")
	}

	shCtx, shCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shCancel()

	if err := srv.Shutdown(shCtx); err != nil {
		log.Println(fmt.Errorf("srv.Shutdown failed: %w", err))
	}

	stdioCancel()

	<-errStdioCh
	<-errHTTPCh
}

func openBrowser(url string) {
	url = fmt.Sprintf("%s?redirect=1", url)
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		log.Printf("Could not open browser automatically: %v; please copy and open link in the browser: %s\n", err, url)
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func loggingHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code.
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Log request details.
		log.Printf("[REQUEST] %s | %s | %s %s",
			start.Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			r.URL.Path)

		// Call the actual handler.
		handler.ServeHTTP(wrapped, r)

		// Log response details.
		duration := time.Since(start)
		log.Printf("[RESPONSE] %s | %s | %s %s | Status: %d | Duration: %v",
			time.Now().Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration)
	})
}
