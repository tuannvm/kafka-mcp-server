// filepath: /Users/tuannvm/Projects/cli/kafka-mcp-server/mcp/prompts.go
package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tuannvm/kafka-mcp-server/kafka"
	"golang.org/x/text/cases"    // Import cases package
	"golang.org/x/text/language" // Import language package
)

// RegisterPrompts defines and registers MCP prompts with the server.
func RegisterPrompts(s *server.MCPServer, kafkaClient kafka.KafkaClient) {
	// --- Metadata Inspection Prompts ---

	// List Brokers Prompt
	listBrokersPrompt := mcp.Prompt{
		Name:        "list_brokers",
		Description: "Shows all broker IDs, hostnames, and ports in the Kafka cluster",
		Arguments:   []mcp.PromptArgument{},
	}

	s.AddPrompt(listBrokersPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		var content string

		// Get broker list
		brokers, err := kafkaClient.ListBrokers(ctx)
		if err != nil {
			content = fmt.Sprintf("⚠️ Error fetching broker information: %s", err.Error())
		} else if len(brokers) == 0 {
			content = "No brokers found in the cluster."
		} else {
			content = "KAFKA BROKERS\n\n"
			content += "Broker Address          Status\n"
			content += "-----------------------------\n"

			for _, broker := range brokers {
				content += fmt.Sprintf("%-22s ✅ Connected\n", broker)
			}

			content += "\n\nSlack Command: /kafka list brokers"
		}

		return &mcp.GetPromptResult{
			Description: "Kafka Broker List",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: content,
					},
				},
			},
		}, nil
	})

	// List Topics Prompt
	listTopicsPrompt := mcp.Prompt{
		Name:        "list_topics",
		Description: "Lists all Kafka topics with partition counts and replication factors",
		Arguments:   []mcp.PromptArgument{},
	}

	s.AddPrompt(listTopicsPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		var content string

		topics, err := kafkaClient.ListTopics(ctx)
		if err != nil {
			content = fmt.Sprintf("⚠️ Error fetching topic list: %s", err.Error())
		} else if len(topics) == 0 {
			content = "No topics found in the cluster."
		} else {
			content = "KAFKA TOPICS\n\n"
			content += "Topic Name\n"
			content += "----------\n"
			for _, topic := range topics {
				// TODO: Enhance to show partition/replication factor by calling DescribeTopic for each? Might be slow.
				content += fmt.Sprintf("%s\n", topic)
			}
			content += "\n\nSlack Command: /kafka list topics"
		}

		return &mcp.GetPromptResult{
			Description: "Kafka Topic List",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: content,
					},
				},
			},
		}, nil
	})

	// Describe Topic Prompt
	describeTopicPrompt := mcp.Prompt{
		Name:        "describe_topic",
		Description: "Displays detailed information about a specific Kafka topic",
		Arguments: []mcp.PromptArgument{
			{
				Name:        "topic_name",
				Description: "Name of the topic to describe",
				Required:    true,
			},
		},
	}

	s.AddPrompt(describeTopicPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		topicName := req.Params.Arguments["topic_name"]
		var content string

		metadata, err := kafkaClient.DescribeTopic(ctx, topicName)
		if err != nil {
			content = fmt.Sprintf("⚠️ Error describing topic '%s': %s", topicName, err.Error())
		} else {
			content = fmt.Sprintf("TOPIC DETAILS: %s\n\n", topicName)
			if metadata.IsInternal {
				content += "Internal Topic: Yes\n\n"
			}

			// TODO: Fetch and display topic configuration using DescribeConfigs if needed

			content += "PARTITIONS\n\n"
			if len(metadata.Partitions) == 0 {
				content += "No partition information available.\n"
			} else {
				content += "Partition | Leader | Replicas | In-Sync Replicas | Status\n"
				content += "-----------------------------------------------------\n"
				for _, p := range metadata.Partitions {
					status := "✅ Healthy"
					if p.ErrorCode != 0 {
						status = fmt.Sprintf("⚠️ Error (%d): %s", p.ErrorCode, p.ErrorMessage)
					}
					replicasStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(p.Replicas)), ","), "[]")
					isrStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(p.ISR)), ","), "[]")
					content += fmt.Sprintf("%-9d | %-6d | %-8s | %-16s | %s\n", p.PartitionID, p.Leader, replicasStr, isrStr, status)
				}
			}

			content += "\n\nSlack Command: "
			content += fmt.Sprintf("/kafka describe topic %s", topicName)
		}

		return &mcp.GetPromptResult{
			Description: "Kafka Topic Description",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: content,
					},
				},
			},
		}, nil
	})

	// Cluster Overview Prompt
	clusterOverviewPrompt := mcp.Prompt{
		Name:        "cluster_overview",
		Description: "Summarizes Kafka cluster health including replicated and offline partitions",
		Arguments:   []mcp.PromptArgument{},
	}

	s.AddPrompt(clusterOverviewPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		var content string
		overview, err := kafkaClient.GetClusterOverview(ctx)

		if err != nil {
			content = fmt.Sprintf("⚠️ Error fetching cluster overview: %s", err.Error())
		} else {
			content = "KAFKA CLUSTER OVERVIEW\n\n"

			content += "CLUSTER SUMMARY\n"
			content += fmt.Sprintf("- Broker Count: %d\n", overview.BrokerCount)
			content += fmt.Sprintf("- Active Controller ID: %d\n", overview.ControllerID) // Note: Need broker host/port for full address
			content += fmt.Sprintf("- Total Topics: %d\n", overview.TopicCount)
			content += fmt.Sprintf("- Total Partitions: %d\n\n", overview.PartitionCount)

			content += "HEALTH STATUS\n"
			urpStatus := "✅"
			if overview.UnderReplicatedPartitionsCount > 0 {
				urpStatus = "⚠️"
			}
			content += fmt.Sprintf("- Under-replicated Partitions: %d %s\n", overview.UnderReplicatedPartitionsCount, urpStatus)

			offlineStatus := "✅"
			if overview.OfflinePartitionsCount > 0 {
				offlineStatus = "⚠️"
			}
			content += fmt.Sprintf("- Offline Partitions: %d %s\n", overview.OfflinePartitionsCount, offlineStatus)

			controllerStatus := "✅"
			if overview.ControllerID == -1 {
				controllerStatus = "⚠️ No Active Controller"
			}
			content += fmt.Sprintf("- Active Controller: %s\n", controllerStatus)

			// TODO: Add Consumer Group health summary if needed (requires separate calls)

			content += "\nSlack Command: /kafka cluster overview\n\n"
			content += "Related Commands:\n"
			content += "- /kafka health check - For detailed health diagnostics\n"
			content += "- /kafka under-replicated - For listing under-replicated partitions"
		}

		return &mcp.GetPromptResult{
			Description: "Kafka Cluster Overview",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: content,
					},
				},
			},
		}, nil
	})

	// --- Consumer Group Management Prompts ---

	// List Consumer Groups Prompt
	listConsumerGroupsPrompt := mcp.Prompt{
		Name:        "list_consumer_groups",
		Description: "Lists all active consumer groups in the Kafka cluster",
		Arguments:   []mcp.PromptArgument{},
	}

	s.AddPrompt(listConsumerGroupsPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		var content string
		groups, err := kafkaClient.ListConsumerGroups(ctx)

		if err != nil {
			content = fmt.Sprintf("⚠️ Error fetching consumer group list: %s", err.Error())
		} else if len(groups) == 0 {
			content = "No consumer groups found in the cluster."
		} else {
			content = "KAFKA CONSUMER GROUPS\n\n"
			content += "Consumer Group ID        State\n"
			content += "-------------------------------\n"
			for _, group := range groups {
				status := group.State
				if group.ErrorCode != 0 {
					status = fmt.Sprintf("%s (⚠️ Error %d: %s)", group.State, group.ErrorCode, group.ErrorMessage)
				}
				content += fmt.Sprintf("%-23s %s\n", group.GroupID, status)
			}
			content += "\n\nSlack Command: /kafka list consumer-groups\n\n"
			content += "Related Commands:\n"
			content += "- /kafka describe consumer-group <group-id> - For details on a specific group\n"
			content += "- /kafka consumer-lag report - For detailed lag information"
		}

		return &mcp.GetPromptResult{
			Description: "Kafka Consumer Groups List",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: content,
					},
				},
			},
		}, nil
	})

	// Reset Offsets Prompt
	resetOffsetsPrompt := mcp.Prompt{
		Name:        "reset_offsets",
		Description: "Resets consumer group offsets for a specific topic",
		Arguments: []mcp.PromptArgument{
			{
				Name:        "group_id",
				Description: "The consumer group ID to reset offsets for",
				Required:    true,
			},
			{
				Name:        "topic",
				Description: "The topic name to reset offsets for",
				Required:    true,
			},
			{
				Name:        "offset",
				Description: "Offset position or special value (earliest, latest, or specific number)",
				Required:    true,
			},
		},
	}

	s.AddPrompt(resetOffsetsPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		groupID := req.Params.Arguments["group_id"]
		topic := req.Params.Arguments["topic"]
		offset := req.Params.Arguments["offset"]

		// Get broker list for commands
		var brokerList string
		brokers, err := kafkaClient.ListBrokers(ctx) // Corrected: Use ListBrokers
		if err != nil {
			slog.WarnContext(ctx, "Failed to get broker list for reset offsets prompt", "error", err)
			brokerList = "[broker:port]" // Fallback placeholder
		} else {
			brokerList = strings.Join(brokers, ",")
		}

		content := fmt.Sprintf("## Reset Offsets for Consumer Group: %s\n\n", groupID)

		content += "### Offset Reset Information\n\n"
		content += fmt.Sprintf("- **Topic:** %s\n", topic)
		content += fmt.Sprintf("- **Target Offset:** %s\n", offset)
		content += "- **Status:** ⚠️ This prompt shows the command to execute the reset.\n\n"

		content += "### Command to Execute Reset\n\n"
		content += "```bash\n"

		// Build the proper command based on the offset type
		if offset == "earliest" || offset == "latest" {
			content += fmt.Sprintf("kafka-consumer-groups --bootstrap-server %s \\\n", brokerList)
			content += fmt.Sprintf("  --group %s --topic %s \\\n", groupID, topic)
			content += fmt.Sprintf("  --reset-offsets --to-%s --execute\n", offset)
		} else {
			// Assuming offset is a specific number or timestamp format (adjust if needed)
			content += fmt.Sprintf("kafka-consumer-groups --bootstrap-server %s \\\n", brokerList)
			content += fmt.Sprintf("  --group %s --topic %s \\\n", groupID, topic)
			// Note: kafka-consumer-groups might use different flags like --to-offset, --to-datetime, etc.
			// This example assumes --to-offset for simplicity.
			content += fmt.Sprintf("  --reset-offsets --to-offset %s --execute\n", offset)
		}
		content += "```\n\n"

		content += "### ⚠️ Warning\n\n"
		content += "Resetting offsets will change the position from which consumers in this group will read messages.\n"
		content += "- If moving **backward**, messages may be reprocessed.\n"
		content += "- If moving **forward**, messages may be skipped.\n\n"

		content += "**Slack Command:**\n"
		content += fmt.Sprintf("`/kafka reset offsets %s %s %s`", groupID, topic, offset)

		return &mcp.GetPromptResult{
			Description: "Kafka Consumer Group Offset Reset Command",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: content,
					},
				},
			},
		}, nil
	})

	// --- Topic Operations Prompts ---

	// Create Topic Prompt
	createTopicPrompt := mcp.Prompt{
		Name:        "create_topic",
		Description: "Creates a new Kafka topic with specified parameters",
		Arguments: []mcp.PromptArgument{
			{
				Name:        "topic_name",
				Description: "Name of the topic to create",
				Required:    true,
			},
			{
				Name:        "partitions",
				Description: "Number of partitions for the topic",
				Required:    false,
			},
			{
				Name:        "replication_factor",
				Description: "Replication factor for the topic",
				Required:    false,
			},
			{
				Name:        "retention_ms",
				Description: "Retention time in milliseconds (-1 for unlimited)",
				Required:    false,
			},
		},
	}

	s.AddPrompt(createTopicPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		topicName := req.Params.Arguments["topic_name"]

		// Default values
		partitions := "3" // Consider fetching cluster default if possible
		if p, ok := req.Params.Arguments["partitions"]; ok && p != "" {
			partitions = p
		}

		replicationFactor := "3" // Consider fetching cluster default if possible
		if rf, ok := req.Params.Arguments["replication_factor"]; ok && rf != "" {
			replicationFactor = rf
		}

		retentionMs := "604800000" // 7 days by default, consider fetching broker default
		if rm, ok := req.Params.Arguments["retention_ms"]; ok && rm != "" {
			retentionMs = rm
		}

		// Get broker list for commands
		var brokerList string
		brokers, err := kafkaClient.ListBrokers(ctx) // Corrected: Use ListBrokers
		if err != nil {
			slog.WarnContext(ctx, "Failed to get broker list for create topic prompt", "error", err)
			brokerList = "[broker:port]" // Fallback placeholder
		} else {
			brokerList = strings.Join(brokers, ",")
		}

		content := fmt.Sprintf("# Create Kafka Topic: %s\n\n", topicName)

		content += "## Topic Configuration\n\n"
		content += fmt.Sprintf("- **Topic Name:** %s\n", topicName)
		content += fmt.Sprintf("- **Partitions:** %s\n", partitions)
		content += fmt.Sprintf("- **Replication Factor:** %s\n", replicationFactor)
		content += fmt.Sprintf("- **Retention:** %s ms\n\n", retentionMs)

		content += "## Command to Create Topic\n\n"
		content += "```bash\n"
		content += fmt.Sprintf("kafka-topics --bootstrap-server %s \\\n", brokerList)
		content += fmt.Sprintf("  --create --topic %s \\\n", topicName)
		content += fmt.Sprintf("  --partitions %s \\\n", partitions)
		content += fmt.Sprintf("  --replication-factor %s \\\n", replicationFactor)
		content += fmt.Sprintf("  --config retention.ms=%s\n", retentionMs)
		// TODO: Add other config options if provided
		content += "```\n\n"

		content += "## Verify Topic Creation\n\n"
		content += "```bash\n"
		content += fmt.Sprintf("kafka-topics --bootstrap-server %s --describe --topic %s\n", brokerList, topicName)
		content += "```\n\n"

		content += "**Slack Command:**\n"
		// Improve Slack command representation if needed
		content += fmt.Sprintf("`/kafka create topic %s partitions=%s replication=%s retention_ms=%s`",
			topicName, partitions, replicationFactor, retentionMs)

		return &mcp.GetPromptResult{
			Description: "Kafka Topic Creation Command",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: content,
					},
				},
			},
		}, nil
	})

	// Increase Partitions Prompt
	increasePartitionsPrompt := mcp.Prompt{
		Name:        "increase_partitions",
		Description: "Increases the number of partitions for an existing Kafka topic",
		Arguments: []mcp.PromptArgument{
			{
				Name:        "topic",
				Description: "Name of the topic to modify",
				Required:    true,
			},
			{
				Name:        "partitions",
				Description: "New number of partitions (must be greater than current)",
				Required:    true,
			},
		},
	}

	s.AddPrompt(increasePartitionsPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		topic := req.Params.Arguments["topic"]
		partitions := req.Params.Arguments["partitions"]

		// Get broker list for commands
		var brokerList string
		brokers, err := kafkaClient.ListBrokers(ctx) // Corrected: Use ListBrokers
		if err != nil {
			slog.WarnContext(ctx, "Failed to get broker list for increase partitions prompt", "error", err)
			brokerList = "[broker:port]" // Fallback placeholder
		} else {
			brokerList = strings.Join(brokers, ",")
		}

		// TODO: Optionally, fetch current partition count first to validate the increase

		content := fmt.Sprintf("# Increase Partitions for Topic: %s\n\n", topic)

		content += "## Partition Change Details\n\n"
		content += fmt.Sprintf("- **Topic:** %s\n", topic)
		content += fmt.Sprintf("- **New Partition Count:** %s\n\n", partitions)

		content += "## Command to Increase Partitions\n\n"
		content += "```bash\n"
		content += fmt.Sprintf("kafka-topics --bootstrap-server %s \\\n", brokerList)
		content += fmt.Sprintf("  --alter --topic %s \\\n", topic)
		content += fmt.Sprintf("  --partitions %s\n", partitions)
		content += "```\n\n"

		content += "## Verify Partition Change\n\n"
		content += "```bash\n"
		content += fmt.Sprintf("kafka-topics --bootstrap-server %s --describe --topic %s\n", brokerList, topic)
		content += "```\n\n"

		content += "### ⚠️ Important Notes\n\n"
		content += "1. You can only **increase** the number of partitions (never decrease).\n"
		content += "2. Increasing partitions may affect message ordering guarantees for consumers relying on partition assignment.\n"
		content += "3. Key-based message routing will distribute across the new total number of partitions.\n\n"

		content += "**Slack Command:**\n"
		content += fmt.Sprintf("`/kafka increase partitions %s to %s`", topic, partitions)

		return &mcp.GetPromptResult{
			Description: "Kafka Topic Partition Increase Command",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: content,
					},
				},
			},
		}, nil
	})

	// --- Message Production & Consumption Prompts ---

	// Produce Message Prompt
	produceMessagePrompt := mcp.Prompt{
		Name:        "produce",
		Description: "Publishes a message to a Kafka topic",
		Arguments: []mcp.PromptArgument{
			{
				Name:        "topic",
				Description: "Name of the topic to produce message to",
				Required:    true,
			},
			{
				Name:        "message",
				Description: "Message payload to publish",
				Required:    true,
			},
			{
				Name:        "key",
				Description: "Optional message key",
				Required:    false,
			},
		},
	}

	s.AddPrompt(produceMessagePrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		topic := req.Params.Arguments["topic"]
		message := req.Params.Arguments["message"]
		key, hasKey := req.Params.Arguments["key"]

		// Get broker list for commands
		var brokerList string
		brokers, err := kafkaClient.ListBrokers(ctx) // Corrected: Use ListBrokers
		if err != nil {
			slog.WarnContext(ctx, "Failed to get broker list for produce prompt", "error", err)
			brokerList = "[broker:port]" // Fallback placeholder
		} else {
			brokerList = strings.Join(brokers, ",")
		}

		// Attempt to produce the message using the client
		produceErr := kafkaClient.ProduceMessage(ctx, topic, []byte(key), []byte(message))

		content := fmt.Sprintf("# Produce Message to Topic: %s\n\n", topic)
		content += "## Message Details\n\n"
		content += fmt.Sprintf("- **Topic:** %s\n", topic)
		content += fmt.Sprintf("- **Message:** `%s`\n", message) // Use backticks for clarity
		if hasKey {
			content += fmt.Sprintf("- **Key:** `%s`\n", key)
		}

		if produceErr != nil {
			content += fmt.Sprintf("- **Status:** ⚠️ Failed to produce message: %s\n\n", produceErr.Error())
		} else {
			content += "- **Status:** ✅ Message produced successfully.\n\n"
		}

		content += "## Command Equivalent (using kafka-console-producer)\n\n"

		if !hasKey {
			content += "```bash\n"
			// Escape single quotes within the message for the echo command
			escapedMessage := strings.ReplaceAll(message, "'", "'\\''")
			content += fmt.Sprintf("echo '%s' | kafka-console-producer \\\n", escapedMessage)
			content += fmt.Sprintf("  --bootstrap-server %s \\\n", brokerList)
			content += fmt.Sprintf("  --topic %s\n", topic)
			content += "```\n\n"
		} else {
			content += "```bash\n"
			// Assumes key and message don't contain the separator ':'
			content += fmt.Sprintf("echo '%s:%s' | kafka-console-producer \\\n", key, message)
			content += fmt.Sprintf("  --bootstrap-server %s \\\n", brokerList)
			content += fmt.Sprintf("  --topic %s \\\n", topic)
			content += "  --property parse.key=true \\\n"
			content += "  --property key.separator=:\n"
			content += "```\n\n"
		}

		content += "## Verify Message Consumption (Example)\n\n"
		content += "```bash\n"
		content += "kafka-console-consumer \\\n"
		content += fmt.Sprintf("  --bootstrap-server %s \\\n", brokerList)
		content += fmt.Sprintf("  --topic %s \\\n", topic)
		content += "  --from-beginning \\\n" // Or remove for only new messages
		content += fmt.Sprintf("  --max-messages 1 \\\n", topic)
		content += "  --property print.key=true \\\n"
		content += "  --property key.separator=:\n"
		content += "```\n\n"

		slackCmd := fmt.Sprintf("`/kafka produce %s \"%s\"`", topic, message)
		if hasKey {
			slackCmd = fmt.Sprintf("`/kafka produce %s key=\"%s\" message=\"%s\"`", topic, key, message)
		}
		content += "**Slack Command:**\n" + slackCmd

		return &mcp.GetPromptResult{
			Description: "Kafka Message Production Result",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: content,
					},
				},
			},
		}, nil
	})

	// Consume Messages Prompt
	consumeMessagesPrompt := mcp.Prompt{
		Name:        "consume",
		Description: "Fetches messages from a Kafka topic",
		Arguments: []mcp.PromptArgument{
			{
				Name:        "topic",
				Description: "Name of the topic to consume messages from",
				Required:    true,
			},
			{
				Name:        "count",
				Description: "Number of messages to fetch",
				Required:    false,
			},
			{
				Name:        "from_beginning",
				Description: "Whether to start from the beginning of the topic",
				Required:    false,
			},
		},
	}

	s.AddPrompt(consumeMessagesPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		topic := req.Params.Arguments["topic"]

		// Default values & Parsing
		countStr := "10"
		if c, ok := req.Params.Arguments["count"]; ok && c != "" {
			countStr = c
		}
		maxMessages := 10 // Default int value
		_, err := fmt.Sscan(countStr, &maxMessages)
		if err != nil {
			slog.WarnContext(ctx, "Invalid count provided, using default", "count", countStr, "default", maxMessages)
			// Keep default maxMessages = 10
		}
		if maxMessages <= 0 {
			maxMessages = 1 // Consume at least 1 if count is non-positive
		}

		// Note: 'from_beginning' is tricky with the current client setup.
		// The kgo client consumes based on the group's committed offset.
		// Resetting requires separate logic (e.g., using admin client or specific kgo calls).
		// This implementation will consume from the current group offset.
		fromBeginning := false
		if fb, ok := req.Params.Arguments["from_beginning"]; ok {
			fromBeginning = fb == "true"
			if fromBeginning {
				slog.WarnContext(ctx, "'from_beginning' requested but not directly supported by this simple consume implementation. Consuming from current offset.")
				// In a real scenario, you might trigger an offset reset here if needed.
			}
		}

		// Get broker list for console command example
		var brokerList string
		brokers, err := kafkaClient.ListBrokers(ctx) // Corrected: Use ListBrokers
		if err != nil {
			slog.WarnContext(ctx, "Failed to get broker list for consume prompt", "error", err)
			brokerList = "[broker:port]" // Fallback placeholder
		} else {
			brokerList = strings.Join(brokers, ",")
		}

		// Attempt to consume messages
		consumedMessages, consumeErr := kafkaClient.ConsumeMessages(ctx, []string{topic}, maxMessages)

		content := fmt.Sprintf("# Consume Messages from Topic: %s\n\n", topic)
		content += "## Consumption Details\n\n"
		content += fmt.Sprintf("- **Topic:** %s\n", topic)
		content += fmt.Sprintf("- **Max Messages Requested:** %d\n", maxMessages)

		if consumeErr != nil {
			content += fmt.Sprintf("- **Status:** ⚠️ Error consuming messages: %s\n\n", consumeErr.Error())
		} else if len(consumedMessages) == 0 {
			content += "- **Status:** ✅ Consumed 0 messages (or timed out waiting).\n\n"
		} else {
			content += fmt.Sprintf("- **Status:** ✅ Consumed %d message(s).\n\n", len(consumedMessages))
			content += "### Consumed Messages\n\n"
			content += "| Partition | Offset | Key | Value | Timestamp (ms) |\n"
			content += "|-----------|--------|-----|-------|----------------|\n"
			for _, msg := range consumedMessages {
				// Truncate long values for display
				displayValue := msg.Value
				if len(displayValue) > 100 {
					displayValue = displayValue[:100] + "..."
				}
				displayKey := msg.Key
				if len(displayKey) > 50 {
					displayKey = displayKey[:50] + "..."
				}
				content += fmt.Sprintf("| %d | %d | `%s` | `%s` | %d |\n", msg.Partition, msg.Offset, displayKey, displayValue, msg.Timestamp)
			}
			content += "\n"
		}

		content += "## Command Equivalent (using kafka-console-consumer)\n\n"
		content += "```bash\n"
		content += "kafka-console-consumer \\\n"
		content += fmt.Sprintf("  --bootstrap-server %s \\\n", brokerList)
		content += fmt.Sprintf("  --topic %s \\\n", topic)
		if fromBeginning {
			content += "  --from-beginning \\\n"
		}
		content += fmt.Sprintf("  --max-messages %d \\\n", maxMessages)
		content += "  --property print.key=true \\\n"
		content += "  --property key.separator=:\n"
		content += "```\n\n"

		content += "**Slack Command:**\n"
		slackCmd := fmt.Sprintf("`/kafka consume %s count=%d`", topic, maxMessages)
		content += slackCmd

		return &mcp.GetPromptResult{
			Description: "Kafka Message Consumption Result",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: content,
					},
				},
			},
		}, nil
	})

	// --- Cluster Health & Diagnostics Prompts ---

	// Health Check Prompt
	healthCheckPrompt := mcp.Prompt{
		Name:        "health_check",
		Description: "Runs a comprehensive health check on the Kafka cluster",
		Arguments:   []mcp.PromptArgument{},
	}

	s.AddPrompt(healthCheckPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		var content string
		overview, overviewErr := kafkaClient.GetClusterOverview(ctx)
		// TODO: Potentially add calls to ListBrokers, ListConsumerGroups for more detail

		content = "# Kafka Cluster Health Check\n\n"

		if overviewErr != nil {
			content += fmt.Sprintf("⚠️ Error fetching cluster overview: %s\n\n", overviewErr.Error())
			// Still show command equivalents if possible
		} else {
			content += "## Health Summary (Based on Overview)\n\n"
			content += "| Component | Status | Details |\n"
			content += "|-----------|--------|---------|\n"

			// Broker Status (basic count)
			brokerStatus := "✅ Available"
			brokerDetails := fmt.Sprintf("%d brokers reported", overview.BrokerCount)
			if len(overview.OfflineBrokerIDs) > 0 {
				brokerStatus = fmt.Sprintf("⚠️ %d Offline", len(overview.OfflineBrokerIDs))
				brokerDetails += fmt.Sprintf(" (Offline IDs: %v)", overview.OfflineBrokerIDs)
			} else if overview.BrokerCount == 0 {
				brokerStatus = "⚠️ No Brokers Found"
				brokerDetails = "Could not connect or no brokers available"
			}
			content += fmt.Sprintf("| Brokers | %s | %s |\n", brokerStatus, brokerDetails)

			// Controller Status
			controllerStatus := "✅ Active"
			controllerDetails := fmt.Sprintf("Controller ID: %d", overview.ControllerID)
			if overview.ControllerID == -1 {
				controllerStatus = "⚠️ Inactive"
				controllerDetails = "No active controller found"
			}
			content += fmt.Sprintf("| Controller | %s | %s |\n", controllerStatus, controllerDetails)

			// Partition Health
			partitionStatus := "✅ Healthy"
			partitionDetails := fmt.Sprintf("URP: %d, Offline: %d", overview.UnderReplicatedPartitionsCount, overview.OfflinePartitionsCount)
			if overview.UnderReplicatedPartitionsCount > 0 || overview.OfflinePartitionsCount > 0 {
				partitionStatus = "⚠️ Issues Found"
			}
			content += fmt.Sprintf("| Partitions | %s | %s |\n", partitionStatus, partitionDetails)

			// TODO: Add Consumer Group health summary (requires ListConsumerGroups/DescribeConsumerGroup calls)
			content += "| Consumer Groups | ? Unknown | Requires separate check (`/kafka consumer-lag report`) |\n"
			content += "\n"
		}

		// Get broker list for command examples
		var brokerList string
		brokers, brokerErr := kafkaClient.ListBrokers(ctx)
		if brokerErr != nil {
			slog.WarnContext(ctx, "Failed to get broker list for health check prompt", "error", brokerErr)
			brokerList = "[broker:port]" // Fallback placeholder
		} else {
			brokerList = strings.Join(brokers, ",")
		}

		content += "## Health Check Commands (Examples)\n\n"
		content += "```bash\n"
		content += "# Check under-replicated partitions\n"
		content += fmt.Sprintf("kafka-topics --bootstrap-server %s --describe --under-replicated-partitions\n\n", brokerList)
		content += "# Check unavailable partitions\n"
		content += fmt.Sprintf("kafka-topics --bootstrap-server %s --describe --unavailable-partitions\n\n", brokerList)
		content += "# Check broker API versions (basic connectivity check)\n"
		content += fmt.Sprintf("kafka-broker-api-versions --bootstrap-server %s\n\n", brokerList)
		content += "# Check consumer group lag (all groups)\n"
		content += fmt.Sprintf("kafka-consumer-groups --bootstrap-server %s --all-groups --describe\n", brokerList)
		content += "```\n\n"

		content += "**Slack Command:**\n`/kafka health check`\n\n"
		content += "**Related Commands:**\n"
		content += "- `/kafka under-replicated` - For detailed under-replicated partition info\n"
		content += "- `/kafka consumer-lag report` - For detailed consumer lag analysis"

		return &mcp.GetPromptResult{
			Description: "Kafka Cluster Health Check Results",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: content,
					},
				},
			},
		}, nil
	})

	// Under-Replicated Partitions Prompt
	underReplicatedPrompt := mcp.Prompt{
		Name:        "under_replicated",
		Description: "Lists topics and partitions where ISR < replication factor",
		Arguments:   []mcp.PromptArgument{},
	}

	s.AddPrompt(underReplicatedPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		var content string
		overview, overviewErr := kafkaClient.GetClusterOverview(ctx)
		// For more detail, would need to call DescribeTopics on topics identified as having URPs

		content = "# Under-Replicated Kafka Partitions\n\n"

		if overviewErr != nil {
			content += fmt.Sprintf("⚠️ Error fetching cluster overview: %s\n\n", overviewErr.Error())
		} else {
			content += "## Status Summary\n\n"
			if overview.UnderReplicatedPartitionsCount == 0 {
				content += "No under-replicated partitions found in the cluster. ✅\n\n"
			} else {
				content += fmt.Sprintf("⚠️ Found %d under-replicated partition(s).\n\n", overview.UnderReplicatedPartitionsCount)
				content += "**Note:** Use the command below or `/kafka describe topic <topic_name>` for specific partition details.\n\n"
				// TODO: If feasible, list topics with URPs by iterating through DescribeTopic results (could be slow)
			}
		}

		// Get broker list for command examples
		var brokerList string
		brokers, brokerErr := kafkaClient.ListBrokers(ctx)
		if brokerErr != nil {
			slog.WarnContext(ctx, "Failed to get broker list for URP prompt", "error", brokerErr)
			brokerList = "[broker:port]" // Fallback placeholder
		} else {
			brokerList = strings.Join(brokers, ",")
		}

		content += "## Command to Check Under-Replicated Partitions\n\n"
		content += "```bash\n"
		content += fmt.Sprintf("kafka-topics --bootstrap-server %s --describe --under-replicated-partitions\n", brokerList)
		content += "```\n\n"

		content += "## Common Causes for Under-Replication\n\n"
		content += "1. **Broker Failure:** A broker hosting a replica is down or unreachable.\n"
		content += "2. **Network Issues:** Connectivity problems between brokers prevent replication.\n"
		content += "3. **Disk Full:** A broker's disk is full and cannot accept new replica data.\n"
		content += "4. **High Load:** Brokers are too busy to keep up with replication demands.\n"
		content += "5. **Configuration:** `min.insync.replicas` might be set higher than available ISRs.\n\n"

		content += "## Diagnostic Steps\n\n"
		content += "- Check broker status and logs (`/kafka list brokers`, check server.log).\n"
		content += "- Verify network connectivity between brokers (`ping`, `traceroute`).\n"
		content += "- Monitor disk usage on all brokers (`df -h`).\n"
		content += "- Check broker CPU/Memory/Network utilization.\n"
		content += "- Review topic configuration (`/kafka describe topic <topic_name>`).\n\n"

		content += "**Slack Command:**\n`/kafka under-replicated`"

		return &mcp.GetPromptResult{
			Description: "Kafka Under-Replicated Partitions Report",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: content,
					},
				},
			},
		}, nil
	})

	// Consumer Lag Report Prompt
	consumerLagPrompt := mcp.Prompt{
		Name:        "consumer_lag_report",
		Description: "Provides a detailed report on consumer lag across all consumer groups",
		Arguments: []mcp.PromptArgument{
			{
				Name:        "threshold",
				Description: "Lag threshold to highlight (number of messages)",
				Required:    false,
			},
		},
	}

	s.AddPrompt(consumerLagPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Default threshold & Parsing
		thresholdStr := "1000"
		if t, ok := req.Params.Arguments["threshold"]; ok && t != "" {
			thresholdStr = t
		}
		var lagThreshold int64 = 1000 // Default int value
		_, err := fmt.Sscan(thresholdStr, &lagThreshold)
		if err != nil {
			slog.WarnContext(ctx, "Invalid lag threshold provided, using default", "threshold", thresholdStr, "default", lagThreshold)
			// Keep default lagThreshold = 1000
		}
		if lagThreshold < 0 {
			lagThreshold = 0 // Threshold cannot be negative
		}

		var content string
		groups, listErr := kafkaClient.ListConsumerGroups(ctx)

		content = "# Kafka Consumer Lag Report\n\n"

		if listErr != nil {
			content += fmt.Sprintf("⚠️ Error listing consumer groups: %s\n\n", listErr.Error())
		} else if len(groups) == 0 {
			content += "No consumer groups found.\n\n"
		} else {
			highLagDetails := ""
			allGroupsSummary := "| Consumer Group | Total Lag | Status |\n"
			allGroupsSummary += "|----------------|-----------|--------|\n"
			groupsWithHighLag := 0

			for _, groupInfo := range groups {
				// Describe each group to get offset/lag info
				// Note: This can be slow for many groups. Consider optimizations like parallel requests or filtering.
				descResult, descErr := kafkaClient.DescribeConsumerGroup(ctx, groupInfo.GroupID, true) // includeOffsets = true

				totalLag := int64(0)
				groupStatus := "✅ Normal"
				if descErr != nil {
					slog.WarnContext(ctx, "Error describing consumer group for lag report", "group", groupInfo.GroupID, "error", descErr)
					groupStatus = fmt.Sprintf("⚠️ Error fetching details: %s", descErr.Error())
				} else if descResult.ErrorCode != 0 {
					groupStatus = fmt.Sprintf("⚠️ Group Error %d: %s", descResult.ErrorCode, descResult.ErrorMessage)
				} else if len(descResult.Offsets) == 0 && descResult.State != "Empty" && descResult.State != "Dead" {
					// Only mark as unknown if group is potentially active but has no offset info
					groupStatus = "? Unknown Lag (No offset info)"
				} else {
					for _, offsetInfo := range descResult.Offsets {
						if offsetInfo.Lag > 0 {
							totalLag += offsetInfo.Lag
							if offsetInfo.Lag > lagThreshold {
								groupStatus = "⚠️ High Lag" // Mark group if any partition exceeds threshold
								highLagDetails += fmt.Sprintf("| %s | %s | %d | %d | %d | %d |\n",
									groupInfo.GroupID, offsetInfo.Topic, offsetInfo.Partition,
									offsetInfo.CommitOffset, offsetInfo.CommitOffset+offsetInfo.Lag, // Approx Log End Offset
									offsetInfo.Lag)
							}
						}
					}
					if groupStatus == "⚠️ High Lag" {
						groupsWithHighLag++
					} else if totalLag > 0 {
						groupStatus = "ℹ️ Some Lag" // Indicate lag exists but below threshold
					}
				}
				allGroupsSummary += fmt.Sprintf("| %s | %d | %s |\n", groupInfo.GroupID, totalLag, groupStatus)
			}

			content += fmt.Sprintf("## Consumer Groups Exceeding Lag Threshold (> %d messages)\n\n", lagThreshold)
			if groupsWithHighLag == 0 {
				content += "No consumer groups found exceeding the lag threshold. ✅\n\n"
			} else {
				content += "| Consumer Group | Topic | Partition | Current Offset | Log End Offset (Approx) | Lag |\n"
				content += "|----------------|-------|-----------|----------------|-------------------------|-----|\n"
				content += highLagDetails
				content += "\n"
			}

			content += "## All Consumer Groups Summary\n\n"
			content += allGroupsSummary
			content += "\n"
		}

		// Get broker list for command examples
		var brokerList string
		brokers, brokerErr := kafkaClient.ListBrokers(ctx)
		if brokerErr != nil {
			slog.WarnContext(ctx, "Failed to get broker list for lag report prompt", "error", brokerErr)
			brokerList = "[broker:port]" // Fallback placeholder
		} else {
			brokerList = strings.Join(brokers, ",")
		}

		content += "## Command to Check Consumer Lag\n\n"
		content += "```bash\n"
		content += "# Describe a specific group (includes lag)\n"
		content += fmt.Sprintf("kafka-consumer-groups --bootstrap-server %s --describe --group [group-id]\n\n", brokerList)
		content += "# Describe all groups (can be verbose)\n"
		content += fmt.Sprintf("kafka-consumer-groups --bootstrap-server %s --describe --all-groups\n", brokerList)
		content += "```\n\n"

		content += "## Addressing High Consumer Lag\n\n"
		content += "1. **Scale Consumers**: Increase the number of consumer instances (up to the number of partitions).\n"
		content += "2. **Optimize Processing**: Profile and improve consumer application logic.\n"
		content += "3. **Check Resources**: Ensure consumers have sufficient CPU, memory, and network bandwidth.\n"
		content += "4. **Increase Partitions**: If consumers are maxed out, consider increasing topic partitions (requires careful planning).\n"
		content += "5. **Tune Consumer Config**: Adjust `fetch.min.bytes`, `fetch.max.wait.ms`, `max.poll.records`.\n\n"

		content += "**Slack Command:**\n"
		content += fmt.Sprintf("`/kafka consumer-lag report threshold=%d`", lagThreshold)

		return &mcp.GetPromptResult{
			Description: "Kafka Consumer Lag Analysis",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: content,
					},
				},
			},
		}, nil
	})

	// --- Existing Prompts (your current implementations) ---

	// Monitor Consumer Group Prompt (already implemented)
	monitorConsumerGroupPrompt := mcp.Prompt{
		Name:        "monitor_consumer_group",
		Description: "Provides commands and guidance for monitoring a Kafka consumer group",
		Arguments: []mcp.PromptArgument{
			{
				Name:        "group_id",
				Description: "The consumer group ID to monitor",
				Required:    true,
			},
			{
				Name:        "include_lag_commands",
				Description: "Whether to include lag monitoring commands",
				Required:    false,
			},
		},
	}

	s.AddPrompt(monitorConsumerGroupPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		groupID := req.Params.Arguments["group_id"]

		includeLagCommands := true // Default
		if include, ok := req.Params.Arguments["include_lag_commands"]; ok {
			includeLagCommands = (include != "false") // Treat anything other than "false" as true
		}

		// Get broker list for commands
		var brokerList string
		brokers, err := kafkaClient.ListBrokers(ctx) // Corrected: Use ListBrokers
		if err != nil {
			slog.WarnContext(ctx, "Failed to get broker list for monitor group prompt", "error", err)
			brokerList = "[broker:port]" // Fallback placeholder
		} else {
			brokerList = strings.Join(brokers, ",")
		}

		// Build content
		content := fmt.Sprintf("# Monitor Kafka Consumer Group: %s\n\n", groupID)

		// Optionally fetch group description for current status
		var groupState, memberCountStr string
		descResult, descErr := kafkaClient.DescribeConsumerGroup(ctx, groupID, false) // includeOffsets = false
		if descErr != nil {
			slog.WarnContext(ctx, "Failed to describe group for monitor prompt", "group", groupID, "error", descErr)
			groupState = "Unknown (Error fetching)"
			memberCountStr = "Unknown"
		} else if descResult.ErrorCode != 0 {
			groupState = fmt.Sprintf("Error (%d: %s)", descResult.ErrorCode, descResult.ErrorMessage)
			memberCountStr = "N/A"
		} else {
			groupState = descResult.State
			memberCountStr = fmt.Sprintf("%d", len(descResult.Members))
		}
		content += fmt.Sprintf("## Current Status\n\n- **State:** %s\n- **Active Members:** %s\n\n", groupState, memberCountStr)

		content += "## Basic Status Commands\n\n"

		content += "### Describe Consumer Group (Includes State, Members, Assignments)\n"
		content += "```bash\n"
		content += fmt.Sprintf("kafka-consumer-groups --bootstrap-server %s \\\n", brokerList)
		content += fmt.Sprintf("  --describe --group %s\n", groupID)
		content += "```\n\n"

		// The --state and --members flags are often redundant if --describe is used, but kept for explicitness
		content += "### Check Current State Only\n"
		content += "```bash\n"
		content += fmt.Sprintf("kafka-consumer-groups --bootstrap-server %s \\\n", brokerList)
		content += fmt.Sprintf("  --describe --group %s --state\n", groupID)
		content += "```\n\n"

		content += "### Show Members in the Group Only\n"
		content += "```bash\n"
		content += fmt.Sprintf("kafka-consumer-groups --bootstrap-server %s \\\n", brokerList)
		content += fmt.Sprintf("  --describe --group %s --members\n", groupID)
		content += "```\n\n"

		if includeLagCommands {
			content += "## Monitor Consumer Lag\n\n"
			content += "The `--describe` command shown above also includes lag information per partition.\n\n"
			content += "```bash\n"
			content += "# Example focusing on lag output interpretation\n"
			content += fmt.Sprintf("kafka-consumer-groups --bootstrap-server %s --describe --group %s\n", brokerList, groupID)
			content += "# Look for columns: CURRENT-OFFSET, LOG-END-OFFSET, LAG\n"
			content += "```\n\n"
			content += "Alternatively, use the dedicated lag report prompt:\n"
			content += "`/kafka consumer-lag report`\n\n"
		}

		content += "## Troubleshooting Tips\n\n"
		content += "## Common Issues\n\n"
		content += "1. Group State Issues (e.g., stuck in `PreparingRebalance`):\n"
		content += "   - Check consumer logs for errors or long processing times (`max.poll.interval.ms`).\n"
		content += "   - Verify network connectivity between consumers and brokers.\n"
		content += "   - Ensure `session.timeout.ms` and `heartbeat.interval.ms` are appropriate.\n\n"
		content += "2. Growing Lag:\n"
		content += "   - Check if consumers can keep up with producer throughput.\n"
		content += "   - Consider increasing consumer parallelism (more consumers/partitions).\n"
		content += "   - Look for bottlenecks in consumer processing.\n\n"
		content += "3. Empty Consumer Groups:\n"
		content += "   - Verify consumer application is running and healthy.\n"
		content += "   - Check for connectivity issues or authentication failures.\n\n"

		result := &mcp.GetPromptResult{
			Description: "Kafka Consumer Group Monitoring Guide",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: content,
					},
				},
			},
		}

		return result, nil
	})

	// --- Kafka Troubleshooting Prompt --- (No data fetching needed, keeping as is)
	troubleshootingPrompt := mcp.Prompt{
		Name:        "kafka_troubleshooting",
		Description: "Provides a general troubleshooting guide for common Kafka issues",
		Arguments: []mcp.PromptArgument{
			{
				Name:        "issue_type",
				Description: "Type of issue to troubleshoot (producer, consumer, broker, or general)",
				Required:    false,
			},
		},
	}

	s.AddPrompt(troubleshootingPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		issueType := "general" // Default
		if it, ok := req.Params.Arguments["issue_type"]; ok && it != "" {
			issueType = it
		}

		// Get broker list for command examples within helper functions
		var brokerList string
		brokers, err := kafkaClient.ListBrokers(ctx) // Corrected: Use ListBrokers
		if err != nil {
			slog.WarnContext(ctx, "Failed to get broker list for troubleshooting prompt", "error", err)
			brokerList = "[broker:port]" // Fallback placeholder
		} else {
			brokerList = strings.Join(brokers, ",")
		}

		// Build content based on issue type, passing brokerList to helpers
		var contentSection string

		switch issueType {
		case "producer":
			contentSection = buildProducerTroubleshootingContent(brokerList)
		case "consumer":
			contentSection = buildConsumerTroubleshootingContent(brokerList)
		case "broker":
			contentSection = buildBrokerTroubleshootingContent(brokerList)
		default: // "general"
			contentSection = buildGeneralTroubleshootingContent(brokerList)
		}

		// Use cases.Title for proper title casing
		titleCaser := cases.Title(language.English)
		content := fmt.Sprintf("# Kafka Troubleshooting Guide: %s\n\n", titleCaser.String(issueType))
		content += contentSection + "\n\n"
		content += buildCommonTroubleshootingSection() // This one doesn't need brokerList

		result := &mcp.GetPromptResult{
			Description: "Kafka Troubleshooting Guide",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: content,
					},
				},
			},
		}

		return result, nil
	})

	// Helper function to build the troubleshooting options prompt
	troubleshootOptionsPrompt := mcp.Prompt{
		Name:        "kafka_troubleshoot_options",
		Description: "Lists troubleshooting options for Kafka issues",
		Arguments:   []mcp.PromptArgument{},
	}

	s.AddPrompt(troubleshootOptionsPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		content := "# Kafka Troubleshooting Options\n\n"
		content += "What area would you like help troubleshooting?\n\n"
		content += "- **Producers**: `/kafka troubleshoot issue_type=producer`\n"
		content += "- **Consumers**: `/kafka troubleshoot issue_type=consumer`\n"
		content += "- **Brokers**: `/kafka troubleshoot issue_type=broker`\n"
		content += "- **General Issues**: `/kafka troubleshoot issue_type=general`\n\n"
		content += "For consumer lag specifically: `/kafka consumer-lag report`\n"

		return &mcp.GetPromptResult{
			Description: "Kafka Troubleshooting Options",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: content,
					},
				},
			},
		}, nil
	})
}

