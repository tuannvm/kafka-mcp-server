# OAuth 2.1 Implementation Guide

## Overview

This document tracks the step-by-step implementation of OAuth 2.1 authentication for kafka-mcp-server using oauth-mcp-proxy@v1.0.0. Follow this guide sequentially and update checkboxes as you complete each step.

**CRITICAL ARCHITECTURAL NOTE**: OAuth option MUST be passed to `NewMCPServer()` at creation time. This requires refactoring main.go to create the OAuth option before creating the MCP server instance.

## Implementation Progress

- [ ] Phase 1: Add Dependencies
- [ ] Phase 2: Update Configuration (internal/config/config.go)
- [ ] Phase 3: Add OAuth Helper Function (internal/mcp/server.go)
- [ ] Phase 4: Refactor Main Entry Point (cmd/main.go)
- [ ] Phase 5: Update Server Start Function (internal/mcp/server.go)
- [ ] Phase 6: Update Documentation (CLAUDE.md)
- [ ] Phase 7: Unit Tests
- [ ] Phase 8: Integration Tests
- [ ] Phase 9: Manual Testing
- [ ] Phase 10: Security Review

---

## Phase 1: Add Dependencies

### Task
Add oauth-mcp-proxy library to the project.

### Commands
```bash
go get github.com/tuannvm/oauth-mcp-proxy@v1.0.0
go mod tidy
```

### Verification
```bash
grep "github.com/tuannvm/oauth-mcp-proxy" go.mod
```

Expected output: `github.com/tuannvm/oauth-mcp-proxy v1.0.0`

---

## Phase 2: Update Configuration

### File: `internal/config/config.go`

### Changes Required

#### 1. Add Import for strconv

Ensure `strconv` is imported:

```go
import (
	"os"
	"strconv"
	"strings"
)
```

#### 2. Add New Fields to Config Struct

Add after existing fields:

```go
// HTTP Server Configuration
HTTPPort int // HTTP server port (default: 8080)

// OAuth Configuration
OAuthEnabled    bool
OAuthMode       string // "native" or "proxy"
OAuthProvider   string // "hmac", "okta", "google", "azuread"
OAuthServerURL  string // Base URL for the MCP server

// OIDC Configuration
OIDCIssuer       string
OIDCClientID     string
OIDCClientSecret string
OIDCAudience     string

// Proxy Mode Configuration
OAuthRedirectURIs string // Comma-separated redirect URIs
JWTSecret         string // Will be converted to []byte for oauth library
```

#### 3. Update LoadConfig Function

Add environment variable parsing (insert before the return statement):

```go
func LoadConfig() Config {
	// ... existing broker/client/transport/SASL/TLS code ...

	// HTTP Port
	httpPortStr := getEnv("MCP_HTTP_PORT", "8080")
	httpPort, err := strconv.Atoi(httpPortStr)
	if err != nil {
		slog.Warn("Invalid MCP_HTTP_PORT value, using default 8080", "value", httpPortStr)
		httpPort = 8080
	}

	// OAuth Configuration
	oauthEnabledStr := getEnv("OAUTH_ENABLED", "false")
	oauthEnabled, err := strconv.ParseBool(oauthEnabledStr)
	if err != nil {
		slog.Warn("Invalid OAUTH_ENABLED value, using default false", "value", oauthEnabledStr)
		oauthEnabled = false
	}
	oauthMode := getEnv("OAUTH_MODE", "native")
	oauthProvider := getEnv("OAUTH_PROVIDER", "okta")
	oauthServerURL := getEnv("OAUTH_SERVER_URL", "")

	// OIDC Configuration
	oidcIssuer := getEnv("OIDC_ISSUER", "")
	oidcClientID := getEnv("OIDC_CLIENT_ID", "")
	oidcClientSecret := getEnv("OIDC_CLIENT_SECRET", "")
	oidcAudience := getEnv("OIDC_AUDIENCE", "")

	// Proxy Mode Configuration
	oauthRedirectURIs := getEnv("OAUTH_REDIRECT_URIS", "")
	jwtSecret := getEnv("JWT_SECRET", "")

	return Config{
		// ... existing fields ...

		HTTPPort: httpPort,

		OAuthEnabled:   oauthEnabled,
		OAuthMode:      oauthMode,
		OAuthProvider:  oauthProvider,
		OAuthServerURL: oauthServerURL,

		OIDCIssuer:       oidcIssuer,
		OIDCClientID:     oidcClientID,
		OIDCClientSecret: oidcClientSecret,
		OIDCAudience:     oidcAudience,

		OAuthRedirectURIs: oauthRedirectURIs,
		JWTSecret:         jwtSecret,
	}
}
```

