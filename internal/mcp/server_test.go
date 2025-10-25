package mcp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/tuannvm/kafka-mcp-server/internal/config"
)

func TestNewMCPServer(t *testing.T) {
	s := NewMCPServer("test-server", "1.0.0")
	if s == nil {
		t.Fatal("Expected NewMCPServer to return a server instance, got nil")
	}
}

func TestNewMCPServer_WithOptions(t *testing.T) {
	mux := http.NewServeMux()
	cfg := config.Config{
		OAuthEnabled:   true,
		OAuthMode:      "native",
		OAuthProvider:  "hmac",
		OAuthServerURL: "http://localhost:8080",
		OIDCIssuer:     "http://localhost:8080",
		OIDCAudience:   "api://test",
		JWTSecret:      "test-secret-key-32-bytes-long-123",
	}

	option, oauthServer, err := CreateOAuthOption(cfg, mux)
	if err != nil {
		t.Fatalf("CreateOAuthOption failed: %v", err)
	}

	s := NewMCPServer("test-server", "1.0.0", option)
	if s == nil {
		t.Fatal("Expected NewMCPServer with OAuth option to return a server instance, got nil")
	}
	if oauthServer == nil {
		t.Error("Expected oauthServer to be non-nil when OAuth is enabled")
	}
}

func TestCreateOAuthOption_Disabled(t *testing.T) {
	cfg := config.Config{
		OAuthEnabled: false,
	}

	mux := http.NewServeMux()
	option, server, err := CreateOAuthOption(cfg, mux)

	if err != nil {
		t.Errorf("Expected no error when OAuth is disabled, got: %v", err)
	}
	if option != nil {
		t.Error("Expected nil option when OAuth is disabled")
	}
	if server != nil {
		t.Error("Expected nil server when OAuth is disabled")
	}
}

func TestCreateOAuthOption_NoMux(t *testing.T) {
	cfg := config.Config{
		OAuthEnabled: true,
	}

	option, server, err := CreateOAuthOption(cfg, nil)

	if err == nil {
		t.Error("Expected error when mux is nil and OAuth is enabled")
	}
	if err != nil && err.Error() != "mux is required when OAuth is enabled" {
		t.Errorf("Expected 'mux is required' error, got: %v", err)
	}
	if option != nil {
		t.Error("Expected nil option on error")
	}
	if server != nil {
		t.Error("Expected nil server on error")
	}
}

func TestCreateOAuthOption_NativeMode(t *testing.T) {
	mux := http.NewServeMux()
	cfg := config.Config{
		OAuthEnabled:   true,
		OAuthMode:      "native",
		OAuthProvider:  "hmac",
		OAuthServerURL: "http://localhost:8080",
		OIDCIssuer:     "http://localhost:8080",
		OIDCAudience:   "api://test-server",
		JWTSecret:      "test-jwt-secret-minimum-32-bytes-long",
	}

	option, oauthServer, err := CreateOAuthOption(cfg, mux)

	if err != nil {
		t.Fatalf("CreateOAuthOption failed for native mode: %v", err)
	}
	if option == nil {
		t.Error("Expected non-nil option for native mode")
	}
	if oauthServer == nil {
		t.Error("Expected non-nil oauthServer for native mode")
	}
}

func TestCreateOAuthOption_ProxyMode(t *testing.T) {
	mux := http.NewServeMux()
	cfg := config.Config{
		OAuthEnabled:      true,
		OAuthMode:         "proxy",
		OAuthProvider:     "google",
		OAuthServerURL:    "http://localhost:8080",
		OIDCIssuer:        "https://accounts.google.com",
		OIDCAudience:      "api://test-server",
		OIDCClientID:      "test-client-id",
		OIDCClientSecret:  "test-client-secret",
		OAuthRedirectURIs: "http://localhost:8080/oauth/callback",
		JWTSecret:         "proxy-mode-jwt-secret-32-bytes-min",
	}

	option, oauthServer, err := CreateOAuthOption(cfg, mux)

	if err != nil {
		t.Fatalf("CreateOAuthOption failed for proxy mode: %v", err)
	}
	if option == nil {
		t.Error("Expected non-nil option for proxy mode")
	}
	if oauthServer == nil {
		t.Error("Expected non-nil oauthServer for proxy mode")
	}
}