// Helper functions to build troubleshooting content for different issue types
// Modified helpers to accept and use brokerList

func buildProducerTroubleshootingContent(brokerList string) string {
	content := "## Producer Troubleshooting\n\n"
	content += "### Common Producer Issues\n\n"

	content += "1. **Unable to Connect to Brokers**\n"
	content += "   - Verify broker addresses are correct \n"
	content += "   - Check network connectivity to brokers\n"
	content += "   - Ensure security settings match broker requirements\n\n"

	content += "2. **Slow Message Production**\n"
	content += "   - Check for broker throttling (quota exceeded)\n"
	content += "   - Verify batch.size and linger.ms settings\n"
	content += "   - Monitor broker resource utilization\n\n"

	content += "3. **Messages Not Being Acknowledged**\n"
	content += "   - Check acks setting (0, 1, all)\n"
	content += "   - Verify min.insync.replicas on the broker\n"
	content += "   - Check for replication issues between brokers\n\n"

	content += "### Producer Monitoring Commands\n\n"
	content += "```bash\n"
	content += "# Check producer metrics with JMX (if enabled)\n"
	content += "# jconsole or other JMX tool\n\n"
	content += "# Monitor network traffic to brokers (example)\n"
	content += fmt.Sprintf("# tcpdump -n host %s and port 9092\n\n", brokerList)
	content += "# Check broker logs for producer client issues\n"
	content += "# tail -f /path/to/kafka/logs/server.log | grep -i producer\n"
	content += "```\n\n"

	content += "### Producer Configuration Best Practices\n\n"
	content += "- Use appropriate acks level (acks=all for critical data)\n"
	content += "- Configure reasonable retries and retry.backoff.ms\n"
	content += "- Consider enabling idempotence for exactly-once semantics\n"
	content += "- Use snappy or lz4 compression for better throughput\n"

	return content
}

