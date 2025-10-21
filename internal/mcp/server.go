package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/tuannvm/kafka-mcp-server/internal/config"
)

// NewMCPServer creates a new MCP server instance.
func NewMCPServer(name, version string) *server.MCPServer {
	// Configure logging (optional, customize as needed)
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	srv := server.NewMCPServer(name, version)
	// Add middleware here if needed later
	// srv.Use(...)
	return srv
}

// Start runs the MCP server based on the configured transport.
func Start(ctx context.Context, s *server.MCPServer, cfg config.Config) error {
	slog.Info("Starting MCP server", "transport", cfg.MCPTransport)

	switch cfg.MCPTransport {
	case "stdio":
		// ServeStdio is a standalone function in the server package, not a method on MCPServer
		return server.ServeStdio(s)
	case "http":
		// TODO: Implement HTTP transport with SSE
		return fmt.Errorf("HTTP transport not yet implemented")
	default:
		return fmt.Errorf("unsupported MCP transport: %s", cfg.MCPTransport)
	}
}
