package mcp

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tuannvm/kafka-mcp-server/kafka"
)

// RegisterResources defines and registers MCP resources with the server.
func RegisterResources(s *server.MCPServer, kafkaClient *kafka.Client) {
	listTopicsResource := mcp.NewResource("list_topics",
		"List available Kafka topics",
	)

	// Use the correct ResourceHandlerFunc signature
	s.AddResource(listTopicsResource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		slog.InfoContext(ctx, "Executing list_topics resource")

		// Call the actual Kafka client method
		topics, err := kafkaClient.ListTopics(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to list topics", "error", err)
			return nil, err
		}

		slog.InfoContext(ctx, "Successfully listed topics", "count", len(topics))

		// Marshal to JSON
		jsonData, marshalErr := json.Marshal(topics)
		if marshalErr != nil {
			slog.ErrorContext(ctx, "Failed to marshal topics to JSON", "error", marshalErr)
			return nil, marshalErr
		}

		// Return as a single text resource content
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				Text: string(jsonData),
			},
		}, nil
	})

	// TODO: Add topic_metadata resource
	// TODO: Add consumer_group info resource
}
