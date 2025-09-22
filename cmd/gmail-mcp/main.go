// Gmail MCP server provides Gmail API access through Model Context Protocol.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
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
	"github.com/hal9000y/gmail-mcp/internal/gservice"
	"github.com/hal9000y/gmail-mcp/internal/tool"
)

func main() {
	httpAddr := flag.String("http-addr", "localhost:0", "HTTP SERVER listen addr")
	oauthTokenFile := flag.String("oauth-token-file", "./data/gmail-mcp-token.json", "Path to cache google oauth token, empty to avoid storing")
	oauthURLParam := flag.String("oauth-url", "", "OAuth URL")
	envFileParam := flag.String("env-file", "", "Path to env file")
	enableStdio := flag.Bool("stdio", false, "Enable stdio transport for MCP (disables stdout logging)")
	logFile := flag.String("log-file", "", "Path to log file (only used with stdio transport, otherwise logs to stdout)")

	flag.Parse()

	persistLogs := setupLogger(enableStdio, logFile)
	defer persistLogs()

	ln := mustListen(httpAddr)
	config := mustCreateOauthCfg(ln.Addr().String(), envFileParam, oauthURLParam)

	if oauthTokenFile == nil {
		panic("-oauth-token-file must be provided")
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

	gmailSvc := gservice.NewGmail(config, tok)
	gmailT := tool.NewServer(gmailSvc, &format.Converter{})
	mcpHTTP := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server { return gmailT }, nil)

	mux.Handle("/mcp", mcpHTTP)

	srv := &http.Server{
		Handler: mux,
	}

	shutdown := make(chan os.Signal, 1)

	signal.Notify(shutdown, syscall.SIGTERM, syscall.SIGINT)

	if _, err := tok.OAuthToken(); errors.Is(err, auth.ErrTokenNotSet) {
		openBrowser(config.RedirectURL)
	}

	stopHTTP, errHTTPCh := serveHTTP(srv, ln)
	defer stopHTTP()

	var errStdioCh <-chan error
	if *enableStdio {
		var stopStdio func()
		stopStdio, errStdioCh = serveStdio(gmailT)
		defer stopStdio()
	}

	select {
	case err := <-errHTTPCh:
		log.Println("Error http server", err)
	case err := <-errStdioCh:
		log.Println("Error stdio", err)
	case <-shutdown:
		log.Println("Shutdown signal received")
	}
}

func serveStdio(srv *mcp.Server) (func(), <-chan error) {
	errStdioCh := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer close(errStdioCh)
		log.Println("Starting stdio transport")

		if err := srv.Run(ctx, &mcp.StdioTransport{}); err != nil {
			err = fmt.Errorf("srv.Run failed: %w", err)
			errStdioCh <- err
		}
	}()

	return func() {
		cancel()

		<-errStdioCh
		log.Println("Stdio transport stopped")
	}, errStdioCh
}

func serveHTTP(srv *http.Server, ln net.Listener) (func(), <-chan error) {
	errHTTPCh := make(chan error, 1)
	go func() {
		defer close(errHTTPCh)

		log.Println("Starting http server on", ln.Addr().String())

		err := srv.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			err = fmt.Errorf("srv.ListenAndServe failed: %w", err)
			log.Println(err)
			errHTTPCh <- err
		}
	}()

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Println(fmt.Errorf("srv.Shutdown failed: %w", err))
		}

		<-errHTTPCh
		log.Println("HTTP server stopped")
	}, errHTTPCh
}

func mustListen(httpAddr *string) net.Listener {
	if httpAddr == nil {
		panic("-http-addr must be provided")
	}

	ln, err := net.Listen("tcp", *httpAddr)
	if err != nil {
		panic(fmt.Errorf("net.Listen failed: %w", err))
	}

	return ln
}

func mustCreateOauthCfg(lnAddr string, envFileParam, oauthURLParam *string) *oauth2.Config {
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

	oauthURL := fmt.Sprintf("http://%s/oauth", lnAddr)
	if oauthURLParam != nil && *oauthURLParam != "" {
		oauthURL = *oauthURLParam
	}

	return &oauth2.Config{
		ClientID:     oauthClientID,
		ClientSecret: oauthClientSec,
		RedirectURL:  oauthURL,
		Scopes:       []string{gmail.GmailReadonlyScope},
		Endpoint:     google.Endpoint,
	}
}

func setupLogger(enableStdio *bool, logFile *string) func() {
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			panic(fmt.Errorf("failed to open log file: %w", err))
		}
		log.SetOutput(f)

		return func() {
			if err := f.Close(); err != nil {
				log.Println(fmt.Errorf("f.Close failed: %w", err))
			}
		}
	}

	if *enableStdio {
		log.SetOutput(io.Discard)
	} else {
		log.SetOutput(os.Stdout)
	}

	return func() {}
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
