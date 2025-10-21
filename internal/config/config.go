package config

import (
	"os"
	"strconv" // Added for boolean parsing
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

	return Config{
		KafkaBrokers: strings.Split(brokers, ","),
		KafkaClientID: clientID,
		MCPTransport: mcpTransport,

		SASLMechanism: saslMechanism,
		SASLUser:      saslUser,
		SASLPassword:  saslPassword,

		TLSEnable:            tlsEnable,
		TLSInsecureSkipVerify: tlsInsecureSkipVerify,
	}
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