func TestStart_StdioMode(t *testing.T) {
	cfg := config.Config{
		MCPTransport: "stdio",
	}

	s := NewMCPServer("test", "1.0.0")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- Start(ctx, s, cfg, nil)
	}()

	select {
	case err := <-errCh:
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("Start failed for stdio mode: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Log("STDIO mode started successfully (timeout expected)")
	}
}

func TestStart_HTTPMode_NoMux(t *testing.T) {
	cfg := config.Config{
		MCPTransport: "http",
		HTTPPort:     18080,
	}

	s := NewMCPServer("test", "1.0.0")
	ctx := context.Background()

	err := Start(ctx, s, cfg, nil)

	if err == nil {
		t.Error("Expected error when starting HTTP mode without mux")
	}
	if err != nil && err.Error() != "mux is required for HTTP transport" {
		t.Errorf("Expected 'mux is required' error, got: %v", err)
	}
}

func TestStart_UnsupportedTransport(t *testing.T) {
	cfg := config.Config{
		MCPTransport: "websocket",
	}

	s := NewMCPServer("test", "1.0.0")
	ctx := context.Background()

	err := Start(ctx, s, cfg, nil)

	if err == nil {
		t.Error("Expected error for unsupported transport")
	}
	expectedMsg := "unsupported MCP transport: websocket"
	if err != nil && err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %v", expectedMsg, err)
	}
}

func TestStartHTTPServer_WithoutOAuth(t *testing.T) {
	if os.Getenv("SKIP_HTTP_TEST") != "" {
		t.Skip("Skipping HTTP server test")
	}

	cfg := config.Config{
		MCPTransport: "http",
		HTTPPort:     18081,
		OAuthEnabled: false,
	}

	mux := http.NewServeMux()
	s := NewMCPServer("test", "1.0.0")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- Start(ctx, s, cfg, mux)
	}()

	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get("http://localhost:18081/mcp")
	if err != nil {
		t.Fatalf("Failed to connect to MCP endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 0 {
		t.Error("Expected valid HTTP response from /mcp endpoint")
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("HTTP server returned unexpected error: %v", err)
		}
	case <-time.After(6 * time.Second):
		t.Error("Server did not shutdown gracefully within timeout")
	}
}

func TestStartHTTPServer_WithOAuth_HMAC(t *testing.T) {
	if os.Getenv("SKIP_HTTP_TEST") != "" {
		t.Skip("Skipping HTTP server test")
	}

	cfg := config.Config{
		MCPTransport:   "http",
		HTTPPort:       18082,
		OAuthEnabled:   true,
		OAuthMode:      "native",
		OAuthProvider:  "hmac",
		OAuthServerURL: "http://localhost:18082",
		OIDCIssuer:     "http://localhost:18082",
		OIDCAudience:   "api://test-server",
		JWTSecret:      "test-hmac-jwt-secret-32-bytes-long",
	}

	mux := http.NewServeMux()

	option, oauthServer, err := CreateOAuthOption(cfg, mux)
	if err != nil {
		t.Fatalf("CreateOAuthOption failed: %v", err)
	}
	if option == nil {
		t.Fatal("Expected non-nil OAuth option")
	}
	if oauthServer == nil {
		t.Fatal("Expected non-nil OAuth server")
	}

	s := NewMCPServer("test", "1.0.0", option)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- Start(ctx, s, cfg, mux)
	}()

	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get("http://localhost:18082/.well-known/oauth-authorization-server")
	if err != nil {
		t.Fatalf("Failed to connect to OAuth metadata endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for OAuth metadata endpoint, got %d", resp.StatusCode)
	}

	resp2, err := http.Get("http://localhost:18082/mcp")
	if err != nil {
		t.Fatalf("Failed to connect to MCP endpoint: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	cancel()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("HTTP server returned unexpected error: %v", err)
		}
	case <-time.After(6 * time.Second):
		t.Error("Server did not shutdown gracefully within timeout")
	}
}