func buildConsumerTroubleshootingContent(brokerList string) string {
	content := "## Consumer Troubleshooting\n\n"
	content += "### Common Consumer Issues\n\n"

	content += "1. **Consumer Group Rebalancing Too Frequently**\n"
	content += "   - Increase session.timeout.ms and heartbeat.interval.ms\n"
	content += "   - Check for network issues between consumers and brokers\n"
	content += "   - Monitor consumer resource usage for slowdowns\n\n"

	content += "2. **High Consumer Lag**\n"
	content += "   - Increase consumer parallelism (more consumers or partitions)\n"
	content += "   - Check for slow message processing in the consumer\n"
	content += "   - Verify max.poll.records and fetch.max.bytes settings\n\n"

	content += "3. **Consumers Not Receiving Messages**\n"
	content += "   - Verify the topic exists and has messages\n"
	content += "   - Check consumer group subscription \n"
	content += "   - Validate consumer offset management (auto.offset.reset)\n\n"

	content += "### Consumer Monitoring Commands\n\n"
	content += "```bash\n"
	content += "# Check consumer group lag\n"
	content += fmt.Sprintf("kafka-consumer-groups --bootstrap-server %s --describe --group [group-id]\n\n", brokerList)
	content += "# Check consumer application log files\n"
	content += "# tail -f /path/to/consumer-application.log | grep -i error\n\n"
	content += "# Reset consumer offsets for testing (use with caution!)\n"
	content += fmt.Sprintf("kafka-consumer-groups --bootstrap-server %s --group [group-id] --topic [topic] --reset-offsets --to-earliest --execute\n", brokerList)
	content += "```\n\n"

	content += "### Consumer Configuration Best Practices\n\n"
	content += "- Configure appropriate auto.offset.reset (earliest or latest)\n"
	content += "- Set reasonable max.poll.interval.ms for your processing requirements\n"
	content += "- Use enable.auto.commit=false for manual offset control when needed\n"
	content += "- Consider fetch.max.bytes to limit memory pressure\n"

	return content
}