### Verification

Create a test file `internal/config/config_test.go`:

```go
package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_OAuthDefaults(t *testing.T) {
	os.Clearenv()
	cfg := LoadConfig()

	assert.False(t, cfg.OAuthEnabled)
	assert.Equal(t, "native", cfg.OAuthMode)
	assert.Equal(t, "okta", cfg.OAuthProvider)
	assert.Equal(t, 8080, cfg.HTTPPort)
}

func TestLoadConfig_OAuthNativeMode(t *testing.T) {
	os.Clearenv()
	os.Setenv("OAUTH_ENABLED", "true")
	os.Setenv("OAUTH_MODE", "native")
	os.Setenv("OAUTH_PROVIDER", "okta")
	os.Setenv("OAUTH_SERVER_URL", "https://localhost:8080")
	os.Setenv("OIDC_ISSUER", "https://company.okta.com")
	os.Setenv("OIDC_AUDIENCE", "api://mcp-server")
	defer os.Clearenv()

	cfg := LoadConfig()

	assert.True(t, cfg.OAuthEnabled)
	assert.Equal(t, "native", cfg.OAuthMode)
	assert.Equal(t, "okta", cfg.OAuthProvider)
	assert.Equal(t, "https://localhost:8080", cfg.OAuthServerURL)
	assert.Equal(t, "https://company.okta.com", cfg.OIDCIssuer)
	assert.Equal(t, "api://mcp-server", cfg.OIDCAudience)
}
```

Run test:
```bash
go test ./internal/config/... -v
```

---

## Phase 3: Add OAuth Helper Function

### File: `internal/mcp/server.go`

### Changes Required

#### 1. Add Imports

Update imports at the top of the file:

```go
import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	oauth "github.com/tuannvm/oauth-mcp-proxy"
	"github.com/tuannvm/oauth-mcp-proxy/mark3labs"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tuannvm/kafka-mcp-server/internal/config"
)
```

#### 2. Add CreateOAuthOption Function

Add this function to `internal/mcp/server.go`:

```go
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

	if cfg.OAuthMode == "proxy" {
		oauthConfig.ClientID = cfg.OIDCClientID
		oauthConfig.ClientSecret = cfg.OIDCClientSecret
		oauthConfig.RedirectURIs = cfg.OAuthRedirectURIs
		oauthConfig.JWTSecret = []byte(cfg.JWTSecret)
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
```

### Verification

Ensure the file compiles:

```bash
go build ./internal/mcp/...
```

Expected: No errors

---

## Phase 4: Refactor Main Entry Point

### File: `cmd/main.go`

### Changes Required

**CRITICAL**: This is the most significant change. The MCP server must be created AFTER the OAuth option is prepared.

#### 1. Add Imports

Ensure these imports are present:

```go
import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	oauth "github.com/tuannvm/oauth-mcp-proxy"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tuannvm/kafka-mcp-server/internal/config"
	"github.com/tuannvm/kafka-mcp-server/internal/kafka"
	"github.com/tuannvm/kafka-mcp-server/internal/mcp"
)
```

#### 2. Refactor main() Function

Replace the server creation section:

```go
func main() {
	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT and SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		slog.Info("Received signal, shutting down", "signal", sig)
		cancel()
	}()

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize Kafka client
	kafkaClient, err := kafka.NewClient(cfg)
	if err != nil {
		slog.Error("Failed to create Kafka client", "error", err)
		os.Exit(1)
	}
	defer kafkaClient.Close()

	// Create HTTP mux and OAuth option if using HTTP transport
	var mux *http.ServeMux
	var oauthOption server.ServerOption
	var oauthServer *oauth.Server

	if cfg.MCPTransport == "http" {
		mux = http.NewServeMux()
		oauthOption, oauthServer, err = mcp.CreateOAuthOption(cfg, mux)
		if err != nil {
			slog.Error("Failed to create OAuth option", "error", err)
			os.Exit(1)
		}
	}

	// Create MCP server with OAuth option if provided
	var s *server.MCPServer
	if oauthOption != nil {
		s = mcp.NewMCPServer("kafka-mcp-server", Version, oauthOption)
	} else {
		s = mcp.NewMCPServer("kafka-mcp-server", Version)
	}

	// Explicitly declare the client as the KafkaClient interface type
	var kafkaInterface kafka.KafkaClient = kafkaClient

	// Register MCP resources and tools
	mcp.RegisterResources(s, kafkaInterface)
	mcp.RegisterTools(s, kafkaInterface, cfg)
	mcp.RegisterPrompts(s, kafkaInterface)

	// Log OAuth startup info if enabled
	if oauthServer != nil {
		oauthServer.LogStartup(false)
	}

	// Start server
	slog.Info("Starting Kafka MCP server", "version", Version, "transport", cfg.MCPTransport)
	if err := mcp.Start(ctx, s, cfg, mux); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}

	slog.Info("Server shutdown complete")
}
```

### Verification

Build the binary:

```bash
go build -o bin/kafka-mcp-server ./cmd/main.go
```

Expected: Binary created without errors

---

## Phase 5: Update Server Start Function

### File: `internal/mcp/server.go`

### Changes Required

#### Update Start Function Signature

Modify the `Start` function to accept the mux parameter:

```go
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
```

#### Replace startHTTPServer Function

Replace the "HTTP transport not yet implemented" placeholder:

```go
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
	slog.Info("Starting HTTP server",
		"address", addr,
		"oauth_enabled", cfg.OAuthEnabled,
		"mcp_endpoint", "/mcp")

	return http.ListenAndServe(addr, mux)
}
```

### Verification

Test the server in different modes:

#### Test STDIO Mode (Backwards Compatibility)
```bash
export MCP_TRANSPORT=stdio
go run cmd/main.go
```

Expected: Server starts in STDIO mode, no errors

#### Test HTTP Mode Without OAuth
```bash
export MCP_TRANSPORT=http
export MCP_HTTP_PORT=8080
go run cmd/main.go &
sleep 1
curl http://localhost:8080/mcp
kill %1
```

Expected: Server starts on port 8080, `/mcp` endpoint accessible

---

## Phase 6: Update Documentation

### File: `CLAUDE.md`

Add OAuth section after the existing configuration sections (after TLS configuration):

```markdown
### OAuth Configuration (HTTP Transport Only)

OAuth 2.1 authentication is available when using HTTP transport (`MCP_TRANSPORT=http`). Supports both native and proxy modes with multiple providers (Okta, Google, Azure AD, HMAC).

**Architecture**: OAuth option must be configured before server creation. The server validates Bearer tokens in the Authorization header and makes authenticated user information available to tools via `oauth.GetUserFromContext(ctx)`.

#### Environment Variables

**HTTP Server:**
- `MCP_HTTP_PORT` - HTTP server port (default: 8080)

**OAuth Settings:**
- `OAUTH_ENABLED` - Enable OAuth (default: false)
- `OAUTH_MODE` - "native" or "proxy" (default: native)
- `OAUTH_PROVIDER` - Provider: "hmac", "okta", "google", "azuread" (default: okta)
- `OAUTH_SERVER_URL` - Full server URL (e.g., https://localhost:8080)

**OIDC Configuration:**
- `OIDC_ISSUER` - OAuth issuer URL (required when OAuth enabled)
- `OIDC_CLIENT_ID` - OAuth client ID (proxy mode only)
- `OIDC_CLIENT_SECRET` - OAuth client secret (proxy mode only)
- `OIDC_AUDIENCE` - OAuth audience (required when OAuth enabled)

**Proxy Mode Only:**
- `OAUTH_REDIRECT_URIS` - Comma-separated redirect URIs
- `JWT_SECRET` - JWT signing secret (use strong random value)

#### Native Mode Example (Okta)

Native mode: Client handles OAuth flow, server validates tokens only.

```bash
export MCP_TRANSPORT=http
export MCP_HTTP_PORT=8080
export OAUTH_ENABLED=true
export OAUTH_MODE=native
export OAUTH_PROVIDER=okta
export OAUTH_SERVER_URL=https://localhost:8080
export OIDC_ISSUER=https://company.okta.com
export OIDC_AUDIENCE=api://kafka-mcp-server

