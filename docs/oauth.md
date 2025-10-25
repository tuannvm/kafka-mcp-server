# OAuth 2.1 Authentication for Kafka MCP Server

## Overview

Kafka MCP Server supports OAuth 2.1 authentication when running in HTTP transport mode. This enables secure, token-based authentication for MCP clients accessing the server over HTTP.

The implementation uses [oauth-mcp-proxy@v1.0.0](https://github.com/tuannvm/oauth-mcp-proxy), a standalone OAuth 2.1 library designed specifically for Go MCP servers.

## Features

- **Two OAuth Modes**: Native mode (client-managed) and Proxy mode (server-managed)
- **Multiple Providers**: Support for HMAC, Okta, Google, and Azure AD
- **Token Caching**: 5-minute token validation cache for performance
- **Bearer Token Authentication**: Standard Authorization header support
- **Automatic Endpoint Registration**: OAuth routes automatically configured
- **Graceful Shutdown**: Proper HTTP server lifecycle management

## Architecture

### OAuth Flow

```
┌─────────────────┐
│   MCP Client    │
│ (Cursor, etc)   │
└────────┬────────┘
         │
         │ Bearer Token
         │ Authorization: Bearer <token>
         ▼
┌─────────────────────────────────┐
│   HTTP Server                   │
│                                 │
│  ┌──────────────────────────┐  │
│  │   OAuth Middleware       │  │
│  │  - Token Extraction      │  │
│  │  - Token Validation      │  │
│  │  - User Context Inject   │  │
│  └──────────┬───────────────┘  │
│             ▼                   │
│  ┌──────────────────────────┐  │
│  │   MCP Server Handler     │  │
│  │   /mcp endpoint          │  │
│  └──────────────────────────┘  │
└─────────────────────────────────┘
         │
         ▼
┌─────────────────┐
│  Kafka Cluster  │
└─────────────────┘
```

### Implementation Architecture

The OAuth integration follows a specific initialization sequence:

1. **Mux Creation**: `http.NewServeMux()` created in `main()`
2. **OAuth Registration**: `CreateOAuthOption(cfg, mux)` registers OAuth routes on mux
3. **Server Creation**: `NewMCPServer(name, version, oauthOption)` with OAuth middleware
4. **HTTP Server**: `startHTTPServer()` mounts MCP handler and starts server

This ensures OAuth routes and middleware are properly configured before the server starts handling requests.

## OAuth Modes

### Native Mode (Client-Managed)

**Best for**: Environments where clients can handle OAuth flows directly.

**Characteristics**:
- Zero server-side secrets
- Clients obtain tokens from OAuth provider
- Server validates Bearer tokens only
- Most secure option

**Configuration**:
```bash
export MCP_TRANSPORT=http
export MCP_HTTP_PORT=8080
export OAUTH_ENABLED=true
export OAUTH_MODE=native
export OAUTH_PROVIDER=okta
export OAUTH_SERVER_URL=http://localhost:8080
export OIDC_ISSUER=https://company.okta.com
export OIDC_AUDIENCE=api://kafka-mcp-server
```

### Proxy Mode (Server-Managed)

**Best for**: Centralized OAuth management, simple clients.

**Characteristics**:
- Server manages OAuth flow
- Requires client ID, client secret, JWT secret
- Server handles token exchange
- Useful for centralized control

**Configuration**:
```bash
export MCP_TRANSPORT=http
export MCP_HTTP_PORT=8080
export OAUTH_ENABLED=true
export OAUTH_MODE=proxy
export OAUTH_PROVIDER=google
export OAUTH_SERVER_URL=http://localhost:8080
export OIDC_ISSUER=https://accounts.google.com
export OIDC_CLIENT_ID=your-client-id.apps.googleusercontent.com
export OIDC_CLIENT_SECRET=your-client-secret
export OIDC_AUDIENCE=your-client-id.apps.googleusercontent.com
export OAUTH_REDIRECT_URIS=http://localhost:8080/oauth/callback
export JWT_SECRET=$(openssl rand -hex 32)
```

## Supported Providers

### 1. HMAC (Development/Testing)

Simple symmetric key authentication for local development.

```bash
export OAUTH_PROVIDER=hmac
export OAUTH_MODE=native
export JWT_SECRET=$(openssl rand -hex 32)
export OIDC_ISSUER=http://localhost:8080
export OIDC_AUDIENCE=api://kafka-mcp-server
```

**Use case**: Local testing without external OAuth provider.

### 2. Okta

Enterprise SSO provider.

**Setup Requirements**:
- Okta account and tenant
- OAuth application configured in Okta
- API audience defined

**Native Mode**:
```bash
export OAUTH_PROVIDER=okta
export OAUTH_MODE=native
export OIDC_ISSUER=https://company.okta.com
export OIDC_AUDIENCE=api://kafka-mcp-server
```

**Proxy Mode**: Add client credentials
```bash
export OIDC_CLIENT_ID=your-okta-client-id
export OIDC_CLIENT_SECRET=your-okta-client-secret
export OAUTH_REDIRECT_URIS=http://localhost:8080/oauth/callback
export JWT_SECRET=$(openssl rand -hex 32)
```

### 3. Google

Google Workspace authentication.

**Setup Requirements**:
- Google Cloud project
- OAuth 2.0 credentials configured
- Authorized redirect URIs

**Configuration**:
```bash
export OAUTH_PROVIDER=google
export OIDC_ISSUER=https://accounts.google.com
export OIDC_AUDIENCE=your-client-id.apps.googleusercontent.com
```

### 4. Azure AD (azure)

Microsoft identity platform.

**Setup Requirements**:
- Azure AD tenant
- App registration in Azure portal
- API permissions configured

**Configuration**:
```bash
export OAUTH_PROVIDER=azure
export OIDC_ISSUER=https://login.microsoftonline.com/{tenant-id}/v2.0
export OIDC_AUDIENCE=api://your-app-id
```

## Environment Variables

### HTTP Server Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `MCP_TRANSPORT` | Transport mode | `stdio` | Yes (set to `http`) |
| `MCP_HTTP_PORT` | HTTP server port | `8080` | No |

### OAuth Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `OAUTH_ENABLED` | Enable OAuth | `false` | Yes |
| `OAUTH_MODE` | Mode: native or proxy | `native` | No |
| `OAUTH_PROVIDER` | Provider: hmac, okta, google, azuread | `okta` | No |
| `OAUTH_SERVER_URL` | Full server URL | - | Yes |

### OIDC Configuration

| Variable | Description | Required |
|----------|-------------|----------|
| `OIDC_ISSUER` | OAuth issuer URL | Yes |
| `OIDC_AUDIENCE` | OAuth audience | Yes |
| `OIDC_CLIENT_ID` | OAuth client ID | Proxy mode only |
| `OIDC_CLIENT_SECRET` | OAuth client secret | Proxy mode only |

### Proxy Mode Only

| Variable | Description | Required |
|----------|-------------|----------|
| `OAUTH_REDIRECT_URIS` | Comma-separated redirect URIs | Yes |
| `JWT_SECRET` | JWT signing secret | Yes |

## OAuth Endpoints

When OAuth is enabled, the following endpoints are automatically registered:

### Discovery Endpoints

- `/.well-known/oauth-authorization-server` - OAuth 2.1 metadata (RFC 8414)
- `/.well-known/openid-configuration` - OIDC discovery
- `/.well-known/oauth-protected-resource` - Protected resource metadata
- `/.well-known/jwks.json` - JSON Web Key Set

### Proxy Mode Endpoints

- `/oauth/authorize` - Authorization endpoint
- `/oauth/callback` - OAuth callback endpoint
- `/oauth/token` - Token exchange endpoint
- `/oauth/register` - Dynamic client registration

### MCP Endpoint

- `/mcp` - MCP server endpoint (protected when OAuth enabled)

## Testing OAuth

### Local Testing with HMAC Provider

The HMAC provider is perfect for local development without external dependencies:

```bash
# Start server with HMAC OAuth
export MCP_TRANSPORT=http
export MCP_HTTP_PORT=8080
export OAUTH_ENABLED=true
export OAUTH_PROVIDER=hmac
export OAUTH_MODE=native
export OAUTH_SERVER_URL=http://localhost:8080
export OIDC_ISSUER=http://localhost:8080
export OIDC_AUDIENCE=api://kafka-mcp-server
export JWT_SECRET=$(openssl rand -hex 32)
export KAFKA_BROKERS=localhost:9092

./bin/kafka-mcp-server
```

**Check OAuth metadata**:
```bash
curl http://localhost:8080/.well-known/oauth-authorization-server | jq
```

**Expected output**:
```json
{
  "issuer": "http://localhost:8080",
  "authorization_endpoint": "http://localhost:8080/oauth/authorize",
  "token_endpoint": "http://localhost:8080/oauth/token",
  "jwks_uri": "http://localhost:8080/.well-known/jwks.json",
  ...
}
```

### Testing with Token

```bash
# Generate test token (implementation-specific)
TOKEN="your-bearer-token"

# Access MCP endpoint with token
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/mcp
```

## Security Considerations

### Production Requirements

1. **TLS/HTTPS Required**: Always use TLS in production
   - Handle TLS at proxy/load balancer level
   - Set `OAUTH_SERVER_URL` to HTTPS URL

2. **Secrets Management**: Never commit secrets to version control
   - Use secret management systems (Vault, AWS Secrets Manager)
   - Rotate JWT secrets regularly
   - Store client secrets securely

3. **Token Security**:
   - Library caches validated tokens for 5 minutes
   - Tokens must be sent in Authorization header
   - Invalid/expired tokens are rejected automatically

4. **Logging**: OAuth tokens and secrets are never logged

### Environment-Specific Recommendations

**Development**:
- Use HMAC provider for simplicity
- `OAUTH_SERVER_URL=http://localhost:8080` is acceptable
- Generate strong JWT secrets: `openssl rand -hex 32`

**Staging**:
- Use real OAuth provider (Okta, Google, Azure AD)
- Configure TLS termination
- Test with actual tokens
- Validate issuer and audience matching

**Production**:
- TLS/HTTPS mandatory
- Strong, rotated JWT secrets
- Monitor token validation errors
- Set up alerts for authentication failures
- Use native mode when possible (no server-side secrets)

## Troubleshooting

### Server Fails to Start

**Issue**: `failed to setup OAuth`

**Solutions**:
1. Verify all required environment variables are set
2. Check OIDC_ISSUER is a valid URL
3. For proxy mode, ensure CLIENT_ID, CLIENT_SECRET, and JWT_SECRET are set
4. Review server logs for specific error messages

### OAuth Endpoints Return 404

**Issue**: OAuth discovery endpoints not found

**Solutions**:
1. Verify `OAUTH_ENABLED=true`
2. Confirm `MCP_TRANSPORT=http`
3. Check server logs for "OAuth configured" message
4. Ensure server started successfully

### Token Validation Fails

**Issue**: Valid tokens are rejected

**Solutions**:
1. Verify token is sent in Authorization header: `Authorization: Bearer <token>`
2. Check issuer in token claims matches `OIDC_ISSUER`
3. Verify audience in token claims matches `OIDC_AUDIENCE`
4. Ensure OAuth provider is accessible from server
5. Check token has not expired
6. Review server logs for validation error details

### Graceful Shutdown Issues

**Issue**: Server doesn't shutdown cleanly

**Solutions**:
1. Ensure server receives SIGINT/SIGTERM signal
2. Check for context cancellation in logs
3. Default shutdown timeout is 5 seconds
4. Review server logs for shutdown errors

## Code Examples

### Accessing User Context in Tools

When OAuth is enabled, authenticated user information is available in tool handlers:

```go
import (
    oauth "github.com/tuannvm/oauth-mcp-proxy"
)

func myToolHandler(ctx context.Context, request ToolRequest) (*ToolResponse, error) {
    // Extract authenticated user from context
    user, ok := oauth.GetUserFromContext(ctx)
    if !ok {
        return nil, fmt.Errorf("authentication required")
    }

    slog.Info("Tool accessed by authenticated user",
        "username", user.Username,
        "email", user.Email,
        "subject", user.Subject)

    // Proceed with tool logic
    // ...
}
```

### Custom OAuth Configuration

For advanced use cases, you can customize OAuth behavior by modifying `internal/mcp/server.go`:

```go
// Example: Add custom logger
oauthConfig.Logger = customLogger

// Example: Customize provider-specific settings
if cfg.OAuthProvider == "okta" {
    // Provider-specific configuration
}
```

## Migration Guide

### From STDIO to HTTP with OAuth

**Step 1**: Ensure backwards compatibility
```bash
# Existing STDIO configuration still works
export MCP_TRANSPORT=stdio
export KAFKA_BROKERS=localhost:9092
```

**Step 2**: Add HTTP transport without OAuth
```bash
export MCP_TRANSPORT=http
export MCP_HTTP_PORT=8080
export KAFKA_BROKERS=localhost:9092
```

**Step 3**: Enable OAuth
```bash
export MCP_TRANSPORT=http
export MCP_HTTP_PORT=8080
export OAUTH_ENABLED=true
export OAUTH_MODE=native
export OAUTH_PROVIDER=okta
export OAUTH_SERVER_URL=https://your-domain.com
export OIDC_ISSUER=https://company.okta.com
export OIDC_AUDIENCE=api://kafka-mcp-server
export KAFKA_BROKERS=localhost:9092
```

**Step 4**: Update MCP clients
- Configure clients to send Bearer tokens
- Update connection URL to HTTP endpoint
- Test token validation

## References

- [OAuth 2.1 Specification](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-12)
- [oauth-mcp-proxy Documentation](https://pkg.go.dev/github.com/tuannvm/oauth-mcp-proxy@v1.0.0)
- [RFC 8414: OAuth 2.0 Authorization Server Metadata](https://datatracker.ietf.org/doc/html/rfc8414)
- [OpenID Connect Discovery](https://openid.net/specs/openid-connect-discovery-1_0.html)

## Support

For OAuth-related issues:
1. Check server logs for detailed error messages
2. Review oauth-mcp-proxy documentation
3. Consult provider-specific setup guides (Okta, Google, Azure AD)
4. Open an issue on GitHub with relevant logs (redact secrets!)
