package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Config holds the application configuration.
type Config struct {
	KafkaBrokers []string // List of Kafka broker addresses
	KafkaClientID string   // Kafka client ID
	MCPTransport string   // MCP transport method ("stdio" or "http")

	// SASL Configuration
	SASLMechanism string // "plain", "scram-sha-256", "scram-sha-512", or "" (disabled)
	SASLUser      string
	SASLPassword  string

	// TLS Configuration
	TLSEnable            bool // Whether to enable TLS
	TLSInsecureSkipVerify bool // Whether to skip TLS certificate verification (use with caution!)
	// TODO: Add paths for CA cert, client cert, client key if needed

	// HTTP Server Configuration
	HTTPPort int // HTTP server port (default: 8080)

	// OAuth Configuration
	OAuthEnabled    bool
	OAuthMode       string // "native" or "proxy"
	OAuthProvider   string // "hmac", "okta", "google", "azure"
	OAuthServerURL  string // Base URL for the MCP server

	// OIDC Configuration
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCAudience     string

	// Proxy Mode Configuration
	OAuthRedirectURIs string // Comma-separated redirect URIs
	JWTSecret         string // Will be converted to []byte for oauth library
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() Config {
	brokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	clientID := getEnv("KAFKA_CLIENT_ID", "kafka-mcp-server")
	mcpTransport := getEnv("MCP_TRANSPORT", "stdio")

	// SASL Env Vars
	saslMechanism := strings.ToLower(getEnv("KAFKA_SASL_MECHANISM", ""))
	saslUser := getEnv("KAFKA_SASL_USER", "")
	saslPassword := getEnv("KAFKA_SASL_PASSWORD", "")

	// TLS Env Vars
	tlsEnableStr := getEnv("KAFKA_TLS_ENABLE", "false")
	tlsInsecureSkipVerifyStr := getEnv("KAFKA_TLS_INSECURE_SKIP_VERIFY", "false")

	tlsEnable, _ := strconv.ParseBool(tlsEnableStr)
	tlsInsecureSkipVerify, _ := strconv.ParseBool(tlsInsecureSkipVerifyStr)

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
		KafkaBrokers: strings.Split(brokers, ","),
		KafkaClientID: clientID,
		MCPTransport: mcpTransport,

		SASLMechanism: saslMechanism,
		SASLUser:      saslUser,
		SASLPassword:  saslPassword,

		TLSEnable:            tlsEnable,
		TLSInsecureSkipVerify: tlsInsecureSkipVerify,

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

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
