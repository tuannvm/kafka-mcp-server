package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	mcp "github.com/mark3labs/mcp-go/mcp" // Explicit alias
	"github.com/tuannvm/kafka-mcp-server/config"
)

// NewMCPServer creates a new MCP server instance.
func NewMCPServer(name, version string) *mcp.Server {
	// Configure logging (optional, customize as needed)
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	server := mcp.NewServer(name, version)
	// Add middleware here if needed later
	// server.Use(...)
	return server
}

// Start runs the MCP server based on the configured transport.
func Start(ctx context.Context, s *mcp.Server, cfg config.Config) error {
	slog.Info("Starting MCP server", "transport", cfg.MCPTransport)

	switch cfg.MCPTransport {
	case "stdio":
		// ServeStdio blocks until the context is cancelled or an error occurs.
		return s.ServeStdio(ctx)
	case "http":
		// TODO: Implement HTTP transport with SSE
		return fmt.Errorf("HTTP transport not yet implemented")
	default:
		return fmt.Errorf("unsupported MCP transport: %s", cfg.MCPTransport)
	}
}
