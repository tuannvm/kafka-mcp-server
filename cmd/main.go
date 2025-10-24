package main

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

// Version is set during build via -X ldflags
var Version = "dev"

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
	cfg := config.LoadConfig() // Changed from cfg, err := config.LoadConfig()

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
