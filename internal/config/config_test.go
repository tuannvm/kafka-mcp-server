package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Set environment variables for testing
	originalBrokers := os.Getenv("KAFKA_BROKERS")
	originalClientID := os.Getenv("KAFKA_CLIENT_ID")
	originalTransport := os.Getenv("MCP_TRANSPORT")
	originalSASLMech := os.Getenv("KAFKA_SASL_MECHANISM")
	originalSASLUser := os.Getenv("KAFKA_SASL_USER")
	originalSASLPass := os.Getenv("KAFKA_SASL_PASSWORD")
	originalTLSEnable := os.Getenv("KAFKA_TLS_ENABLE")
	originalTLSInsecure := os.Getenv("KAFKA_TLS_INSECURE_SKIP_VERIFY")

	defer func() {
		// Restore original environment variables
		if err := os.Setenv("KAFKA_BROKERS", originalBrokers); err != nil {
			t.Logf("Failed to restore KAFKA_BROKERS: %v", err)
		}
		if err := os.Setenv("KAFKA_CLIENT_ID", originalClientID); err != nil {
			t.Logf("Failed to restore KAFKA_CLIENT_ID: %v", err)
		}
		if err := os.Setenv("MCP_TRANSPORT", originalTransport); err != nil {
			t.Logf("Failed to restore MCP_TRANSPORT: %v", err)
		}
		if err := os.Setenv("KAFKA_SASL_MECHANISM", originalSASLMech); err != nil {
			t.Logf("Failed to restore KAFKA_SASL_MECHANISM: %v", err)
		}
		if err := os.Setenv("KAFKA_SASL_USER", originalSASLUser); err != nil {
			t.Logf("Failed to restore KAFKA_SASL_USER: %v", err)
		}
		if err := os.Setenv("KAFKA_SASL_PASSWORD", originalSASLPass); err != nil {
			t.Logf("Failed to restore KAFKA_SASL_PASSWORD: %v", err)
		}
		if err := os.Setenv("KAFKA_TLS_ENABLE", originalTLSEnable); err != nil {
			t.Logf("Failed to restore KAFKA_TLS_ENABLE: %v", err)
		}
		if err := os.Setenv("KAFKA_TLS_INSECURE_SKIP_VERIFY", originalTLSInsecure); err != nil {
			t.Logf("Failed to restore KAFKA_TLS_INSECURE_SKIP_VERIFY: %v", err)
		}
	}()

	if err := os.Setenv("KAFKA_BROKERS", "test-broker1:9092,test-broker2:9092"); err != nil {
		t.Fatalf("Failed to set KAFKA_BROKERS: %v", err)
	}
	if err := os.Setenv("KAFKA_CLIENT_ID", "test-client"); err != nil {
		t.Fatalf("Failed to set KAFKA_CLIENT_ID: %v", err)
	}
	if err := os.Setenv("MCP_TRANSPORT", "stdio"); err != nil {
		t.Fatalf("Failed to set MCP_TRANSPORT: %v", err)
	}
	if err := os.Setenv("KAFKA_SASL_MECHANISM", "plain"); err != nil {
		t.Fatalf("Failed to set KAFKA_SASL_MECHANISM: %v", err)
	}
	if err := os.Setenv("KAFKA_SASL_USER", "testuser"); err != nil {
		t.Fatalf("Failed to set KAFKA_SASL_USER: %v", err)
	}
	if err := os.Setenv("KAFKA_SASL_PASSWORD", "testpass"); err != nil {
		t.Fatalf("Failed to set KAFKA_SASL_PASSWORD: %v", err)
	}
	if err := os.Setenv("KAFKA_TLS_ENABLE", "true"); err != nil {
		t.Fatalf("Failed to set KAFKA_TLS_ENABLE: %v", err)
	}
	if err := os.Setenv("KAFKA_TLS_INSECURE_SKIP_VERIFY", "true"); err != nil {
		t.Fatalf("Failed to set KAFKA_TLS_INSECURE_SKIP_VERIFY: %v", err)
	}

	cfg := LoadConfig()

	if len(cfg.KafkaBrokers) != 2 || cfg.KafkaBrokers[0] != "test-broker1:9092" || cfg.KafkaBrokers[1] != "test-broker2:9092" {
		t.Errorf("Expected KafkaBrokers [test-broker1:9092 test-broker2:9092], got %v", cfg.KafkaBrokers)
	}
	if cfg.KafkaClientID != "test-client" {
		t.Errorf("Expected KafkaClientID test-client, got %s", cfg.KafkaClientID)
	}
	if cfg.MCPTransport != "stdio" {
		t.Errorf("Expected MCPTransport stdio, got %s", cfg.MCPTransport)
	}
	if cfg.SASLMechanism != "plain" {
		t.Errorf("Expected SASLMechanism plain, got %s", cfg.SASLMechanism)
	}
	if cfg.SASLUser != "testuser" {
		t.Errorf("Expected SASLUser testuser, got %s", cfg.SASLUser)
	}
	if cfg.SASLPassword != "testpass" {
		t.Errorf("Expected SASLPassword testpass, got %s", cfg.SASLPassword)
	}
	if !cfg.TLSEnable {
		t.Errorf("Expected TLSEnable true, got %v", cfg.TLSEnable)
	}
	if !cfg.TLSInsecureSkipVerify {
		t.Errorf("Expected TLSInsecureSkipVerify true, got %v", cfg.TLSInsecureSkipVerify)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// Clear environment variables to test defaults
	if err := os.Unsetenv("KAFKA_BROKERS"); err != nil {
		t.Logf("Failed to unset KAFKA_BROKERS: %v", err)
	}
	if err := os.Unsetenv("KAFKA_CLIENT_ID"); err != nil {
		t.Logf("Failed to unset KAFKA_CLIENT_ID: %v", err)
	}
	if err := os.Unsetenv("MCP_TRANSPORT"); err != nil {
		t.Logf("Failed to unset MCP_TRANSPORT: %v", err)
	}
	if err := os.Unsetenv("KAFKA_SASL_MECHANISM"); err != nil {
		t.Logf("Failed to unset KAFKA_SASL_MECHANISM: %v", err)
	}
	if err := os.Unsetenv("KAFKA_TLS_ENABLE"); err != nil {
		t.Logf("Failed to unset KAFKA_TLS_ENABLE: %v", err)
	}
	if err := os.Unsetenv("KAFKA_TLS_INSECURE_SKIP_VERIFY"); err != nil {
		t.Logf("Failed to unset KAFKA_TLS_INSECURE_SKIP_VERIFY: %v", err)
	}

	cfg := LoadConfig()

	if len(cfg.KafkaBrokers) != 1 || cfg.KafkaBrokers[0] != "localhost:9092" {
		t.Errorf("Expected default KafkaBrokers [localhost:9092], got %v", cfg.KafkaBrokers)
	}
	if cfg.KafkaClientID != "kafka-mcp-server" {
		t.Errorf("Expected default KafkaClientID kafka-mcp-server, got %s", cfg.KafkaClientID)
	}
	if cfg.MCPTransport != "stdio" {
		t.Errorf("Expected default MCPTransport stdio, got %s", cfg.MCPTransport)
	}
	if cfg.SASLMechanism != "" {
		t.Errorf("Expected default SASLMechanism \"\", got %s", cfg.SASLMechanism)
	}
	if cfg.TLSEnable {
		t.Errorf("Expected default TLSEnable false, got %v", cfg.TLSEnable)
	}
	if cfg.TLSInsecureSkipVerify {
		t.Errorf("Expected default TLSInsecureSkipVerify false, got %v", cfg.TLSInsecureSkipVerify)
	}
	if cfg.HTTPPort != 8080 {
		t.Errorf("Expected default HTTPPort 8080, got %d", cfg.HTTPPort)
	}
	if cfg.OAuthEnabled {
		t.Errorf("Expected default OAuthEnabled false, got %v", cfg.OAuthEnabled)
	}
	if cfg.OAuthMode != "native" {
		t.Errorf("Expected default OAuthMode native, got %s", cfg.OAuthMode)
	}
	if cfg.OAuthProvider != "okta" {
		t.Errorf("Expected default OAuthProvider okta, got %s", cfg.OAuthProvider)
	}
}