# Kafka config
export KAFKA_BROKERS=localhost:9092

make run
```

#### Proxy Mode Example (Google)

Proxy mode: Server manages OAuth flow and token exchange.

```bash
export MCP_TRANSPORT=http
export MCP_HTTP_PORT=8080
export OAUTH_ENABLED=true
export OAUTH_MODE=proxy
export OAUTH_PROVIDER=google
export OAUTH_SERVER_URL=https://localhost:8080
export OIDC_ISSUER=https://accounts.google.com
export OIDC_CLIENT_ID=your-client-id.apps.googleusercontent.com
export OIDC_CLIENT_SECRET=your-client-secret
export OIDC_AUDIENCE=your-client-id.apps.googleusercontent.com
export OAUTH_REDIRECT_URIS=http://localhost:8080/oauth/callback
export JWT_SECRET=$(openssl rand -hex 32)

# Kafka config
export KAFKA_BROKERS=localhost:9092

make run
```

#### OAuth Endpoints

When OAuth is enabled, these endpoints are automatically registered:

- `/.well-known/oauth-authorization-server` - OAuth 2.1 metadata (RFC 8414)
- `/.well-known/openid-configuration` - OIDC discovery
- `/.well-known/oauth-protected-resource` - Protected resource metadata
- `/oauth/authorize` - Authorization endpoint (proxy mode)
- `/oauth/callback` - Callback endpoint (proxy mode)
- `/oauth/token` - Token endpoint (proxy mode)
- `/mcp` - MCP server endpoint (protected when OAuth enabled)

#### Testing OAuth with HMAC Provider

For local testing without external OAuth provider:

```bash
export MCP_TRANSPORT=http
export OAUTH_ENABLED=true
export OAUTH_PROVIDER=hmac
export OAUTH_MODE=native
export OAUTH_SERVER_URL=http://localhost:8080
export OIDC_ISSUER=http://localhost:8080
export OIDC_AUDIENCE=api://kafka-mcp-server
export JWT_SECRET=$(openssl rand -hex 32)

make run

# In another terminal, check metadata
curl http://localhost:8080/.well-known/oauth-authorization-server | jq
```

#### Troubleshooting

**Issue**: "invalid OAuth configuration" or "failed to setup OAuth"
- Verify all required fields for your mode are set (see examples above)
- Check OIDC_ISSUER and OIDC_AUDIENCE are valid URLs
- For proxy mode, ensure OIDC_CLIENT_ID, OIDC_CLIENT_SECRET, and JWT_SECRET are set

**Issue**: "mux is required when OAuth is enabled"
- This is an internal error; check that HTTP transport is properly configured

**Issue**: Token validation fails
- Verify token is sent in Authorization header: `Authorization: Bearer <token>`
- Check issuer and audience in token claims match configuration
- Confirm OAuth provider is accessible from the server
- Check server logs for detailed validation errors

**Issue**: Server starts but OAuth endpoints return 404
- Verify OAuth is enabled: `OAUTH_ENABLED=true`
- Check server logs for "OAuth configured" message
- Ensure you're using HTTP transport, not STDIO

#### Security Notes

- **TLS Required**: Always use TLS/HTTPS in production (handled at proxy/load balancer level)
- **Secrets Management**: Never commit JWT_SECRET or OIDC_CLIENT_SECRET to version control
- **Token Caching**: Library caches validated tokens for 5 minutes for performance
- **Rotation**: Rotate JWT secrets regularly in proxy mode
- **Logging**: OAuth tokens and secrets are never logged
```

