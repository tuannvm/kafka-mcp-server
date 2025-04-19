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
		os.Setenv("KAFKA_BROKERS", originalBrokers)
		os.Setenv("KAFKA_CLIENT_ID", originalClientID)
		os.Setenv("MCP_TRANSPORT", originalTransport)
		os.Setenv("KAFKA_SASL_MECHANISM", originalSASLMech)
		os.Setenv("KAFKA_SASL_USER", originalSASLUser)
		os.Setenv("KAFKA_SASL_PASSWORD", originalSASLPass)
		os.Setenv("KAFKA_TLS_ENABLE", originalTLSEnable)
		os.Setenv("KAFKA_TLS_INSECURE_SKIP_VERIFY", originalTLSInsecure)
	}()

	os.Setenv("KAFKA_BROKERS", "test-broker1:9092,test-broker2:9092")
	os.Setenv("KAFKA_CLIENT_ID", "test-client")
	os.Setenv("MCP_TRANSPORT", "stdio")
	os.Setenv("KAFKA_SASL_MECHANISM", "plain")
	os.Setenv("KAFKA_SASL_USER", "testuser")
	os.Setenv("KAFKA_SASL_PASSWORD", "testpass")
	os.Setenv("KAFKA_TLS_ENABLE", "true")
	os.Setenv("KAFKA_TLS_INSECURE_SKIP_VERIFY", "true")

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
	// Clear relevant environment variables to test defaults
	originalBrokers := os.Getenv("KAFKA_BROKERS")
	originalClientID := os.Getenv("KAFKA_CLIENT_ID")
	originalTransport := os.Getenv("MCP_TRANSPORT")
	originalSASLMech := os.Getenv("KAFKA_SASL_MECHANISM")
	originalTLSEnable := os.Getenv("KAFKA_TLS_ENABLE")
	originalTLSInsecure := os.Getenv("KAFKA_TLS_INSECURE_SKIP_VERIFY")

	defer func() {
		// Restore original environment variables
		os.Setenv("KAFKA_BROKERS", originalBrokers)
		os.Setenv("KAFKA_CLIENT_ID", originalClientID)
		os.Setenv("MCP_TRANSPORT", originalTransport)
		os.Setenv("KAFKA_SASL_MECHANISM", originalSASLMech)
		os.Setenv("KAFKA_TLS_ENABLE", originalTLSEnable)
		os.Setenv("KAFKA_TLS_INSECURE_SKIP_VERIFY", originalTLSInsecure)
	}()

	os.Unsetenv("KAFKA_BROKERS")
	os.Unsetenv("KAFKA_CLIENT_ID")
	os.Unsetenv("MCP_TRANSPORT")
	os.Unsetenv("KAFKA_SASL_MECHANISM")
	os.Unsetenv("KAFKA_TLS_ENABLE")
	os.Unsetenv("KAFKA_TLS_INSECURE_SKIP_VERIFY")

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
}