func buildBrokerTroubleshootingContent(brokerList string) string {
	content := "## Broker Troubleshooting\n\n"
	content += "### Common Broker Issues\n\n"

	content += "1. **Under-replicated Partitions**\n"
	content += "   - Check disk space on brokers\n"
	content += "   - Verify network connectivity between brokers\n"
	content += "   - Check broker logs for failed replication attempts\n\n"

	content += "2. **High CPU/Memory Usage**\n"
	content += "   - Monitor garbage collection on brokers (JVM settings)\n"
	content += "   - Check message sizes and throughput\n"
	content += "   - Review thread pool configurations\n\n"

	content += "3. **Offline Partitions**\n"
	content += "   - Check if broker is offline or unreachable\n"
	content += "   - Verify controller can communicate with all brokers\n"
	content += "   - Check ZooKeeper connection and state\n\n"

	content += "### Broker Monitoring Commands\n\n"
	content += "```bash\n"
	content += "# Check server.properties config on a broker\n"
	content += "# cat /path/to/kafka/config/server.properties\n\n"
	content += "# Check broker logs\n"
	content += "# tail -f /path/to/kafka/logs/server.log\n\n"
	content += "# Monitor JVM heap usage (requires PID)\n"
	content += "# jstat -gcutil [kafka-pid] 1000\n\n"
	content += "# Check under-replicated partitions\n"
	content += fmt.Sprintf("kafka-topics --bootstrap-server %s --describe --under-replicated-partitions\n\n", brokerList)
	content += "# Check controller logs (on the controller broker)\n"
	content += "# grep -i controller /path/to/kafka/logs/server.log\n"
	content += "```\n\n"

	content += "### Broker Configuration Best Practices\n\n"
	content += "- Configure appropriate replication factor (at least 3 for production)\n"
	content += "- Set min.insync.replicas=2 for critical topics\n"
	content += "- Tune num.io.threads and num.network.threads for your workload\n"
	content += "- Properly size JVM heap based on broker memory\n"

	return content
}