---

## Phase 7: Unit Tests

### File: `internal/config/config_test.go`

Add comprehensive OAuth configuration tests:

```go
package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_HTTPPortDefault(t *testing.T) {
	os.Clearenv()
	cfg := LoadConfig()
	assert.Equal(t, 8080, cfg.HTTPPort)
}

func TestLoadConfig_HTTPPortCustom(t *testing.T) {
	os.Clearenv()
	os.Setenv("MCP_HTTP_PORT", "9090")
	defer os.Clearenv()

	cfg := LoadConfig()
	assert.Equal(t, 9090, cfg.HTTPPort)
}

func TestLoadConfig_HTTPPortInvalid(t *testing.T) {
	os.Clearenv()
	os.Setenv("MCP_HTTP_PORT", "invalid")
	defer os.Clearenv()

	cfg := LoadConfig()
	assert.Equal(t, 8080, cfg.HTTPPort) // Falls back to default
}

func TestLoadConfig_OAuthDefaults(t *testing.T) {
	os.Clearenv()
	cfg := LoadConfig()

	assert.False(t, cfg.OAuthEnabled)
	assert.Equal(t, "native", cfg.OAuthMode)
	assert.Equal(t, "okta", cfg.OAuthProvider)
	assert.Empty(t, cfg.OAuthServerURL)
	assert.Empty(t, cfg.OIDCIssuer)
	assert.Empty(t, cfg.OIDCClientID)
}

func TestLoadConfig_OAuthNativeMode(t *testing.T) {
	os.Clearenv()
	os.Setenv("OAUTH_ENABLED", "true")
	os.Setenv("OAUTH_MODE", "native")
	os.Setenv("OAUTH_PROVIDER", "okta")
	os.Setenv("OAUTH_SERVER_URL", "https://localhost:8080")
	os.Setenv("OIDC_ISSUER", "https://company.okta.com")
	os.Setenv("OIDC_AUDIENCE", "api://mcp-server")
	defer os.Clearenv()

	cfg := LoadConfig()

	assert.True(t, cfg.OAuthEnabled)
	assert.Equal(t, "native", cfg.OAuthMode)
	assert.Equal(t, "okta", cfg.OAuthProvider)
	assert.Equal(t, "https://localhost:8080", cfg.OAuthServerURL)
	assert.Equal(t, "https://company.okta.com", cfg.OIDCIssuer)
	assert.Equal(t, "api://mcp-server", cfg.OIDCAudience)
	assert.Empty(t, cfg.OIDCClientID)
	assert.Empty(t, cfg.OIDCClientSecret)
}

func TestLoadConfig_OAuthProxyMode(t *testing.T) {
	os.Clearenv()
	os.Setenv("OAUTH_ENABLED", "true")
	os.Setenv("OAUTH_MODE", "proxy")
	os.Setenv("OAUTH_PROVIDER", "google")
	os.Setenv("OAUTH_SERVER_URL", "https://localhost:8080")
	os.Setenv("OIDC_ISSUER", "https://accounts.google.com")
	os.Setenv("OIDC_CLIENT_ID", "client-id")
	os.Setenv("OIDC_CLIENT_SECRET", "client-secret")
	os.Setenv("OIDC_AUDIENCE", "api://mcp-server")
	os.Setenv("OAUTH_REDIRECT_URIS", "http://localhost:8080/callback,http://localhost:8080/callback2")
	os.Setenv("JWT_SECRET", "super-secret-key")
	defer os.Clearenv()

	cfg := LoadConfig()

	assert.Equal(t, "proxy", cfg.OAuthMode)
	assert.Equal(t, "google", cfg.OAuthProvider)
	assert.Equal(t, "client-id", cfg.OIDCClientID)
	assert.Equal(t, "client-secret", cfg.OIDCClientSecret)
	assert.Equal(t, "http://localhost:8080/callback,http://localhost:8080/callback2", cfg.OAuthRedirectURIs)
	assert.Equal(t, "super-secret-key", cfg.JWTSecret)
}

func TestLoadConfig_OAuthEnabledInvalid(t *testing.T) {
	os.Clearenv()
	os.Setenv("OAUTH_ENABLED", "not-a-bool")
	defer os.Clearenv()

	cfg := LoadConfig()
	assert.False(t, cfg.OAuthEnabled) // Falls back to default
}
```

