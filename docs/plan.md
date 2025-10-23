# OAuth 2.1 Implementation Plan

## Overview

Integrate OAuth 2.1 authentication for kafka-mcp-server using the [oauth-mcp-proxy](https://github.com/tuannvm/oauth-mcp-proxy) library. This implementation follows the pattern established in [mcp-trino](https://github.com/tuannvm/mcp-trino).

## Architecture

### OAuth Modes

**Native Mode** (Zero server-side secrets)

- Clients handle OAuth flow directly
- Server validates Bearer tokens via OAuth provider
- No client secrets stored on server
- Most secure option

**Proxy Mode** (Centralized management)

- Server manages OAuth flow and token exchange
- Requires client ID, client secret, and JWT secret
- Supports redirect URI configuration
- Useful for centralized control

### Transport Requirement

OAuth authentication requires HTTP transport (Bearer tokens in headers). STDIO transport will remain available for local/trusted environments without authentication.

## Implementation Phases

### Phase 1: Dependencies

- Add `github.com/tuannvm/oauth-mcp-proxy@v1.0.0` to go.mod
- Run `go mod tidy` to fetch dependencies

### Phase 2: Configuration

Update `internal/config/config.go`:

```go
type Config struct {
    // ... existing fields ...

    // HTTP Server Configuration
    HTTPPort int // HTTP server port (default: 8080)

    // OAuth Configuration
    OAuthEnabled    bool
    OAuthMode       string // "native" or "proxy"
    OAuthProvider   string // "hmac", "okta", "google", "azuread"
    OAuthServerURL  string // Base URL for the MCP server

    // OIDC Configuration
    OIDCIssuer      string
    OIDCClientID    string
    OIDCClientSecret string
    OIDCAudience    string

    // Proxy Mode Configuration
    OAuthRedirectURIs string // Comma-separated redirect URIs
    JWTSecret         string // Will be converted to []byte for oauth library
}
```

Environment variables:

- `MCP_HTTP_PORT` - HTTP server port (default: 8080)
- `OAUTH_ENABLED` - Enable OAuth (default: false)
- `OAUTH_MODE` - "native" or "proxy" (default: native)
- `OAUTH_PROVIDER` - Provider type (default: okta)
- `OAUTH_SERVER_URL` - Server base URL
- `OIDC_ISSUER` - OAuth issuer URL
- `OIDC_CLIENT_ID` - OAuth client ID (proxy mode only)
- `OIDC_CLIENT_SECRET` - OAuth client secret (proxy mode only)
- `OIDC_AUDIENCE` - OAuth audience
- `OAUTH_REDIRECT_URIS` - Comma-separated redirect URIs (proxy mode only)
- `JWT_SECRET` - JWT signing secret (proxy mode only)

### Phase 3: Add OAuth Helper Function

Add to `internal/mcp/server.go`:

```go
import (
    oauth "github.com/tuannvm/oauth-mcp-proxy"
    "github.com/tuannvm/oauth-mcp-proxy/mark3labs"
    "github.com/mark3labs/mcp-go/server"
)

// CreateOAuthOption creates OAuth server option if OAuth is enabled
func CreateOAuthOption(cfg config.Config, mux *http.ServeMux) (server.ServerOption, *oauth.Server, error) {
    if !cfg.OAuthEnabled {
        return nil, nil, nil
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

    slog.Info("OAuth configured", "mode", cfg.OAuthMode, "provider", cfg.OAuthProvider)
    return oauthOption, oauthServer, nil
}
```

### Phase 4: Update Main Entry Point

Update `cmd/main.go` to create OAuth option BEFORE server creation:

```go
func main() {
    // ... existing setup code ...

    cfg := config.LoadConfig()

    // Initialize Kafka client
    kafkaClient, err := kafka.NewClient(cfg)
    if err != nil {
        slog.Error("Failed to create Kafka client", "error", err)
        os.Exit(1)
    }
    defer kafkaClient.Close()

    // Create HTTP mux if using HTTP transport
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

    // Register MCP resources and tools
    var kafkaInterface kafka.KafkaClient = kafkaClient
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

### Phase 5: Update Server Start Function

Update `internal/mcp/server.go` Start function:

```go
func Start(ctx context.Context, s *server.MCPServer, cfg config.Config, mux *http.ServeMux) error {
    switch cfg.MCPTransport {
    case "stdio":
        return server.ServeStdio(s)
    case "http":
        return startHTTPServer(ctx, s, cfg, mux)
    default:
        return fmt.Errorf("unsupported transport: %s", cfg.MCPTransport)
    }
}

func startHTTPServer(ctx context.Context, s *server.MCPServer, cfg config.Config, mux *http.ServeMux) error {
    streamable := server.NewStreamableHTTPServer(
        s,
        server.WithHTTPContextFunc(oauth.CreateHTTPContextFunc()),
    )
    mux.Handle("/mcp", streamable)

    addr := fmt.Sprintf(":%d", cfg.HTTPPort)
    slog.Info("Starting HTTP server", "address", addr, "oauth_enabled", cfg.OAuthEnabled)
    return http.ListenAndServe(addr, mux)
}
```

**Critical Notes**:
- OAuth option MUST be passed to `NewMCPServer()` at creation time
- Mux must be created BEFORE calling CreateOAuthOption (WithOAuth registers routes on it)
- CreateHTTPContextFunc() extracts Bearer tokens from Authorization header
- OAuth endpoints automatically registered when CreateOAuthOption is called

### Phase 6: Documentation Updates

Update `CLAUDE.md`:

- Add OAuth configuration section
- Document environment variables
- Provide examples for both modes
- Add troubleshooting guide

## Code Changes Summary

### Files to Modify

1. `internal/config/config.go` - Add OAuth and HTTP port configuration fields and parsing
2. `internal/mcp/server.go` - Implement HTTP transport with OAuth middleware
3. `CLAUDE.md` - Document OAuth configuration and usage
4. `go.mod` - Add oauth-mcp-proxy dependency

### Files to Create

- None (optional: example configs for different providers)

## Configuration Examples

### Native Mode (Okta)

```bash
export OAUTH_ENABLED=true
export OAUTH_MODE=native
export OAUTH_PROVIDER=okta
export OAUTH_SERVER_URL=https://localhost:8080
export OIDC_ISSUER=https://company.okta.com
export OIDC_AUDIENCE=https://mcp-server.company.com
export MCP_TRANSPORT=http
```

### Proxy Mode (Google)

```bash
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
export MCP_TRANSPORT=http
```

## Testing Strategy

### Unit Tests

- Config parsing for all OAuth parameters
- OAuth middleware initialization
- Error handling for invalid configurations

### Integration Tests

- HTTP server startup with OAuth enabled/disabled
- Token validation (using test tokens)
- Both native and proxy mode flows

### Manual Testing

1. STDIO mode without OAuth (backwards compatibility)
2. HTTP mode without OAuth
3. HTTP mode with native OAuth (Okta)
4. HTTP mode with proxy OAuth (Google)
5. Invalid token rejection
6. User context extraction from valid tokens

## Deployment Considerations

### Backwards Compatibility

- STDIO transport remains default
- OAuth disabled by default
- Existing configurations continue to work

### Security Notes

- Never log OAuth secrets or tokens
- Use TLS in production (HTTPS)
- Rotate JWT secrets regularly in proxy mode
- Validate issuer URLs match expected providers

### Migration Path

1. Add OAuth configuration (disabled)
2. Test HTTP transport without OAuth
3. Configure OAuth provider
4. Enable OAuth in staging
5. Test client integration
6. Roll out to production

## Dependencies

### Direct Dependencies

- `github.com/tuannvm/oauth-mcp-proxy@v1.0.0` - OAuth middleware
- `github.com/mark3labs/mcp-go@v0.41.1` - MCP framework (already present)

### OAuth Provider Requirements

- **Okta**: Okta account, OAuth application configured
- **Google**: Google Cloud project, OAuth 2.0 credentials
- **Azure AD**: Azure AD tenant, app registration
- **HMAC**: Shared secret for development/testing

## Success Criteria

- [ ] OAuth dependency integrated
- [ ] Config supports all OAuth parameters
- [ ] HTTP transport implemented with SSE
- [ ] OAuth middleware integrated for both modes
- [ ] Documentation updated
- [ ] Unit tests pass
- [ ] Integration tests for both modes
- [ ] Manual testing with real OAuth provider
- [ ] Backwards compatibility verified
- [ ] Security review completed