func TestStartHTTPServer_GracefulShutdown(t *testing.T) {
	if os.Getenv("SKIP_HTTP_TEST") != "" {
		t.Skip("Skipping HTTP server test")
	}

	cfg := config.Config{
		MCPTransport: "http",
		HTTPPort:     18083,
		OAuthEnabled: false,
	}

	mux := http.NewServeMux()
	s := NewMCPServer("test", "1.0.0")

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	startTime := time.Now()

	go func() {
		errCh <- Start(ctx, s, cfg, mux)
	}()

	time.Sleep(100 * time.Millisecond)

	cancel()

	select {
	case err := <-errCh:
		shutdownDuration := time.Since(startTime)
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Expected http.ErrServerClosed or nil, got: %v", err)
		}
		if shutdownDuration > 6*time.Second {
			t.Errorf("Graceful shutdown took too long: %v (expected < 6s)", shutdownDuration)
		}
		t.Logf("Server shutdown gracefully in %v", shutdownDuration)
	case <-time.After(7 * time.Second):
		t.Error("Server did not shutdown within 7 seconds (5s timeout + 2s buffer)")
	}
}

func TestCreateOAuthOption_InvalidConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfg         config.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "Missing Issuer",
			cfg: config.Config{
				OAuthEnabled:   true,
				OAuthProvider:  "okta",
				OAuthServerURL: "http://localhost:8080",
				OIDCAudience:   "api://test",
			},
			expectError: true,
		},
		{
			name: "Missing Audience",
			cfg: config.Config{
				OAuthEnabled:   true,
				OAuthProvider:  "okta",
				OAuthServerURL: "http://localhost:8080",
				OIDCIssuer:     "https://company.okta.com",
			},
			expectError: true,
		},
		{
			name: "Proxy mode missing ClientID",
			cfg: config.Config{
				OAuthEnabled:   true,
				OAuthMode:      "proxy",
				OAuthProvider:  "google",
				OAuthServerURL: "http://localhost:8080",
				OIDCIssuer:     "https://accounts.google.com",
				OIDCAudience:   "api://test",
				JWTSecret:      "test-secret-key",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			option, server, err := CreateOAuthOption(tt.cfg, mux)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
			if tt.expectError && option != nil {
				t.Error("Expected nil option on error")
			}
			if tt.expectError && server != nil {
				t.Error("Expected nil server on error")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestStartHTTPServer_PortConflict(t *testing.T) {
	if os.Getenv("SKIP_HTTP_TEST") != "" {
		t.Skip("Skipping HTTP server test")
	}

	port := 18084

	go func() {
		_ = http.ListenAndServe(fmt.Sprintf(":%d", port), http.NewServeMux())
	}()
	time.Sleep(100 * time.Millisecond)

	cfg := config.Config{
		MCPTransport: "http",
		HTTPPort:     port,
		OAuthEnabled: false,
	}

	mux := http.NewServeMux()
	s := NewMCPServer("test", "1.0.0")
	ctx := context.Background()

	err := Start(ctx, s, cfg, mux)

	if err == nil {
		t.Error("Expected error when port is already in use")
	}
}

func TestStartHTTPServer_MultipleProviders(t *testing.T) {
	if os.Getenv("SKIP_HTTP_TEST") != "" {
		t.Skip("Skipping HTTP server test")
	}

	tests := []struct {
		provider    string
		shouldWork  bool
		description string
	}{
		{"hmac", true, "HMAC provider should work with JWTSecret"},
		{"okta", false, "Okta requires real OIDC provider"},
		{"google", false, "Google requires real OIDC provider"},
		{"azure", false, "Azure requires real OIDC provider"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("Provider_%s", tt.provider), func(t *testing.T) {
			port := 18085 + i
			cfg := config.Config{
				MCPTransport:   "http",
				HTTPPort:       port,
				OAuthEnabled:   true,
				OAuthMode:      "native",
				OAuthProvider:  tt.provider,
				OAuthServerURL: fmt.Sprintf("http://localhost:%d", port),
				OIDCIssuer:     fmt.Sprintf("http://localhost:%d", port),
				OIDCAudience:   "api://test",
				JWTSecret:      "test-jwt-secret-32-bytes-minimum-ok",
			}

			mux := http.NewServeMux()
			option, _, err := CreateOAuthOption(cfg, mux)

			if tt.shouldWork {
				if err != nil {
					t.Errorf("CreateOAuthOption failed for %s: %v", tt.provider, err)
				}
				if option == nil {
					t.Errorf("Expected non-nil option for %s", tt.provider)
				}
			} else {
				if err != nil {
					t.Logf("%s - Expected error without real provider: %v", tt.description, err)
				}
			}
		})
	}
}