Run tests:
```bash
go test ./internal/config/... -v -cover
```

Expected: All tests pass

---

## Phase 8: Integration Tests

### File: `internal/mcp/server_test.go`

Create HTTP server integration tests:

```go
package mcp_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tuannvm/kafka-mcp-server/internal/config"
	"github.com/tuannvm/kafka-mcp-server/internal/mcp"
)

func TestStartHTTPServerWithoutOAuth(t *testing.T) {
	cfg := config.Config{
		MCPTransport: "http",
		HTTPPort:     18080,
		OAuthEnabled: false,
	}

	mux := http.NewServeMux()
	s := mcp.NewMCPServer("test", "1.0.0")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := mcp.Start(ctx, s, cfg, mux)
		if err != nil && err != http.ErrServerClosed {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test MCP endpoint exists
	resp, err := http.Get("http://localhost:18080/mcp")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.NotNil(t, resp)

	cancel() // Shutdown
}

func TestCreateOAuthOption_Disabled(t *testing.T) {
	cfg := config.Config{
		OAuthEnabled: false,
	}

	mux := http.NewServeMux()
	option, server, err := mcp.CreateOAuthOption(cfg, mux)

	assert.NoError(t, err)
	assert.Nil(t, option)
	assert.Nil(t, server)
}

func TestCreateOAuthOption_NoMux(t *testing.T) {
	cfg := config.Config{
		OAuthEnabled: true,
	}

	option, server, err := mcp.CreateOAuthOption(cfg, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mux is required")
	assert.Nil(t, option)
	assert.Nil(t, server)
}
```

Run tests:
```bash
go test ./internal/mcp/... -v
```

---

## Phase 9: Manual Testing

### Test Checklist

#### 9.1 STDIO Mode (Backwards Compatibility)
- [ ] Start server with STDIO transport
- [ ] Verify no OAuth endpoints or errors
- [ ] Test basic MCP functionality

```bash
export MCP_TRANSPORT=stdio
export KAFKA_BROKERS=localhost:9092
go run cmd/main.go
```

Expected: Server starts successfully in STDIO mode

#### 9.2 HTTP Mode Without OAuth
- [ ] Start server with HTTP transport, OAuth disabled
- [ ] Access `/mcp` endpoint
- [ ] Verify no authentication required
- [ ] Check OAuth endpoints return 404

```bash
export MCP_TRANSPORT=http
export MCP_HTTP_PORT=8080
export KAFKA_BROKERS=localhost:9092
go run cmd/main.go &

# Test MCP endpoint
curl -v http://localhost:8080/mcp

# Verify no OAuth endpoints
curl -v http://localhost:8080/.well-known/oauth-authorization-server

kill %1
```

Expected: MCP endpoint accessible, OAuth endpoints not available

#### 9.3 HTTP Mode With Native OAuth (HMAC Provider)
- [ ] Configure HMAC provider for testing
- [ ] Start server
- [ ] Verify OAuth metadata endpoints exist
- [ ] Check server logs show "OAuth configured"

