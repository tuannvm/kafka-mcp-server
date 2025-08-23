package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server" // Import directly without alias
	"github.com/tuannvm/kafka-mcp-server/config"
	"github.com/tuannvm/kafka-mcp-server/kafka"
)

// RegisterTools defines and registers MCP tools with the server.
// Updated signature to accept config.Config
func RegisterTools(s *server.MCPServer, kafkaClient kafka.KafkaClient, cfg config.Config) {
	// --- produce_message tool definition and handler ---
	produceTool := mcp.NewTool("produce_message",
		mcp.WithDescription("Produces a single message to a specified Kafka topic. Use this tool when you need to send data, events, or notifications to a Kafka topic. The message can include an optional key for partitioning and routing."),
		mcp.WithString("topic", mcp.Required(), mcp.Description("The name of the Kafka topic to send the message to. Must be an existing topic name.")),
		mcp.WithString("key", mcp.Description("Optional message key used for partitioning. Messages with the same key will be sent to the same partition. Leave empty for random partitioning.")),
		mcp.WithString("value", mcp.Required(), mcp.Description("The message content/payload to send. Can be plain text, JSON, or any string data.")),
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
	consumeTool := mcp.NewTool("consume_messages",
		mcp.WithDescription("Consumes messages from one or more Kafka topics in a single batch operation. Use this tool to retrieve recent messages for analysis, monitoring, or processing. Messages are consumed from the latest available offsets."),
		mcp.WithArray("topics", mcp.Required(), mcp.Description("Array of Kafka topic names to consume messages from. Each topic must exist in the cluster.")),
		mcp.WithNumber("max_messages", mcp.Description("Maximum number of messages to consume across all topics (default: 10). Use higher values for bulk processing, lower values for quick sampling.")),
	)

	s.AddTool(consumeTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) { // Use mcp.CallToolRequest, mcp.CallToolResult
		topicsArg, _ := req.Params.Arguments["topics"].([]interface{})
		
		// Handle max_messages parameter with default
		maxMessages := 10 // Default value
		if maxMessagesArg, exists := req.Params.Arguments["max_messages"]; exists {
			if val, ok := maxMessagesArg.(float64); ok && val > 0 {
				maxMessages = int(val)
			}
		}

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
		mcp.WithDescription("Lists all configured Kafka broker addresses that the server is connecting to. Use this tool to verify connectivity and understand the cluster topology. Returns the broker hostnames and ports as configured."),
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
		mcp.WithDescription("Provides comprehensive metadata and configuration details for a specific Kafka topic. Returns information about partitions, replication factors, leaders, replicas, and in-sync replicas (ISRs). Use this tool to understand topic structure, troubleshoot replication issues, or verify topic configuration."),
		mcp.WithString("topic_name", mcp.Required(), mcp.Description("The exact name of the Kafka topic to describe. Topic names are case-sensitive and must exist in the cluster.")),
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
		mcp.WithDescription("Enumerates all consumer groups known by the Kafka cluster, including their current states. Use this tool to discover active consumer applications, monitor consumer group health, or identify unused consumer groups. Returns group IDs, states, and error codes."),
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
		mcp.WithDescription("Provides detailed information about a specific consumer group including its current state, active members, partition assignments, and optionally offset and lag information. Use this tool to troubleshoot consumer lag, monitor group membership, or analyze partition distribution across consumers."),
		mcp.WithString("group_id", mcp.Required(), mcp.Description("The unique identifier of the consumer group to describe. Must be an existing consumer group registered with the cluster.")),
		mcp.WithBoolean("include_offsets", mcp.Description("Whether to include detailed partition offset and lag information (default: false). Enabling this provides commit offsets, current lag, and end offsets but may be slower for groups with many partitions.")),
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
		mcp.WithDescription("Retrieves configuration settings for Kafka resources such as topics or brokers. Use this tool to examine retention policies, replication settings, segment sizes, cleanup policies, and other configuration parameters. Helps troubleshoot performance issues and verify configuration compliance."),
		mcp.WithString("resource_type", mcp.Required(), mcp.Description("Type of Kafka resource to query. Must be either 'topic' (for topic-specific configs) or 'broker' (for broker-specific configs).")),
		mcp.WithString("resource_name", mcp.Required(), mcp.Description("Name of the resource to describe. For topics: use the exact topic name. For brokers: use the broker ID (numeric string like '1', '2', etc.).")),
		mcp.WithArray("config_keys", mcp.Description("Optional array of specific configuration keys to retrieve (e.g., ['retention.ms', 'segment.bytes']). If omitted, returns all non-default configuration values for the resource.")),
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

		// Map string type to ConfigResourceType
		resourceType, err := kafkaClient.StringToResourceType(resourceTypeStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid resource_type: %s", resourceTypeStr)), nil
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
		mcp.WithDescription("Provides a comprehensive health summary of the entire Kafka cluster including broker status, controller information, topic and partition counts, and replication health metrics. Use this tool for cluster monitoring, health checks, and getting a quick overview of cluster state and potential issues."),
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

	// --- list_topics tool definition and handler ---
	listTopicsTool := mcp.NewTool("list_topics",
		mcp.WithDescription("Retrieves a complete list of all topics in the Kafka cluster along with their metadata including partition counts, replication factors, and internal topic flags. Use this tool to discover available topics, understand cluster topology, or inventory data streams in the cluster."),
	)

	s.AddTool(listTopicsTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slog.InfoContext(ctx, "Executing list_topics tool")

		// Call the ListTopics method from the KafkaClient interface
		topics, err := kafkaClient.ListTopics(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to list topics", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list topics: %v", err)), nil
		}

		// For each topic, get additional metadata like partition count and replication factor
		topicsWithMetadata := make([]map[string]interface{}, 0, len(topics))

		for _, topicName := range topics {
			// Get detailed metadata for each topic
			metadata, err := kafkaClient.DescribeTopic(ctx, topicName)
			if err != nil {
				slog.WarnContext(ctx, "Failed to get metadata for topic", "topic", topicName, "error", err)
				// Continue with the next topic instead of failing entirely
				topicsWithMetadata = append(topicsWithMetadata, map[string]interface{}{
					"name":  topicName,
					"error": err.Error(),
				})
				continue
			}

			// Calculate partition count and replication factor
			partitionCount := len(metadata.Partitions)

			// Find the most common replication factor (assuming it's consistent across partitions)
			replicationFactor := 0
			if partitionCount > 0 && len(metadata.Partitions) > 0 {
				replicationFactor = len(metadata.Partitions[0].Replicas)
			}

			topicsWithMetadata = append(topicsWithMetadata, map[string]interface{}{
				"name":               topicName,
				"partition_count":    partitionCount,
				"replication_factor": replicationFactor,
				"is_internal":        metadata.IsInternal,
			})
		}

		// Marshal to JSON
		jsonData, err := json.Marshal(topicsWithMetadata)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to marshal topics to JSON", "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Internal error: %v", err)), nil
		}

		slog.InfoContext(ctx, "Successfully listed topics", "count", len(topics))
		return mcp.NewToolResultText(string(jsonData)), nil
	})

	// TODO: Add admin tools (create_topic, delete_topic, etc.)
}
