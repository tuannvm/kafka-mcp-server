package mcp

import (
	"context"
	"encoding/json" // Added for JSON marshalling
	"log/slog"

	mcp "github.com/mark3labs/mcp-go/mcp" // Explicit alias
	"github.com/tuannvm/kafka-mcp-server/kafka"
)

// RegisterResources defines and registers MCP resources with the server.
func RegisterResources(s *mcp.Server, kafkaClient *kafka.Client) {
	listTopicsResource := mcp.NewResource("list_topics",
		mcp.WithDescription("List available Kafka topics"), // Use WithDescription
	)

	// Workaround: Use Tool request/result types for the resource handler signature
	s.AddResource(listTopicsResource, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.InfoContext(ctx, "Executing list_topics resource")

		// Call the actual Kafka client method
		topics, err := kafkaClient.ListTopics(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to list topics", "error", err)
			// Use NewToolResultError as resource-specific error result seems unavailable
			return mcp.NewToolResultError(err.Error()), nil
		}

		slog.InfoContext(ctx, "Successfully listed topics", "count", len(topics))

		// Marshal to JSON
		jsonData, marshalErr := json.Marshal(topics)
		if marshalErr != nil {
			slog.ErrorContext(ctx, "Failed to marshal topics to JSON", "error", marshalErr)
			// Use NewToolResultError for marshalling errors
			return mcp.NewToolResultError("Internal server error: failed to marshal topics"), nil
		}

		// Use NewToolResultText with JSON content as resource-specific JSON result seems unavailable
		// Note: This might result in Content-Type: text/plain instead of application/json
		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// TODO: Add topic_metadata resource
	// TODO: Add consumer_group info resource
}