```bash
export MCP_TRANSPORT=http
export MCP_HTTP_PORT=8080
export OAUTH_ENABLED=true
export OAUTH_MODE=native
export OAUTH_PROVIDER=hmac
export OAUTH_SERVER_URL=http://localhost:8080
export OIDC_ISSUER=http://localhost:8080
export OIDC_AUDIENCE=api://kafka-mcp-server
export JWT_SECRET=$(openssl rand -hex 32)
export KAFKA_BROKERS=localhost:9092

go run cmd/main.go &

# Check OAuth metadata
curl http://localhost:8080/.well-known/oauth-authorization-server | jq

# Check OIDC discovery
curl http://localhost:8080/.well-known/openid-configuration | jq

kill %1
```

Expected:
- Server logs show "OAuth configured" with provider=hmac
- OAuth metadata endpoints return valid JSON
- MCP endpoint at `/mcp` is available

#### 9.4 HTTP Mode With Proxy OAuth (Google - Configuration Only)
- [ ] Configure all proxy mode fields
- [ ] Start server
- [ ] Verify server starts without errors
- [ ] Check proxy mode endpoints exist

```bash
export MCP_TRANSPORT=http
export MCP_HTTP_PORT=8080
export OAUTH_ENABLED=true
export OAUTH_MODE=proxy
export OAUTH_PROVIDER=google
export OAUTH_SERVER_URL=http://localhost:8080
export OIDC_ISSUER=https://accounts.google.com
export OIDC_CLIENT_ID=test-client-id
export OIDC_CLIENT_SECRET=test-client-secret
export OIDC_AUDIENCE=test-client-id
export OAUTH_REDIRECT_URIS=http://localhost:8080/oauth/callback
export JWT_SECRET=$(openssl rand -hex 32)
export KAFKA_BROKERS=localhost:9092

go run cmd/main.go &

# Check proxy mode endpoints
curl -v http://localhost:8080/oauth/authorize
curl -v http://localhost:8080/oauth/callback
curl -v http://localhost:8080/oauth/token

kill %1
```

Expected:
- Server starts successfully
- Logs show "OAuth configured" with provider=google, mode=proxy
- Proxy endpoints return responses (not 404)

---

## Phase 10: Security Review

### Security Checklist

#### Configuration Security
- [ ] JWT_SECRET uses strong random value (32+ bytes)
- [ ] Client secrets are not logged in application logs
- [ ] OAuth tokens are not logged
- [ ] Environment variables properly documented
- [ ] No secrets committed to git

Check:
```bash
# Verify secrets are not in code
grep -r "JWT_SECRET" --exclude-dir=.git --exclude="*.md" .
grep -r "CLIENT_SECRET" --exclude-dir=.git --exclude="*.md" .

# Check gitignore
cat .gitignore | grep -E "\\.env|\\.secret"
```

#### Runtime Security
- [ ] Token validation is working (test with invalid token)
- [ ] Invalid tokens are rejected with appropriate errors
- [ ] Expired tokens are rejected
- [ ] User context is properly extracted from valid tokens

#### Deployment Security
- [ ] TLS/HTTPS recommended in all documentation
- [ ] Proxy mode secrets rotation documented
- [ ] Provider-specific security notes added
- [ ] Production deployment checklist created

#### Code Review
- [ ] No hardcoded credentials
- [ ] Error messages don't leak sensitive information
- [ ] All OAuth errors are properly wrapped and logged
- [ ] User input validation in place

---

## Progress Notes

### 2025-01-23 - Implementation Guide Created
- Created comprehensive implementation guide
- Documented critical architectural requirement: OAuth option must be passed at server creation
- Provided detailed step-by-step instructions for all 10 phases
- Included verification steps and test cases

### Issues Encountered
_(Add notes as you implement)_

**Example format:**
```
2025-01-XX - Build Error in Phase 3
- Issue: Import path error for oauth-mcp-proxy
- Solution: Used correct import path with mark3labs subpackage
- Time spent: 15 minutes
```

### Decisions Made
_(Document key architectural decisions)_

