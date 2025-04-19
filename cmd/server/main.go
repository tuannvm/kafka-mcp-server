package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/tuannvm/kafka-mcp-server/config"
	"github.com/tuannvm/kafka-mcp-server/kafka"
	"github.com/tuannvm/kafka-mcp-server/mcp"
	"github.com/tuannvm/kafka-mcp-server/server"
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

	// Create MCP server
	s := server.NewMCPServer("kafka-mcp-server", Version)

	// Explicitly declare the client as the KafkaClient interface type
	var kafkaInterface kafka.KafkaClient = kafkaClient

	// Register MCP resources and tools
	mcp.RegisterResources(s, kafkaInterface)
	mcp.RegisterTools(s, kafkaInterface, cfg)

	// Start server
	slog.Info("Starting Kafka MCP server", "version", Version, "transport", cfg.MCPTransport)
	if err := server.Start(ctx, s, cfg); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}

	slog.Info("Server shutdown complete")
}
