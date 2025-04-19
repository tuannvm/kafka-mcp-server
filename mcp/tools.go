package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	mcp "github.com/mark3labs/mcp-go/mcp" // Explicit alias
	"github.com/tuannvm/kafka-mcp-server/kafka"
)

// RegisterTools defines and registers MCP tools with the server.
func RegisterTools(s *mcp.Server, kafkaClient *kafka.Client) {
	produceTool := mcp.NewTool("produce_message",
		mcp.WithDescription("Produce a message to a Kafka topic"),
		mcp.WithString("topic", mcp.Required(), mcp.Description("Target Kafka topic name")),
		mcp.WithString("key", mcp.Description("Optional message key (string)")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Message value (string)")),
	)

	s.AddTool(produceTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Access parameters via req.Params.Arguments
		topic, _ := req.Params.Arguments["topic"].(string)
		keyArg, _ := req.Params.Arguments["key"].(string) // Key is optional
		value, _ := req.Params.Arguments["value"].(string)

		slog.InfoContext(ctx, "Executing produce_message tool", "topic", topic, "key", keyArg)

		err := kafkaClient.ProduceMessage(ctx, topic, []byte(keyArg), []byte(value))
		if err != nil {
			slog.ErrorContext(ctx, "Failed to produce message", "error", err)
			// Use NewToolResultError for tool errors
			return mcp.NewToolResultError(err.Error()), nil
		}

		slog.InfoContext(ctx, "Message produced successfully", "topic", topic)
		// Use NewToolResultText for tool text results
		return mcp.NewToolResultText("Message produced successfully to topic " + topic), nil
	})

	// Consume Messages Tool
	consumeTool := mcp.NewTool("consume_messages",
		mcp.WithDescription("Consume a batch of messages from Kafka topics."),
		// Note: group_id is implicitly managed by the client config for now.
		// A future enhancement could allow specifying group_id per call,
		// but that requires more complex client management.
		mcp.WithArray("topics", mcp.Required(), mcp.Description("List of Kafka topics to consume from.")),
		mcp.WithInteger("max_messages", mcp.Default(10), mcp.Description("Maximum number of messages to consume in one batch.")),
	)

	s.AddTool(consumeTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		topicsArg, _ := req.Params.Arguments["topics"].([]interface{})
		maxMessagesArg, _ := req.Params.Arguments["max_messages"].(int64) // MCP uses int64 for integers

		// Convert []interface{} to []string for topics
		topics := make([]string, 0, len(topicsArg))
		for _, t := range topicsArg {
			if topicStr, ok := t.(string); ok {
				topics = append(topics, topicStr)
			} else {
				slog.WarnContext(ctx, "Invalid topic type in request", "topic", t)
				// Optionally return an error here
			}
		}

		if len(topics) == 0 {
			return mcp.NewToolResultError("No valid topics provided."), nil
		}

		maxMessages := int(maxMessagesArg) // Convert to int for client method
		if maxMessages <= 0 {
			maxMessages = 1 // Ensure at least 1 message is requested if not positive
		}

		slog.InfoContext(ctx, "Executing consume_messages tool", "topics", topics, "maxMessages", maxMessages)

		// Call the client method
		// Note: groupID is not passed here, assuming it's set in client config
		messages, err := kafkaClient.ConsumeMessages(ctx, topics, maxMessages)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to consume messages", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to consume messages: %v", err)), nil
		}

		slog.InfoContext(ctx, "Successfully consumed messages", "count", len(messages))

		// Marshal result to JSON
		jsonData, marshalErr := json.Marshal(messages)
		if marshalErr != nil {
			slog.ErrorContext(ctx, "Failed to marshal consumed messages to JSON", "error", marshalErr)
			return mcp.NewToolResultError("Internal server error: failed to marshal results"), nil
		}

		// Return JSON as text
		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// TODO: Add admin tools (create_topic, delete_topic, etc.)
}