**Example format:**
```
2025-01-XX - OAuth Option Architecture
- Decision: Refactor main.go to create OAuth option before NewMCPServer
- Rationale: Required by oauth-mcp-proxy@v1.0.0 API design
- Impact: Major refactor to main.go but cleaner separation of concerns
```

---

## Verification Commands

Quick verification after implementation:

```bash
# Check dependency installed
grep "oauth-mcp-proxy" go.mod

# Run all tests
make test-no-kafka

# Build binary
make build

# Verify binary works - STDIO mode
export MCP_TRANSPORT=stdio && ./bin/kafka-mcp-server &
sleep 1 && kill %1

# Verify binary works - HTTP mode
export MCP_TRANSPORT=http && export MCP_HTTP_PORT=8080 && ./bin/kafka-mcp-server &
sleep 1 && curl http://localhost:8080/mcp && kill %1
```

---

## Rollback Plan

If critical issues arise during implementation:

### Quick Rollback (Development)
```bash
# Revert all uncommitted changes
git checkout .
git clean -fd

# Verify tests still pass
make test-no-kafka
```

### Selective Rollback

1. **Revert go.mod changes**:
   ```bash
   git checkout go.mod go.sum
   go mod tidy
   ```

2. **Revert config changes**:
   ```bash
   git checkout internal/config/
   go test ./internal/config/...
   ```

3. **Revert server changes**:
   ```bash
   git checkout internal/mcp/server.go
   go build ./internal/mcp/...
   ```

4. **Revert main.go**:
   ```bash
   git checkout cmd/main.go
   go build -o bin/kafka-mcp-server ./cmd/main.go
   ```

5. **Verify STDIO still works**:
   ```bash
   export MCP_TRANSPORT=stdio
   ./bin/kafka-mcp-server
   ```

---

## Success Criteria

Implementation is complete and successful when:

### Functional Requirements
- [ ] All unit tests pass (`go test ./...`)
- [ ] Integration tests pass
- [ ] STDIO mode works (backwards compatibility verified)
- [ ] HTTP mode works without OAuth
- [ ] HTTP mode works with OAuth native mode (tested with HMAC provider)
- [ ] HTTP mode works with OAuth proxy mode (configuration verified)
- [ ] OAuth endpoints return valid responses
- [ ] Token validation works correctly

### Code Quality
- [ ] No compiler errors or warnings
- [ ] All imports are used
- [ ] Code follows Go conventions
- [ ] Error handling is comprehensive
- [ ] Logging is appropriate (no secrets logged)

### Documentation
- [ ] CLAUDE.md updated with OAuth configuration
- [ ] All environment variables documented
- [ ] Examples provided for both OAuth modes
- [ ] Troubleshooting guide complete

### Security
- [ ] Security review completed
- [ ] No credentials in code or logs
- [ ] TLS requirements documented
- [ ] Secrets management documented

### Testing
- [ ] Manual testing checklist 100% complete
- [ ] All test scenarios pass
- [ ] Edge cases tested (invalid tokens, missing config, etc.)

---

## Next Steps After Implementation

1. **Create Pull Request**
   - Ensure all tests pass
   - Update CHANGELOG.md
   - Request code review

2. **Staging Deployment**
   - Deploy to staging environment
   - Test with real OAuth provider (Okta or Google)
   - Verify token validation with actual tokens

3. **Documentation**
   - Add provider-specific setup guides (Okta, Google, Azure AD)
   - Create deployment runbook
   - Document monitoring and troubleshooting procedures

4. **Production Preparation**
   - Set up secrets management (e.g., Vault, AWS Secrets Manager)
   - Configure TLS termination at load balancer
   - Set up monitoring and alerting
   - Create rollback procedures

---

## Support and Resources

- **oauth-mcp-proxy docs**: https://pkg.go.dev/github.com/tuannvm/oauth-mcp-proxy@v1.0.0
- **mcp-go docs**: https://pkg.go.dev/github.com/mark3labs/mcp-go@v0.41.1
- **OAuth 2.1 spec**: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-12
- **Project issues**: https://github.com/tuannvm/kafka-mcp-server/issues