func TestLoadConfig_OAuthNativeMode(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("OAUTH_ENABLED", "true")
	_ = os.Setenv("OAUTH_MODE", "native")
	_ = os.Setenv("OAUTH_PROVIDER", "okta")
	_ = os.Setenv("OAUTH_SERVER_URL", "https://localhost:8080")
	_ = os.Setenv("OIDC_ISSUER", "https://company.okta.com")
	_ = os.Setenv("OIDC_AUDIENCE", "api://mcp-server")
	defer os.Clearenv()

	cfg := LoadConfig()

	if !cfg.OAuthEnabled {
		t.Errorf("Expected OAuthEnabled true, got %v", cfg.OAuthEnabled)
	}
	if cfg.OAuthMode != "native" {
		t.Errorf("Expected OAuthMode native, got %s", cfg.OAuthMode)
	}
	if cfg.OAuthProvider != "okta" {
		t.Errorf("Expected OAuthProvider okta, got %s", cfg.OAuthProvider)
	}
	if cfg.OAuthServerURL != "https://localhost:8080" {
		t.Errorf("Expected OAuthServerURL https://localhost:8080, got %s", cfg.OAuthServerURL)
	}
	if cfg.OIDCIssuer != "https://company.okta.com" {
		t.Errorf("Expected OIDCIssuer https://company.okta.com, got %s", cfg.OIDCIssuer)
	}
	if cfg.OIDCAudience != "api://mcp-server" {
		t.Errorf("Expected OIDCAudience api://mcp-server, got %s", cfg.OIDCAudience)
	}
	if cfg.OIDCClientID != "" {
		t.Errorf("Expected OIDCClientID empty in native mode, got %s", cfg.OIDCClientID)
	}
}

