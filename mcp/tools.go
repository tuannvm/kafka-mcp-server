package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	mcpServer "github.com/mark3labs/mcp-go/server" // Import the server package with alias
	"github.com/tuannvm/kafka-mcp-server/config"
	"github.com/tuannvm/kafka-mcp-server/kafka"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// RegisterTools defines and registers MCP tools with the server.
// Updated signature to accept config.Config
func RegisterTools(s *mcpServer.MCPServer, kafkaClient *kafka.Client, cfg config.Config) {
	// --- produce_message tool definition and handler ---
	produceTool := mcp.NewTool("produce_message",
		mcp.WithDescription("Produce a message to a Kafka topic"),
		mcp.WithString("topic", mcp.Required(), mcp.Description("Target Kafka topic name")),
		mcp.WithString("key", mcp.Description("Optional message key (string)")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Message value (string)")),
	)

	s.AddTool(produceTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { // Use mcp.CallToolRequest, mcp.CallToolResult
		// Access parameters via req.Params.Arguments
		topic, _ := req.Params.Arguments["topic"].(string)
		keyArg, _ := req.Params.Arguments["key"].(string)
		value, _ := req.Params.Arguments["value"].(string)

		slog.InfoContext(ctx, "Executing produce_message tool", "topic", topic, "key", keyArg)

		err := kafkaClient.ProduceMessage(ctx, topic, []byte(keyArg), []byte(value))
		if err != nil {
			slog.ErrorContext(ctx, "Failed to produce message", "error", err)
			return mcp.NewToolResultError(err.Error()), nil // Use mcp.NewToolResultError
		}

		slog.InfoContext(ctx, "Message produced successfully", "topic", topic)
		return mcp.NewToolResultText("Message produced successfully to topic " + topic), nil // Use mcp.NewToolResultText
	})

	// --- consume_messages tool definition and handler ---
	// NOTE: There seem to be compilation errors here related to mcp.WithInteger/mcp.Default
	consumeTool := mcp.NewTool("consume_messages",
		mcp.WithDescription("Consume a batch of messages from Kafka topics."),
		mcp.WithArray("topics", mcp.Required(), mcp.Description("List of Kafka topics to consume from.")),
		// mcp.WithInteger("max_messages", mcp.Default(10), mcp.Description("Maximum number of messages to consume in one batch.")), // Potential error source
	)

	s.AddTool(consumeTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { // Use mcp.CallToolRequest, mcp.CallToolResult
		topicsArg, _ := req.Params.Arguments["topics"].([]interface{})
		// maxMessagesArg, _ := req.Params.Arguments["max_messages"].(int64) // Related to potential error above

		// Convert []interface{} to []string for topics
		topics := make([]string, 0, len(topicsArg))
		for _, t := range topicsArg {
			if topicStr, ok := t.(string); ok {
				topics = append(topics, topicStr)
			} else {
				slog.WarnContext(ctx, "Invalid topic type in request", "topic", t)
			}
		}

		if len(topics) == 0 {
			return mcp.NewToolResultError("No valid topics provided."), nil // Use mcp.NewToolResultError
		}

		// maxMessages := int(maxMessagesArg) // Convert to int for client method
		// if maxMessages <= 0 {
		// 	maxMessages = 1 // Ensure at least 1 message is requested if not positive
		// }
		maxMessages := 10 // Hardcode default for now due to potential error

		slog.InfoContext(ctx, "Executing consume_messages tool", "topics", topics, "maxMessages", maxMessages)

		// Call the client method
		messages, err := kafkaClient.ConsumeMessages(ctx, topics, maxMessages)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to consume messages", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to consume messages: %v", err)), nil // Use mcp.NewToolResultError
		}

		slog.InfoContext(ctx, "Successfully consumed messages", "count", len(messages))

		// Marshal result to JSON
		jsonData, marshalErr := json.Marshal(messages)
		if marshalErr != nil {
			slog.ErrorContext(ctx, "Failed to marshal consumed messages to JSON", "error", marshalErr)
			return mcp.NewToolResultError("Internal server error: failed to marshal results"), nil // Use mcp.NewToolResultError
		}

		// Return JSON as text
		return mcp.NewToolResultText(string(jsonData)), nil // Use mcp.NewToolResultText
	})

	// --- NEW: List Brokers Tool ---
	listBrokersTool := mcp.NewTool("list_brokers",
		mcp.WithDescription("Lists the configured Kafka broker addresses."),
		// No parameters needed
	)

	s.AddTool(listBrokersTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { // Use mcp.CallToolRequest, mcp.CallToolResult
		slog.InfoContext(ctx, "Executing list_brokers tool")

		brokers := cfg.KafkaBrokers // Access brokers from config

		// Marshal broker list to JSON
		jsonData, marshalErr := json.Marshal(brokers)
		if marshalErr != nil {
			slog.ErrorContext(ctx, "Failed to marshal broker list to JSON", "error", marshalErr)
			return mcp.NewToolResultError("Internal server error: failed to marshal results"), nil // Use mcp.NewToolResultError
		}

		slog.InfoContext(ctx, "Successfully retrieved broker list", "brokers", brokers)
		// Return JSON as text
		return mcp.NewToolResultText(string(jsonData)), nil // Use mcp.NewToolResultText
	})

	// --- NEW: Describe Topic Tool ---
	describeTopicTool := mcp.NewTool("describe_topic",
		mcp.WithDescription("Provides detailed metadata for a specific Kafka topic, including partition leaders, replicas, and ISRs."),
		mcp.WithString("topic_name", mcp.Required(), mcp.Description("The name of the topic to describe.")),
	)

	s.AddTool(describeTopicTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		topicName, ok := req.Params.Arguments["topic_name"].(string)
		if !ok || topicName == "" {
			return mcp.NewToolResultError("Missing or invalid required parameter: topic_name (string)"), nil
		}

		slog.InfoContext(ctx, "Executing describe_topic tool", "topic", topicName)

		// Call the client method
		metadata, err := kafkaClient.DescribeTopic(ctx, topicName)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to describe topic", "topic", topicName, "error", err)
			// Pass the specific error message from the client
			return mcp.NewToolResultError(fmt.Sprintf("Failed to describe topic '%s': %v", topicName, err)), nil
		}

		slog.InfoContext(ctx, "Successfully described topic", "topic", topicName)

		// Marshal result to JSON
		jsonData, marshalErr := json.MarshalIndent(metadata, "", "  ") // Use MarshalIndent for readability
		if marshalErr != nil {
			slog.ErrorContext(ctx, "Failed to marshal topic metadata to JSON", "topic", topicName, "error", marshalErr)
			return mcp.NewToolResultError("Internal server error: failed to marshal results"), nil
		}

		// Return JSON as text
		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// --- NEW: List Consumer Groups Tool ---
	listGroupsTool := mcp.NewTool("list_consumer_groups",
		mcp.WithDescription("Enumerates active consumer groups known by the Kafka cluster."),
		// No parameters needed for this tool
	)

	s.AddTool(listGroupsTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.InfoContext(ctx, "Executing list_consumer_groups tool")

		// Call the client method
		groups, err := kafkaClient.ListConsumerGroups(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to list consumer groups", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list consumer groups: %v", err)), nil
		}

		slog.InfoContext(ctx, "Successfully listed consumer groups", "count", len(groups))

		// Marshal result to JSON
		jsonData, marshalErr := json.MarshalIndent(groups, "", "  ") // Use MarshalIndent for readability
		if marshalErr != nil {
			slog.ErrorContext(ctx, "Failed to marshal consumer group list to JSON", "error", marshalErr)
			return mcp.NewToolResultError("Internal server error: failed to marshal results"), nil
		}

		// Return JSON as text
		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// --- NEW: Describe Consumer Group Tool ---
	describeGroupTool := mcp.NewTool("describe_consumer_group",
		mcp.WithDescription("Shows details for a specific consumer group, including state, members, and optionally partition offsets and lag."),
		mcp.WithString("group_id", mcp.Required(), mcp.Description("The ID of the consumer group to describe.")),
		// Define boolean without default; handle default in handler
		mcp.WithBoolean("include_offsets", mcp.Description("Whether to include partition offset and lag information (default: false, can be slow).")),
	)

	s.AddTool(describeGroupTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		groupID, ok := req.Params.Arguments["group_id"].(string)
		if !ok || groupID == "" {
			return mcp.NewToolResultError("Missing or invalid required parameter: group_id (string)"), nil
		}

		// Handle default for include_offsets
		includeOffsets := false // Default value
		if includeOffsetsArg, exists := req.Params.Arguments["include_offsets"]; exists {
			if val, ok := includeOffsetsArg.(bool); ok {
				includeOffsets = val
			}
		}

		slog.InfoContext(ctx, "Executing describe_consumer_group tool", "group", groupID, "includeOffsets", includeOffsets)

		// Call the client method
		groupDetails, err := kafkaClient.DescribeConsumerGroup(ctx, groupID, includeOffsets)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to describe consumer group", "group", groupID, "error", err)
			// Pass the specific error message from the client
			return mcp.NewToolResultError(fmt.Sprintf("Failed to describe consumer group '%s': %v", groupID, err)), nil
		}

		slog.InfoContext(ctx, "Successfully described consumer group", "group", groupID)

		// Marshal result to JSON
		jsonData, marshalErr := json.MarshalIndent(groupDetails, "", "  ") // Use MarshalIndent for readability
		if marshalErr != nil {
			slog.ErrorContext(ctx, "Failed to marshal consumer group details to JSON", "group", groupID, "error", marshalErr)
			return mcp.NewToolResultError("Internal server error: failed to marshal results"), nil
		}

		// Return JSON as text
		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// --- NEW: Describe Configs Tool ---
	describeConfigsTool := mcp.NewTool("describe_configs",
		mcp.WithDescription("Fetches configuration entries for a specific resource (topic or broker)."),
		mcp.WithString("resource_type", mcp.Required(), mcp.Description("Type of resource ('topic' or 'broker').")),
		mcp.WithString("resource_name", mcp.Required(), mcp.Description("Name of the topic or ID of the broker.")),
		mcp.WithArray("config_keys", mcp.Description("Optional list of specific config keys to fetch. If empty, fetches all non-default configs.")),
	)

	s.AddTool(describeConfigsTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resourceTypeStr, ok := req.Params.Arguments["resource_type"].(string)
		if !ok || (resourceTypeStr != "topic" && resourceTypeStr != "broker") {
			return mcp.NewToolResultError("Missing or invalid required parameter: resource_type (must be 'topic' or 'broker')"), nil
		}
		resourceName, ok := req.Params.Arguments["resource_name"].(string)
		if !ok || resourceName == "" {
			return mcp.NewToolResultError("Missing or invalid required parameter: resource_name (string)"), nil
		}

		var configKeys []string
		if keysArg, exists := req.Params.Arguments["config_keys"].([]interface{}); exists {
			configKeys = make([]string, 0, len(keysArg))
			for _, k := range keysArg {
				if keyStr, ok := k.(string); ok {
					configKeys = append(configKeys, keyStr)
				} else {
					slog.WarnContext(ctx, "Invalid config key type in request", "key", k)
				}
			}
		}

		// Map string type to kmsg.ConfigResourceType
		var resourceType kmsg.ConfigResourceType
		switch resourceTypeStr {
		case "topic":
			resourceType = kmsg.ConfigResourceTypeTopic
		case "broker":
			resourceType = kmsg.ConfigResourceTypeBroker
		default:
			// Should be caught by initial validation, but handle defensively
			return mcp.NewToolResultError(fmt.Sprintf("Internal error: unhandled resource_type '%s'", resourceTypeStr)), nil
		}

		slog.InfoContext(ctx, "Executing describe_configs tool", "resourceType", resourceTypeStr, "resourceName", resourceName, "configKeys", configKeys)

		// Call the client method
		configDetails, err := kafkaClient.DescribeConfigs(ctx, resourceType, resourceName, configKeys)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to describe configs", "resourceType", resourceTypeStr, "resourceName", resourceName, "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to describe configs for %s '%s': %v", resourceTypeStr, resourceName, err)), nil
		}

		slog.InfoContext(ctx, "Successfully described configs", "resourceType", resourceTypeStr, "resourceName", resourceName)

		// Marshal result to JSON
		jsonData, marshalErr := json.MarshalIndent(configDetails, "", "  ")
		if marshalErr != nil {
			slog.ErrorContext(ctx, "Failed to marshal config details to JSON", "resourceType", resourceTypeStr, "resourceName", resourceName, "error", marshalErr)
			return mcp.NewToolResultError("Internal server error: failed to marshal results"), nil
		}

		// Return JSON as text
		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// --- NEW: Cluster Overview Tool ---
	clusterOverviewTool := mcp.NewTool("cluster_overview",
		mcp.WithDescription("Aggregates high-level cluster health data, such as controller, broker/topic/partition counts, and under-replicated/offline partitions."),
		// No parameters needed for this tool
	)

	s.AddTool(clusterOverviewTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.InfoContext(ctx, "Executing cluster_overview tool")

		// Call the client method
		overview, err := kafkaClient.GetClusterOverview(ctx)
		if err != nil {
			// Log the error, but potentially return partial overview data if available
			slog.ErrorContext(ctx, "Failed to get complete cluster overview", "error", err)
			// If overview is non-nil, it might contain partial data + error message
			if overview != nil {
				// Marshal partial result to JSON
				jsonData, marshalErr := json.MarshalIndent(overview, "", "  ")
				if marshalErr != nil {
					slog.ErrorContext(ctx, "Failed to marshal partial cluster overview to JSON", "error", marshalErr)
					return mcp.NewToolResultError("Internal server error: failed to marshal partial results"), nil
				}
				// Return partial data with a warning
				return mcp.NewToolResultText(fmt.Sprintf("Warning: Could not retrieve complete overview. Partial data: %s", string(jsonData))), nil
			}
			// If overview is nil, return a generic error
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get cluster overview: %v", err)), nil
		}

		slog.InfoContext(ctx, "Successfully retrieved cluster overview")

		// Marshal full result to JSON
		jsonData, marshalErr := json.MarshalIndent(overview, "", "  ")
		if marshalErr != nil {
			slog.ErrorContext(ctx, "Failed to marshal cluster overview to JSON", "error", marshalErr)
			return mcp.NewToolResultError("Internal server error: failed to marshal results"), nil
		}

		// Return JSON as text
		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// TODO: Add admin tools (create_topic, delete_topic, etc.)
}
