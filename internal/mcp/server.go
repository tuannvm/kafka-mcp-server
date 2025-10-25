package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	oauth "github.com/tuannvm/oauth-mcp-proxy"
	"github.com/tuannvm/oauth-mcp-proxy/mark3labs"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tuannvm/kafka-mcp-server/internal/config"
)

// NewMCPServer creates a new MCP server instance.
func NewMCPServer(name, version string, opts ...server.ServerOption) *server.MCPServer {
	// Configure logging (optional, customize as needed)
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	srv := server.NewMCPServer(name, version, opts...)
	return srv
}

// CreateOAuthOption creates OAuth server option if OAuth is enabled.
// This function MUST be called before creating the MCPServer instance.
//
// Returns:
//   - server.ServerOption: The OAuth option to pass to NewMCPServer (nil if OAuth disabled)
//   - *oauth.Server: The OAuth server instance for logging and management (nil if OAuth disabled)
//   - error: Any error during OAuth setup
//
// The mux parameter must be a pre-created http.ServeMux where OAuth routes will be registered.
func CreateOAuthOption(cfg config.Config, mux *http.ServeMux) (server.ServerOption, *oauth.Server, error) {
	if !cfg.OAuthEnabled {
		return nil, nil, nil
	}

	if mux == nil {
		return nil, nil, fmt.Errorf("mux is required when OAuth is enabled")
	}

	oauthConfig := &oauth.Config{
		Provider:  cfg.OAuthProvider,
		Mode:      cfg.OAuthMode,
		Issuer:    cfg.OIDCIssuer,
		Audience:  cfg.OIDCAudience,
		ServerURL: cfg.OAuthServerURL,
	}

	if cfg.OAuthProvider == "hmac" {
		oauthConfig.JWTSecret = []byte(cfg.JWTSecret)
	}

	if cfg.OAuthMode == "proxy" {
		oauthConfig.ClientID = cfg.OIDCClientID
		oauthConfig.ClientSecret = cfg.OIDCClientSecret
		oauthConfig.RedirectURIs = cfg.OAuthRedirectURIs
		if cfg.OAuthProvider != "hmac" {
			oauthConfig.JWTSecret = []byte(cfg.JWTSecret)
		}
	}

	oauthServer, oauthOption, err := mark3labs.WithOAuth(mux, oauthConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup OAuth: %w", err)
	}

	slog.Info("OAuth configured",
		"mode", cfg.OAuthMode,
		"provider", cfg.OAuthProvider,
		"issuer", cfg.OIDCIssuer)

	return oauthOption, oauthServer, nil
}

// Start runs the MCP server based on the configured transport.
// For HTTP transport, mux must be provided (can be nil for stdio).
func Start(ctx context.Context, s *server.MCPServer, cfg config.Config, mux *http.ServeMux) error {
	slog.Info("Starting MCP server", "transport", cfg.MCPTransport)

	switch cfg.MCPTransport {
	case "stdio":
		return server.ServeStdio(s)
	case "http":
		return startHTTPServer(ctx, s, cfg, mux)
	default:
		return fmt.Errorf("unsupported MCP transport: %s", cfg.MCPTransport)
	}
}

func startHTTPServer(ctx context.Context, s *server.MCPServer, cfg config.Config, mux *http.ServeMux) error {
	if mux == nil {
		return fmt.Errorf("mux is required for HTTP transport")
	}

	// Create StreamableHTTPServer with token extraction
	// CreateHTTPContextFunc extracts Bearer tokens from Authorization header
	streamable := server.NewStreamableHTTPServer(
		s,
		server.WithHTTPContextFunc(oauth.CreateHTTPContextFunc()),
	)
	mux.Handle("/mcp", streamable)

	addr := fmt.Sprintf(":%d", cfg.HTTPPort)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		slog.Info("HTTP server shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("HTTP server shutdown error", "error", err)
		}
	}()

	slog.Info("Starting HTTP server",
		"address", addr,
		"oauth_enabled", cfg.OAuthEnabled,
		"mcp_endpoint", "/mcp")

	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
}