func buildGeneralTroubleshootingContent(brokerList string) string {
	content := "## General Kafka Troubleshooting\n\n"
	content += "### Health Check Commands\n\n"

	content += "```bash\n"
	content += "# Check if ZooKeeper is running (if used, replace host/port)\n"
	content += "# echo ruok | nc localhost 2181\n"
	content += "# echo stat | nc localhost 2181\n\n"
	content += "# List Kafka topics\n"
	content += fmt.Sprintf("kafka-topics --bootstrap-server %s --list\n\n", brokerList)
	content += "# Check broker information\n"
	content += fmt.Sprintf("kafka-broker-api-versions --bootstrap-server %s\n\n", brokerList)
	content += "# Check cluster controller (via ZK, if applicable)\n"
	content += "# zookeeper-shell localhost:2181 get /controller\n\n"
	content += "# Check under-replicated partitions\n"
	content += fmt.Sprintf("kafka-topics --bootstrap-server %s --describe --under-replicated-partitions\n", brokerList)
	content += "```\n\n"

	return content
}

func buildCommonTroubleshootingSection() string {
	content := "## Common Troubleshooting Tips for All Kafka Components\n\n"

	content += "### General Diagnostic Steps\n\n"
	content += "1. **Verify Connectivity**\n"
	content += "   - Ensure network connectivity between all components\n"
	content += "   - Check firewall and security group settings\n"
	content += "   - Verify DNS resolution is working correctly\n\n"

	content += "2. **Check Authentication and Authorization**\n"
	content += "   - Verify SSL/TLS certificates if used\n"
	content += "   - Check SASL credentials and configuration\n"
	content += "   - Verify ACLs allow required operations\n\n"

	content += "3. **Review Logging Configuration**\n"
	content += "   - Increase log levels for troubleshooting (e.g., DEBUG or TRACE)\n"
	content += "   - Check application logs, broker logs, and client logs\n"
	content += "   - Look for ERROR and WARN level messages\n\n"

	content += "### Monitoring and Metrics\n\n"
	content += "- Use JMX metrics to monitor Kafka performance\n"
	content += "- Set up alerts for critical conditions:\n"
	content += "  - Under-replicated partitions\n"
	content += "  - High consumer lag\n"
	content += "  - Broker resource utilization\n"
	content += "  - Connection failures\n\n"

	content += "### Testing Tools\n\n"
	content += "- kafka-console-producer/consumer for basic functionality tests\n"
	content += "- Use Conduktor, Kafka Tool, or other GUI tools for visual inspection\n"
	content += "- Consider load testing with Kafka performance tools for production validation\n"

	return content
}
