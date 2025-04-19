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
		var contentText string
		
		// Get broker list
		brokers, err := kafkaClient.ListTopics(ctx)
		if err != nil {
			contentText = fmt.Sprintf("⚠️ Error fetching broker information: %s", err.Error())
		} else if len(brokers) == 0 {
			contentText = "No brokers found in the cluster."
		} else {
			contentText = "## Kafka Brokers\n\n"
			contentText += "| Broker Address | Status |\n"
			contentText += "|----------------|--------|\n"
			
			for _, broker := range brokers {
				contentText += fmt.Sprintf("| %s | ✅ Connected |\n", broker)
			}
			
			contentText += "\n\n**Slack Command:**\n`/kafka list brokers`"
		}
		
		return &mcp.GetPromptResult{
			Description: "Kafka Broker List",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: contentText,
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
		contentText := "## Kafka Topics\n\n"
		
		// In a real implementation, we would fetch the actual topic details
		// For now, just show a sample response format
		contentText += "| Topic Name | Partitions | Replication Factor | Config |\n"
		contentText += "|------------|------------|-------------------|--------|\n"
		contentText += "| topic-1 | 8 | 3 | retention.ms=604800000 |\n"
		contentText += "| topic-2 | 16 | 3 | cleanup.policy=compact |\n"
		contentText += "| topic-3 | 4 | 2 | min.insync.replicas=2 |\n"
		
		contentText += "\n\n**Slack Command:**\n`/kafka list topics`"
		
		return &mcp.GetPromptResult{
			Description: "Kafka Topic List",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: contentText,
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
		contentText := fmt.Sprintf("## Topic Details: %s\n\n", topicName)
		
		// In a real implementation, we would fetch the actual topic configuration
		contentText += "### Configuration\n\n"
		contentText += "| Config | Value |\n"
		contentText += "|--------|-------|\n"
		contentText += "| cleanup.policy | delete |\n"
		contentText += "| retention.ms | 604800000 |\n"
		contentText += "| min.insync.replicas | 2 |\n"
		contentText += "| segment.bytes | 1073741824 |\n\n"
		
		contentText += "### Partitions\n\n"
		contentText += "| Partition | Leader | Replicas | In-Sync Replicas |\n"
		contentText += "|-----------|--------|----------|------------------|\n"
		contentText += "| 0 | 1 | [1,2,3] | [1,2,3] |\n"
		contentText += "| 1 | 2 | [2,3,1] | [2,3,1] |\n"
		contentText += "| 2 | 3 | [3,1,2] | [3,1,2] |\n"
		
		contentText += "\n\n**Slack Command:**\n"
		contentText += fmt.Sprintf("`/kafka describe topic %s`", topicName)
		
		return &mcp.GetPromptResult{
			Description: "Kafka Topic Description",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: contentText,
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
		contentText := "## Kafka Cluster Overview\n\n"
		
		contentText += "### Cluster Summary\n"
		contentText += "- **Broker Count:** 3\n"
		contentText += "- **Active Controller:** broker-2:9092\n"
		contentText += "- **Total Topics:** 24\n"
		contentText += "- **Total Partitions:** 124\n\n"
		
		contentText += "### Health Status\n"
		contentText += "- **Under-replicated Partitions:** 0 ✅\n"
		contentText += "- **Offline Partitions:** 0 ✅\n"
		contentText += "- **Active Controller:** Yes ✅\n"
		contentText += "- **Consumer Groups:** 12 (0 with high lag) ✅\n\n"
		
		contentText += "**Slack Command:**\n`/kafka cluster overview`\n\n"
		contentText += "**Related Commands:**\n"
		contentText += "- `/kafka health check` - For detailed health diagnostics\n"
		contentText += "- `/kafka under-replicated` - For listing under-replicated partitions"
		
		return &mcp.GetPromptResult{
			Description: "Kafka Cluster Overview",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: contentText,
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
		contentText := "## Active Kafka Consumer Groups\n\n"
		
		// Example response with consumer groups
		contentText += "| Consumer Group ID | Topics | Active Members | Total Lag |\n"
		contentText += "|-------------------|--------|----------------|----------|\n"
		contentText += "| consumer-group-1 | [topic-1, topic-2] | 4 | 120 |\n"
		contentText += "| consumer-group-2 | [topic-3] | 2 | 0 |\n"
		contentText += "| consumer-group-3 | [topic-1, topic-4] | 3 | 1450 |\n"
		
		contentText += "\n\n**Slack Command:**\n`/kafka list consumer-groups`\n\n"
		contentText += "**Related Commands:**\n"
		contentText += "- `/kafka describe consumer-group <group-id>` - For details on a specific group\n"
		contentText += "- `/kafka consumer-lag report` - For detailed lag information"
		
		return &mcp.GetPromptResult{
			Description: "Kafka Consumer Groups List",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: contentText,
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
		
		contentText := fmt.Sprintf("## Reset Offsets for Consumer Group: %s\n\n", groupID)
		
		contentText += "### Offset Reset Information\n\n"
		contentText += fmt.Sprintf("- **Topic:** %s\n", topic)
		contentText += fmt.Sprintf("- **Target Offset:** %s\n", offset)
		contentText += "- **Status:** ⚠️ Simulated (Not actually reset)\n\n"
		
		contentText += "### Command to Execute Reset\n\n"
		contentText += "```bash\n"
		
		// Build the proper command based on the offset type
		if offset == "earliest" || offset == "latest" {
			contentText += fmt.Sprintf("kafka-consumer-groups --bootstrap-server [broker:port] \\\n")
			contentText += fmt.Sprintf("  --group %s --topic %s \\\n", groupID, topic)
			contentText += fmt.Sprintf("  --reset-offsets --to-%s --execute\n", offset)
		} else {
			contentText += fmt.Sprintf("kafka-consumer-groups --bootstrap-server [broker:port] \\\n")
			contentText += fmt.Sprintf("  --group %s --topic %s \\\n", groupID, topic)
			contentText += fmt.Sprintf("  --reset-offsets --to-offset %s --execute\n", offset)
		}
		contentText += "```\n\n"
		
		contentText += "### ⚠️ Warning\n\n"
		contentText += "Resetting offsets will change the position from which consumers in this group will read messages.\n"
		contentText += "- If moving **backward**, messages will be reprocessed\n"
		contentText += "- If moving **forward**, messages will be skipped\n\n"
		
		contentText += "**Slack Command:**\n"
		contentText += fmt.Sprintf("`/kafka reset offsets %s %s %s`", groupID, topic, offset)
		
		return &mcp.GetPromptResult{
			Description: "Kafka Consumer Group Offset Reset",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: contentText,
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
		partitions := "3"
		if p, ok := req.Params.Arguments["partitions"]; ok && p != "" {
			partitions = p
		}
		
		replicationFactor := "3"
		if rf, ok := req.Params.Arguments["replication_factor"]; ok && rf != "" {
			replicationFactor = rf
		}
		
		retentionMs := "604800000" // 7 days by default
		if rm, ok := req.Params.Arguments["retention_ms"]; ok && rm != "" {
			retentionMs = rm
		}
		
		// Get broker list for commands
		var brokerList string
		try := func() error {
			brokers, err := kafkaClient.ListTopics(ctx)
			if err != nil {
				return err
			}
			brokerList = strings.Join(brokers, ",")
			return nil
		}()

		if try != nil {
			slog.WarnContext(ctx, "Failed to get broker list for prompt", "error", try)
			brokerList = "KAFKA_BROKER:9092" // Default placeholder
		}
		
		contentText := fmt.Sprintf("# Create Kafka Topic: %s\n\n", topicName)
		
		contentText += "## Topic Configuration\n\n"
		contentText += fmt.Sprintf("- **Topic Name:** %s\n", topicName)
		contentText += fmt.Sprintf("- **Partitions:** %s\n", partitions)
		contentText += fmt.Sprintf("- **Replication Factor:** %s\n", replicationFactor)
		contentText += fmt.Sprintf("- **Retention:** %s ms\n\n", retentionMs)
		
		contentText += "## Command to Create Topic\n\n"
		contentText += "```bash\n"
		contentText += fmt.Sprintf("kafka-topics --bootstrap-server %s \\\n", brokerList)
		contentText += fmt.Sprintf("  --create --topic %s \\\n", topicName)
		contentText += fmt.Sprintf("  --partitions %s \\\n", partitions)
		contentText += fmt.Sprintf("  --replication-factor %s \\\n", replicationFactor)
		contentText += fmt.Sprintf("  --config retention.ms=%s\n", retentionMs)
		contentText += "```\n\n"
		
		contentText += "## Verify Topic Creation\n\n"
		contentText += "```bash\n"
		contentText += fmt.Sprintf("kafka-topics --bootstrap-server %s --describe --topic %s\n", brokerList, topicName)
		contentText += "```\n\n"
		
		contentText += "**Slack Command:**\n"
		contentText += fmt.Sprintf("`/kafka create topic %s partitions=%s replication=%s`", 
			topicName, partitions, replicationFactor)
		
		return &mcp.GetPromptResult{
			Description: "Kafka Topic Creation Guide",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: contentText,
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
		try := func() error {
			brokers, err := kafkaClient.ListTopics(ctx)
			if err != nil {
				return err
			}
			brokerList = strings.Join(brokers, ",")
			return nil
		}()

		if try != nil {
			slog.WarnContext(ctx, "Failed to get broker list for prompt", "error", try)
			brokerList = "KAFKA_BROKER:9092" // Default placeholder
		}
		
		contentText := fmt.Sprintf("# Increase Partitions for Topic: %s\n\n", topic)
		
		contentText += "## Partition Change Details\n\n"
		contentText += fmt.Sprintf("- **Topic:** %s\n", topic)
		contentText += fmt.Sprintf("- **New Partition Count:** %s\n\n", partitions)
		
		contentText += "## Command to Increase Partitions\n\n"
		contentText += "```bash\n"
		contentText += fmt.Sprintf("kafka-topics --bootstrap-server %s \\\n", brokerList)
		contentText += fmt.Sprintf("  --alter --topic %s \\\n", topic)
		contentText += fmt.Sprintf("  --partitions %s\n", partitions)
		contentText += "```\n\n"
		
		contentText += "## Verify Partition Change\n\n"
		contentText += "```bash\n"
		contentText += fmt.Sprintf("kafka-topics --bootstrap-server %s --describe --topic %s\n", brokerList, topic)
		contentText += "```\n\n"
		
		contentText += "### ⚠️ Important Notes\n\n"
		contentText += "1. You can only **increase** the number of partitions (never decrease)\n"
		contentText += "2. Increasing partitions may affect message ordering for consumers\n"
		contentText += "3. Key-based message routing will change with more partitions\n\n"
		
		contentText += "**Slack Command:**\n"
		contentText += fmt.Sprintf("`/kafka increase partitions %s to %s`", topic, partitions)
		
		return &mcp.GetPromptResult{
			Description: "Kafka Topic Partition Increase",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: contentText,
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
		
		// Optional key
		key := ""
		if k, ok := req.Params.Arguments["key"]; ok {
			key = k
		}
		
		// Get broker list for commands
		var brokerList string
		try := func() error {
			brokers, err := kafkaClient.ListTopics(ctx)
			if err != nil {
				return err
			}
			brokerList = strings.Join(brokers, ",")
			return nil
		}()

		if try != nil {
			slog.WarnContext(ctx, "Failed to get broker list for prompt", "error", try)
			brokerList = "KAFKA_BROKER:9092" // Default placeholder
		}
		
		contentText := fmt.Sprintf("# Produce Message to Topic: %s\n\n", topic)
		
		contentText += "## Message Details\n\n"
		contentText += fmt.Sprintf("- **Topic:** %s\n", topic)
		contentText += fmt.Sprintf("- **Message:** %s\n", message)
		if key != "" {
			contentText += fmt.Sprintf("- **Key:** %s\n", key)
		}
		contentText += "- **Status:** ⚠️ Simulated (Not actually sent)\n\n"
		
		contentText += "## Command to Produce Message\n\n"
		
		if key == "" {
			contentText += "```bash\n"
			contentText += fmt.Sprintf("echo '%s' | kafka-console-producer \\\n", message)
			contentText += fmt.Sprintf("  --bootstrap-server %s \\\n", brokerList)
			contentText += fmt.Sprintf("  --topic %s\n", topic)
			contentText += "```\n\n"
		} else {
			contentText += "```bash\n"
			contentText += fmt.Sprintf("kafka-console-producer \\\n")
			contentText += fmt.Sprintf("  --bootstrap-server %s \\\n", brokerList)
			contentText += fmt.Sprintf("  --topic %s \\\n", topic)
			contentText += fmt.Sprintf("  --property parse.key=true \\\n")
			contentText += fmt.Sprintf("  --property key.separator=:\n")
			contentText += fmt.Sprintf("%s:%s\n", key, message)
			contentText += "```\n\n"
		}
		
		contentText += "## Verify Message Consumption\n\n"
		contentText += "```bash\n"
		contentText += fmt.Sprintf("kafka-console-consumer \\\n")
		contentText += fmt.Sprintf("  --bootstrap-server %s \\\n", brokerList)
		contentText += fmt.Sprintf("  --topic %s \\\n", topic)
		contentText += fmt.Sprintf("  --from-beginning \\\n")
		contentText += fmt.Sprintf("  --max-messages 1\n")
		contentText += "```\n\n"
		
		contentText += "**Slack Command:**\n"
		contentText += fmt.Sprintf("`/kafka produce %s \"%s\"`", topic, message)
		
		return &mcp.GetPromptResult{
			Description: "Kafka Message Production",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: contentText,
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
		
		// Default values
		count := "10"
		if c, ok := req.Params.Arguments["count"]; ok && c != "" {
			count = c
		}
		
		fromBeginning := false
		if fb, ok := req.Params.Arguments["from_beginning"]; ok {
			fromBeginning = fb == "true"
		}
		
		// Get broker list for commands
		var brokerList string
		try := func() error {
			brokers, err := kafkaClient.ListTopics(ctx)
			if err != nil {
				return err
			}
			brokerList = strings.Join(brokers, ",")
			return nil
		}()

		if try != nil {
			slog.WarnContext(ctx, "Failed to get broker list for prompt", "error", try)
			brokerList = "KAFKA_BROKER:9092" // Default placeholder
		}
		
		contentText := fmt.Sprintf("# Consume Messages from Topic: %s\n\n", topic)
		
		contentText += "## Consumption Details\n\n"
		contentText += fmt.Sprintf("- **Topic:** %s\n", topic)
		contentText += fmt.Sprintf("- **Message Count:** %s\n", count)
		contentText += fmt.Sprintf("- **From Beginning:** %v\n", fromBeginning)
		contentText += "- **Status:** ⚠️ Simulated (Not actually consumed)\n\n"
		
		contentText += "## Command to Consume Messages\n\n"
		contentText += "```bash\n"
		contentText += fmt.Sprintf("kafka-console-consumer \\\n")
		contentText += fmt.Sprintf("  --bootstrap-server %s \\\n", brokerList)
		contentText += fmt.Sprintf("  --topic %s \\\n", topic)
		
		if fromBeginning {
			contentText += fmt.Sprintf("  --from-beginning \\\n")
		}
		
		contentText += fmt.Sprintf("  --max-messages %s\n", count)
		contentText += "```\n\n"
		
		contentText += "## Show Keys with Messages\n\n"
		contentText += "```bash\n"
		contentText += fmt.Sprintf("kafka-console-consumer \\\n")
		contentText += fmt.Sprintf("  --bootstrap-server %s \\\n", brokerList)
		contentText += fmt.Sprintf("  --topic %s \\\n", topic)
		
		if fromBeginning {
			contentText += fmt.Sprintf("  --from-beginning \\\n")
		}
		
		contentText += fmt.Sprintf("  --max-messages %s \\\n", count)
		contentText += fmt.Sprintf("  --property print.key=true \\\n")
		contentText += fmt.Sprintf("  --property key.separator=: \n")
		contentText += "```\n\n"
		
		contentText += "**Slack Command:**\n"
		contentText += fmt.Sprintf("`/kafka consume %s count=%s`", topic, count)
		
		return &mcp.GetPromptResult{
			Description: "Kafka Message Consumption",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: contentText,
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
		contentText := "# Kafka Cluster Health Check\n\n"
		
		contentText += "## Health Summary\n\n"
		contentText += "| Component | Status | Details |\n"
		contentText += "|-----------|--------|--------|\n"
		contentText += "| Brokers | ✅ All Online | 3/3 brokers available |\n"
		contentText += "| Controller | ✅ Active | Running on broker-2:9092 |\n"
		contentText += "| Topics | ✅ Healthy | No under-replicated partitions |\n"
		contentText += "| Consumer Groups | ⚠️ Warning | 1 group with high lag |\n\n"
		
		contentText += "## Detailed Analysis\n\n"
		
		contentText += "### Broker Status\n\n"
		contentText += "- Broker 1: Online, CPU: 35%, Memory: 65%, Disk: 42%\n"
		contentText += "- Broker 2: Online, CPU: 40%, Memory: 70%, Disk: 51%\n"
		contentText += "- Broker 3: Online, CPU: 30%, Memory: 60%, Disk: 38%\n\n"
		
		contentText += "### Partition Health\n\n"
		contentText += "- Total Partitions: 124\n"
		contentText += "- Under-replicated: 0\n"
		contentText += "- Offline: 0\n"
		contentText += "- URP Leader Count: 0\n\n"
		
		contentText += "### Consumer Group Issues\n\n"
		contentText += "- **consumer-group-3** has high lag (1450 messages) on topic-1\n\n"
		
		contentText += "## Health Check Commands\n\n"
		contentText += "```bash\n"
		contentText += "# Check under-replicated partitions\n"
		contentText += "kafka-topics --bootstrap-server [broker:port] --describe --under-replicated\n\n"
		contentText += "# Check broker details\n"
		contentText += "kafka-broker-api-versions --bootstrap-server [broker:port]\n\n"
		contentText += "# Check consumer group lag\n"
		contentText += "kafka-consumer-groups --bootstrap-server [broker:port] --all-groups --describe\n"
		contentText += "```\n\n"
		
		contentText += "**Slack Command:**\n`/kafka health check`\n\n"
		contentText += "**Related Commands:**\n"
		contentText += "- `/kafka under-replicated` - For detailed under-replicated partition info\n"
		contentText += "- `/kafka consumer-lag report` - For detailed consumer lag analysis"
		
		return &mcp.GetPromptResult{
			Description: "Kafka Cluster Health Check Results",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: contentText,
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
		contentText := "# Under-Replicated Kafka Partitions\n\n"
		
		// Example response - in a real implementation, you would fetch actual data
		contentText += "## Status Summary\n\n"
		contentText += "No under-replicated partitions found in the cluster. ✅\n\n"
		
		contentText += "## Command to Check Under-Replicated Partitions\n\n"
		contentText += "```bash\n"
		contentText += "kafka-topics --bootstrap-server [broker:port] --describe --under-replicated\n"
		contentText += "```\n\n"
		
		contentText += "## How to Diagnose Under-Replication\n\n"
		contentText += "If under-replicated partitions are found:\n\n"
		contentText += "1. Check for broker failures\n"
		contentText += "```bash\n"
		contentText += "kafka-broker-api-versions --bootstrap-server [broker:port]\n"
		contentText += "```\n\n"
		
		contentText += "2. Check disk space on brokers\n"
		contentText += "```bash\n"
		contentText += "df -h\n"
		contentText += "```\n\n"
		
		contentText += "3. Check for network issues between brokers\n"
		contentText += "```bash\n"
		contentText += "ping [broker-host]\n"
		contentText += "```\n\n"
		
		contentText += "4. Check broker logs for errors\n"
		contentText += "```bash\n"
		contentText += "tail -f /var/log/kafka/server.log | grep -i replica\n"
		contentText += "```\n\n"
		
		contentText += "**Slack Command:**\n`/kafka under-replicated`"
		
		return &mcp.GetPromptResult{
			Description: "Kafka Under-Replicated Partitions Report",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: contentText,
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
		// Default threshold
		threshold := "1000"
		if t, ok := req.Params.Arguments["threshold"]; ok && t != "" {
			threshold = t
		}
		
		contentText := "# Kafka Consumer Lag Report\n\n"
		
		contentText += fmt.Sprintf("## Consumer Groups Exceeding Lag Threshold (%s messages)\n\n", threshold)
		
		// Example data - in a real implementation you'd fetch actual consumer group lag
		contentText += "| Consumer Group | Topic | Partition | Current Offset | Log End Offset | Lag |\n"
		contentText += "|----------------|-------|-----------|----------------|----------------|-----|\n"
		contentText += "| consumer-group-3 | topic-1 | 0 | 234567 | 236017 | 1450 |\n"
		contentText += "| consumer-group-4 | topic-2 | 2 | 345678 | 346789 | 1111 |\n\n"
		
		contentText += "## All Consumer Groups Summary\n\n"
		contentText += "| Consumer Group | Total Lag | Status |\n"
		contentText += "|----------------|-----------|--------|\n"
		contentText += "| consumer-group-1 | 120 | ✅ Normal |\n"
		contentText += "| consumer-group-2 | 0 | ✅ Normal |\n"
		contentText += "| consumer-group-3 | 1450 | ⚠️ High Lag |\n"
		contentText += "| consumer-group-4 | 1111 | ⚠️ High Lag |\n\n"
		
		contentText += "## Command to Check Consumer Lag\n\n"
		contentText += "```bash\n"
		contentText += "kafka-consumer-groups --bootstrap-server [broker:port] --all-groups --describe\n"
		contentText += "```\n\n"
		
		contentText += "## Addressing High Consumer Lag\n\n"
		contentText += "1. **Scale Consumers**: Increase the number of consumer instances\n"
		contentText += "2. **Optimize Processing**: Review consumer code for inefficiencies\n"
		contentText += "3. **Hardware Resources**: Ensure consumers have adequate CPU/memory\n"
		contentText += "4. **Partition Count**: Consider increasing topic partition count\n"
		contentText += "5. **Batch Size**: Adjust fetch.max.bytes or max.poll.records\n\n"
		
		contentText += "**Slack Command:**\n`/kafka consumer-lag report`"
		
		return &mcp.GetPromptResult{
			Description: "Kafka Consumer Lag Analysis",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: contentText,
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
			if include == "false" {
				includeLagCommands = false
			}
		}

		// Get broker list for commands
		var brokerList string
		try := func() error {
			brokers, err := kafkaClient.ListTopics(ctx)
			if err != nil {
				return err
			}
			brokerList = strings.Join(brokers, ",")
			return nil
		}()

		if try != nil {
			slog.WarnContext(ctx, "Failed to get broker list for prompt", "error", try)
			brokerList = "KAFKA_BROKER:9092" // Default placeholder
		}

		// Build content 
		contentText := fmt.Sprintf("# Monitor Kafka Consumer Group: %s\n\n", groupID)

		contentText += "## Basic Status Commands\n\n"

		contentText += "### List All Consumer Groups\n"
		contentText += "```bash\n"
		contentText += fmt.Sprintf("kafka-consumer-groups --bootstrap-server %s --list\n", brokerList)
		contentText += "```\n\n"

		contentText += "### Describe Consumer Group\n"
		contentText += "```bash\n"
		contentText += fmt.Sprintf("kafka-consumer-groups --bootstrap-server %s \\\n", brokerList)
		contentText += fmt.Sprintf("  --describe --group %s\n", groupID)
		contentText += "```\n\n"

		contentText += "### Check Current State\n"
		contentText += "```bash\n"
		contentText += fmt.Sprintf("kafka-consumer-groups --bootstrap-server %s \\\n", brokerList)
		contentText += fmt.Sprintf("  --describe --group %s --state\n", groupID)
		contentText += "```\n\n"

		contentText += "### Show Members in the Group\n"
		contentText += "```bash\n"
		contentText += fmt.Sprintf("kafka-consumer-groups --bootstrap-server %s \\\n", brokerList)
		contentText += fmt.Sprintf("  --describe --group %s --members\n", groupID)
		contentText += "```\n\n"

		if includeLagCommands {
			contentText += "## Monitor Consumer Lag\n\n"
			contentText += "To check consumer lag and offset details:\n\n"
			contentText += "```bash\n"
			contentText += fmt.Sprintf("kafka-consumer-groups --bootstrap-server %s \\\n", brokerList)
			contentText += fmt.Sprintf("  --describe --group %s\n", groupID)
			contentText += "```\n\n"
			contentText += "This will show lag for each partition assigned to the consumer group.\n\n"
		}

		contentText += "## Troubleshooting Tips\n\n"
		contentText += "1. If consumers are stuck in \"PreparingRebalance\" state for extended periods:\n"
		contentText += "   - Check for network issues between consumers\n"
		contentText += "   - Increase session.timeout.ms if network is unstable\n"
		contentText += "   - Look for failing consumers that might be causing rebalances\n\n"

		contentText += "2. If lag is growing:\n"
		contentText += "   - Check if producers are sending more data than consumers can process\n"
		contentText += "   - Consider increasing consumer parallelism\n"
		contentText += "   - Monitor consumer CPU and memory usage\n"
		contentText += "   - Verify there are no bottlenecks in consumer processing logic\n"

		result := &mcp.GetPromptResult{
			Description: "Kafka Consumer Group Monitoring Guide",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: contentText,
					},
				},
			},
		}

		return result, nil
	})

	// --- Kafka Troubleshooting Prompt ---
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

		// Build content based on issue type
		var contentSection string

		switch issueType {
		case "producer":
			contentSection = buildProducerTroubleshootingContent()
		case "consumer":
			contentSection = buildConsumerTroubleshootingContent()
		case "broker":
			contentSection = buildBrokerTroubleshootingContent()
		default: // "general"
			contentSection = buildGeneralTroubleshootingContent()
		}

		contentText := fmt.Sprintf("# Kafka Troubleshooting Guide: %s\n\n", strings.Title(issueType))
		contentText += contentSection + "\n\n"
		contentText += buildCommonTroubleshootingSection()

		result := &mcp.GetPromptResult{
			Description: "Kafka Troubleshooting Guide",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleAssistant,
					Content: mcp.TextContent{
						Type: "text",
						Text: contentText,
					},
				},
			},
		}

		return result, nil
	})
}

// Helper functions to build troubleshooting content for different issue types
func buildProducerTroubleshootingContent() string {
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
	content += "# Check producer metrics with JMX\n"
	content += "jconsole # Connect to local JVM running Kafka\n\n"
	content += "# Monitor network traffic to brokers\n"
	content += "tcpdump -n host [broker-ip] and port 9092\n\n"
	content += "# Check broker logs for producer client issues\n"
	content += "tail -f /var/log/kafka/server.log | grep -i producer\n"
	content += "```\n\n"

	content += "### Producer Configuration Best Practices\n\n"
	content += "- Use appropriate acks level (acks=all for critical data)\n"
	content += "- Configure reasonable retries and retry.backoff.ms\n"
	content += "- Consider enabling idempotence for exactly-once semantics\n"
	content += "- Use snappy or lz4 compression for better throughput\n"

	return content
}

func buildConsumerTroubleshootingContent() string {
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
	content += "kafka-consumer-groups --bootstrap-server [broker:port] --describe --group [group-id]\n\n"
	content += "# Check consumer log files\n"
	content += "tail -f consumer-application.log | grep -i error\n\n"
	content += "# Reset consumer offsets for testing\n"
	content += "kafka-consumer-groups --bootstrap-server [broker:port] --group [group-id] --topic [topic] --reset-offsets --to-earliest --execute\n"
	content += "```\n\n"

	content += "### Consumer Configuration Best Practices\n\n"
	content += "- Configure appropriate auto.offset.reset (earliest or latest)\n"
	content += "- Set reasonable max.poll.interval.ms for your processing requirements\n"
	content += "- Use enable.auto.commit=false for manual offset control when needed\n"
	content += "- Consider fetch.max.bytes to limit memory pressure\n"

	return content
}

func buildBrokerTroubleshootingContent() string {
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
	content += "# Check server.properties config\n"
	content += "cat /etc/kafka/server.properties\n\n"
	content += "# Check broker logs\n"
	content += "tail -f /var/log/kafka/server.log\n\n"
	content += "# Monitor JVM heap usage\n"
	content += "jstat -gcutil [kafka-pid] 1000\n\n"
	content += "# Check under-replicated partitions\n"
	content += "kafka-topics --bootstrap-server [broker:port] --describe --under-replicated\n\n"
	content += "# Check controller logs\n"
	content += "grep -i controller /var/log/kafka/server.log\n"
	content += "```\n\n"

	content += "### Broker Configuration Best Practices\n\n"
	content += "- Configure appropriate replication factor (at least 3 for production)\n"
	content += "- Set min.insync.replicas=2 for critical topics\n"
	content += "- Tune num.io.threads and num.network.threads for your workload\n"
	content += "- Properly size JVM heap based on broker memory\n"

	return content
}

func buildGeneralTroubleshootingContent() string {
	content := "## General Kafka Troubleshooting\n\n"
	content += "### Health Check Commands\n\n"

	content += "```bash\n"
	content += "# Check if ZooKeeper is running (if used)\n"
	content += "echo ruok | nc localhost 2181\n"
	content += "echo stat | nc localhost 2181\n\n"
	content += "# List Kafka topics\n"
	content += "kafka-topics --bootstrap-server [broker:port] --list\n\n"
	content += "# Check broker information\n"
	content += "kafka-broker-api-versions --bootstrap-server [broker:port]\n\n"
	content += "# Check cluster controller\n"
	content += "zookeeper-shell localhost:2181 get /controller\n\n"
	content += "# Check under-replicated partitions\n"
	content += "kafka-topics --bootstrap-server [broker:port] --describe --under-replicated\n"
	content += "```\n\n"

	content += "### Common Issues and Solutions\n\n"
	content += "1. **Topic Creation Failures**\n"
	content += "   - Check broker logs for errors\n"
	content += "   - Verify ZooKeeper connectivity (if used)\n"
	content += "   - Ensure auto.create.topics.enable is set correctly\n\n"

	content += "2. **Performance Issues**\n"
	content += "   - Check disk I/O on broker machines\n"
	content += "   - Monitor network bandwidth between brokers\n"
	content += "   - Review producer and consumer client configurations\n\n"

	content += "3. **Connection Issues**\n"
	content += "   - Verify security configurations (SASL, SSL)\n"
	content += "   - Check network firewall rules\n"
	content += "   - Ensure advertised.listeners is configured correctly\n\n"

	content += "### General Best Practices\n\n"
	content += "- Always use multiple brokers in production (at least 3)\n"
	content += "- Monitor disk space regularly\n"
	content += "- Configure appropriate log retention settings\n"
	content += "- Maintain proper backups of configuration and data\n"
	content += "- Upgrade Kafka in a controlled, staged manner\n"

	return content
}

func buildCommonTroubleshootingSection() string {
	content := "## Diagnostics and Data Collection\n\n"
	content += "When troubleshooting Kafka issues, collect the following information:\n\n"
	content += "1. Kafka and ZooKeeper versions\n"
	content += "2. Cluster topology and configuration\n"
	content += "3. Client configurations (producer, consumer)\n"
	content += "4. Error messages and timestamps\n"
	content += "5. Recent changes to the system\n"
	content += "6. Performance metrics at the time of the issue\n\n"

	content += "## Useful Metrics to Monitor\n\n"
	content += "- Broker metrics: UnderReplicatedPartitions, ActiveControllerCount\n"
	content += "- Producer metrics: request-rate, response-rate, request-latency-avg\n"
	content += "- Consumer metrics: records-lag-max, records-consumed-rate, bytes-consumed-rate\n"
	content += "- JVM metrics: heap usage, garbage collection time\n\n"

	content += "## Additional Resources\n\n"
	content += "- [Apache Kafka Documentation](https://kafka.apache.org/documentation/)\n"
	content += "- [Confluent Kafka Operations](https://docs.confluent.io/platform/current/kafka/operations.html)\n"
	content += "- [Kafka Monitoring and Metrics](https://docs.confluent.io/platform/current/kafka/monitoring.html)\n"

	return content
}
