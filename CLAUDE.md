# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running
- `make build` - Build the binary to `bin/kafka-mcp-server`
- `make run-dev` - Run directly from source: `go run cmd/server/main.go`
- `make run` - Run the built binary
- `go build -ldflags "-X main.Version=$(VERSION)" -o bin/kafka-mcp-server ./cmd/server` - Build with version

### Testing
- `make test` - Run all tests including integration tests (requires Docker)
- `make test-no-kafka` or `SKIP_KAFKA_TESTS=true go test ./...` - Skip Kafka integration tests
- `go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...` - Run tests with coverage

### Code Quality
- `make lint` - Run all linters (same as CI pipeline)
- `golangci-lint run --timeout=5m` - Run Go linter
- `go mod tidy` - Clean up module dependencies

### Docker
- `make run-docker` - Build and run Docker container
- `make docker-compose-up` - Start with Docker Compose
- `make docker-compose-down` - Stop Docker Compose

## Project Architecture

### Core Components
- **cmd/server/main.go** - Application entry point with graceful shutdown
- **config/** - Environment-based configuration management
- **kafka/** - Kafka client wrapper using franz-go library
- **mcp/** - MCP server implementation with tools, resources, and prompts

### Key Dependencies
- **github.com/twmb/franz-go** - High-performance Kafka client
- **github.com/mark3labs/mcp-go** - Model Context Protocol implementation
- **github.com/testcontainers/testcontainers-go/modules/kafka** - Integration testing

### Configuration
Environment variables control all configuration:
- `KAFKA_BROKERS` - Comma-separated broker list (default: localhost:9092)
- `KAFKA_CLIENT_ID` - Client identifier (default: kafka-mcp-server)
- `MCP_TRANSPORT` - Transport method: stdio/http (default: stdio)
- `KAFKA_SASL_*` - SASL authentication (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)
- `KAFKA_TLS_*` - TLS configuration

### MCP Implementation
The server exposes:
- **Tools** - Kafka operations (produce/consume messages, describe topics/groups, etc.)
- **Resources** - Cluster health reports and diagnostics (overview, health-check, under-replicated-partitions, consumer-lag-report)
- **Prompts** - Pre-configured workflows for common operations

### Code Structure
- **kafka/interface.go** - Defines KafkaClient interface
- **kafka/client.go** - Implementation using franz-go
- **mcp/tools.go** - MCP tool handlers
- **mcp/resources.go** - MCP resource handlers  
- **mcp/prompts.go** - Pre-configured prompt definitions

### Entry Point Details
The main.go correctly passes the KafkaClient interface to MCP registration functions, enabling dependency injection and testability.

## CI/CD Pipeline

The project uses GitHub Actions with:
- **Code Quality** - golangci-lint, go mod tidy verification
- **Security** - govulncheck, Trivy scanning, SBOM generation  
- **Testing** - Unit and integration tests with coverage
- **Build Verification** - For PRs and non-main branches

## Testing Notes

Integration tests require Docker for Kafka containers. Use `SKIP_KAFKA_TESTS=true` to run only unit tests during development.