func TestLoadConfig_OAuthProxyMode(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("OAUTH_ENABLED", "true")
	_ = os.Setenv("OAUTH_MODE", "proxy")
	_ = os.Setenv("OAUTH_PROVIDER", "google")
	_ = os.Setenv("OIDC_CLIENT_ID", "client-id")
	_ = os.Setenv("OIDC_CLIENT_SECRET", "client-secret")
	_ = os.Setenv("OAUTH_REDIRECT_URIS", "http://localhost:8080/callback")
	_ = os.Setenv("JWT_SECRET", "super-secret-key")
	defer os.Clearenv()

	cfg := LoadConfig()

	if cfg.OAuthMode != "proxy" {
		t.Errorf("Expected OAuthMode proxy, got %s", cfg.OAuthMode)
	}
	if cfg.OAuthProvider != "google" {
		t.Errorf("Expected OAuthProvider google, got %s", cfg.OAuthProvider)
	}
	if cfg.OIDCClientID != "client-id" {
		t.Errorf("Expected OIDCClientID client-id, got %s", cfg.OIDCClientID)
	}
	if cfg.OIDCClientSecret != "client-secret" {
		t.Errorf("Expected OIDCClientSecret client-secret, got %s", cfg.OIDCClientSecret)
	}
	if cfg.OAuthRedirectURIs != "http://localhost:8080/callback" {
		t.Errorf("Expected OAuthRedirectURIs http://localhost:8080/callback, got %s", cfg.OAuthRedirectURIs)
	}
	if cfg.JWTSecret != "super-secret-key" {
		t.Errorf("Expected JWTSecret super-secret-key, got %s", cfg.JWTSecret)
	}
}

func TestLoadConfig_HTTPPortCustom(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("MCP_HTTP_PORT", "9090")
	defer os.Clearenv()

	cfg := LoadConfig()

	if cfg.HTTPPort != 9090 {
		t.Errorf("Expected HTTPPort 9090, got %d", cfg.HTTPPort)
	}
}

func TestLoadConfig_HTTPPortInvalid(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("MCP_HTTP_PORT", "invalid")
	defer os.Clearenv()

	cfg := LoadConfig()

	if cfg.HTTPPort != 8080 {
		t.Errorf("Expected HTTPPort to fallback to 8080, got %d", cfg.HTTPPort)
	}
}
