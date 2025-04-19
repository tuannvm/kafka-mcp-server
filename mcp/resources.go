package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tuannvm/kafka-mcp-server/kafka"
)

// RegisterResources defines and registers MCP resources with the server.
func RegisterResources(s *server.MCPServer, kafkaClient kafka.KafkaClient) {
	// --- 1. Direct Resources ---

	// Kafka Broker Log Resource
	kafkaBrokerLogResource := mcp.NewResource(
		"kafka_server_log",
		"Kafka Server Log",
		mcp.WithResourceDescription("Rolling broker activity log"),
		mcp.WithMIMEType("text/plain"),
	)

	s.AddResource(kafkaBrokerLogResource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		slog.InfoContext(ctx, "Executing kafka_server_log resource")

		// In a real implementation, this would read the actual log file
		// For now, returning simulated log data
		logData := "Simulated Kafka server log data\n"
		logData += "[2023-07-10 12:00:01,123] INFO [KafkaServer id=1] Started (kafka.server.KafkaServer)\n"
		logData += "[2023-07-10 12:00:02,456] INFO [SocketServer brokerId=1] Started data-plane acceptor and processor(s) for endpoint : ListenerName(PLAINTEXT) (kafka.network.SocketServer)\n"
		logData += "[2023-07-10 12:00:03,789] INFO [BrokerToControllerChannelManager broker=1 name=forwarding]: Recorded new controller, from now on will use node 1 (kafka.server.BrokerToControllerRequestThread)\n"

		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				Text: logData,
			},
		}, nil
	})

	// Broker Configuration Resource
	brokerConfigResource := mcp.NewResource(
		"broker_config",
		"Broker Configuration",
		mcp.WithResourceDescription("YAML-encoded Kafka broker settings"),
		mcp.WithMIMEType("application/x-yaml"),
	)

	s.AddResource(brokerConfigResource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		slog.InfoContext(ctx, "Executing broker_config resource")

		// This would typically fetch from the actual broker config
		// For now, returning a placeholder or sample config in YAML format
		yamlConfig := `# Kafka Broker Configuration
broker.id: 1
listeners: PLAINTEXT://localhost:9092
num.network.threads: 3
num.io.threads: 8
socket.send.buffer.bytes: 102400
socket.receive.buffer.bytes: 102400
socket.request.max.bytes: 104857600
log.dirs: /tmp/kafka-logs
num.partitions: 1
num.recovery.threads.per.data.dir: 1
offsets.topic.replication.factor: 1
transaction.state.log.replication.factor: 1
transaction.state.log.min.isr: 1
log.retention.hours: 168
log.segment.bytes: 1073741824
log.retention.check.interval.ms: 300000
zookeeper.connect: localhost:2181
zookeeper.connection.timeout.ms: 18000
group.initial.rebalance.delay.ms: 0
`

		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				Text: yamlConfig,
			},
		}, nil
	})

	// Topic Schemas Resource
	topicSchemasResource := mcp.NewResource(
		"topic_schemas",
		"All Topic Schemas",
		mcp.WithResourceDescription("JSON-schema definitions for all topics"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(topicSchemasResource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		slog.InfoContext(ctx, "Executing topic_schemas resource")

		// In a real implementation, this would fetch actual schema registry data
		// For now, returning sample schema data
		schemas := map[string]interface{}{
			"schemas": map[string]interface{}{
				"orders": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "integer",
						},
						"customer_id": map[string]interface{}{
							"type": "integer",
						},
						"amount": map[string]interface{}{
							"type": "number",
						},
						"status": map[string]interface{}{
							"type":    "string",
							"enum":    []string{"pending", "completed", "cancelled"},
							"default": "pending",
						},
					},
					"required": []string{"id", "customer_id", "amount"},
				},
				"users": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type": "integer",
						},
						"name": map[string]interface{}{
							"type": "string",
						},
						"email": map[string]interface{}{
							"type":   "string",
							"format": "email",
						},
					},
					"required": []string{"id", "name", "email"},
				},
			},
		}

		jsonData, err := json.Marshal(schemas)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to marshal schema data", "error", err)
			return nil, err
		}

		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				Text: string(jsonData),
			},
		}, nil
	})

	// Access Control Lists Resource
	aclResource := mcp.NewResource(
		"acl_definitions",
		"Access Control Lists",
		mcp.WithResourceDescription("Current ACL rules for cluster resources"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(aclResource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		slog.InfoContext(ctx, "Executing acl_definitions resource")

		// In a real implementation, this would fetch actual ACLs from Kafka
		// For now, returning sample data
		acls := []map[string]interface{}{
			{
				"principal":     "User:alice",
				"resource_type": "Topic",
				"resource_name": "orders",
				"pattern_type":  "LITERAL",
				"operation":     "READ",
				"permission":    "ALLOW",
			},
			{
				"principal":     "User:bob",
				"resource_type": "Group",
				"resource_name": "analytics-group",
				"pattern_type":  "LITERAL",
				"operation":     "READ",
				"permission":    "ALLOW",
			},
			{
				"principal":     "User:admin",
				"resource_type": "Cluster",
				"resource_name": "kafka-cluster",
				"pattern_type":  "LITERAL",
				"operation":     "ALL",
				"permission":    "ALLOW",
			},
		}

		jsonData, err := json.Marshal(acls)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to marshal ACL data", "error", err)
			return nil, err
		}

		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				Text: string(jsonData),
			},
		}, nil
	})

	// --- 2. Resource Templates ---

	// Topic Description Resource
	topicDescribeResource := mcp.NewResourceTemplate(
		"kafka-mcp://{cluster}/topics/{topic}/describe",
		"Describe Topic",
		mcp.WithTemplateDescription("Partition, replica, and ISR details for a topic"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(topicDescribeResource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Extract parameters from the URI
		params := extractParamsFromURI(req.Params.URI, "kafka-mcp://{cluster}/topics/{topic}/describe")

		topicName, ok := params["topic"]
		if !ok || topicName == "" {
			return nil, fmt.Errorf("topic parameter is required")
		}

		slog.InfoContext(ctx, "Executing topic_describe resource", "topic", topicName)

		metadata, err := kafkaClient.DescribeTopic(ctx, topicName)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to describe topic", "topic", topicName, "error", err)
			return nil, err
		}

		jsonData, err := json.Marshal(metadata)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to marshal topic metadata", "error", err)
			return nil, err
		}

		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				Text: string(jsonData),
			},
		}, nil
	})

	// Consumer Group Offsets Resource
	consumerGroupOffsetsResource := mcp.NewResourceTemplate(
		"kafka-mcp://{cluster}/consumerGroups/{groupId}/offsets",
		"Consumer Group Offsets",
		mcp.WithTemplateDescription("Current vs. end offsets per partition"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(consumerGroupOffsetsResource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Extract parameters from the URI
		params := extractParamsFromURI(req.Params.URI, "kafka-mcp://{cluster}/consumerGroups/{groupId}/offsets")

		groupID, ok := params["groupId"]
		if !ok || groupID == "" {
			return nil, fmt.Errorf("groupId parameter is required")
		}

		slog.InfoContext(ctx, "Executing consumer_group_offsets resource", "groupId", groupID)

		// Get consumer group details including offsets
		descResult, err := kafkaClient.DescribeConsumerGroup(ctx, groupID, true) // includeOffsets = true
		if err != nil {
			slog.ErrorContext(ctx, "Failed to describe consumer group", "groupId", groupID, "error", err)
			return nil, err
		}

		jsonData, err := json.Marshal(descResult)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to marshal consumer group data", "error", err)
			return nil, err
		}

		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				Text: string(jsonData),
			},
		}, nil
	})

	// Topic Messages Resource
	topicMessagesResource := mcp.NewResourceTemplate(
		"kafka-mcp://{cluster}/topics/{topic}/messages",
		"Fetch Topic Messages",
		mcp.WithTemplateDescription("Retrieve a batch of messages from a topic"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(topicMessagesResource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Extract parameters from the URI
		uriParams := extractParamsFromURI(req.Params.URI, "kafka-mcp://{cluster}/topics/{topic}/messages")

		topic, ok := uriParams["topic"]
		if !ok || topic == "" {
			return nil, fmt.Errorf("topic parameter is required")
		}

		// Parse query parameters (offset, count)
		queryParams := extractQueryParams(req.Params.URI)

		// Default values
		count := 10
		if countStr, ok := queryParams["count"]; ok && countStr != "" {
			fmt.Sscanf(countStr, "%d", &count)
			if count <= 0 {
				count = 1
			}
		}

		fromBeginning := false
		if offsetStr, ok := queryParams["offset"]; ok && offsetStr == "0" {
			fromBeginning = true
		}

		slog.InfoContext(ctx, "Executing topic_messages resource",
			"topic", topic,
			"count", count,
			"from_beginning", fromBeginning)

		// Consume messages from the topic
		messages, err := kafkaClient.ConsumeMessages(ctx, []string{topic}, count)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to consume messages", "topic", topic, "error", err)
			return nil, err
		}

		jsonData, err := json.Marshal(messages)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to marshal messages", "error", err)
			return nil, err
		}

		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				Text: string(jsonData),
			},
		}, nil
	})

	// Cluster Metrics Resource
	clusterMetricsResource := mcp.NewResourceTemplate(
		"kafka-mcp://{cluster}/metrics/{metricType}",
		"Cluster Metrics",
		mcp.WithTemplateDescription("Time-series data for CPU, memory, throughput, or latency"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(clusterMetricsResource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Extract parameters from the URI
		params := extractParamsFromURI(req.Params.URI, "kafka-mcp://{cluster}/metrics/{metricType}")

		metricType, ok := params["metricType"]
		if !ok || metricType == "" {
			return nil, fmt.Errorf("metricType parameter is required")
		}

		validTypes := map[string]bool{
			"cpu":        true,
			"memory":     true,
			"throughput": true,
			"latency":    true,
		}

		if !validTypes[metricType] {
			return nil, fmt.Errorf("invalid metric type: %s, valid types are: cpu, memory, throughput, latency", metricType)
		}

		slog.InfoContext(ctx, "Executing cluster_metrics resource", "metricType", metricType)

		// In a real implementation, this would fetch actual metrics from JMX, Prometheus, etc.
		// For now, returning sample data with a timestamp series
		now := time.Now()
		datapoints := make([]map[string]interface{}, 5)

		for i := 0; i < 5; i++ {
			timestamp := now.Add(time.Duration(-i) * time.Minute).Format(time.RFC3339)
			var value float64

			switch metricType {
			case "cpu":
				value = 0.2 + float64(i)*0.05
			case "memory":
				value = 70.0 - float64(i)*2.5
			case "throughput":
				value = 150.0 + float64(i)*10.0
			case "latency":
				value = 5.0 - float64(i)*0.3
				if value < 0 {
					value = 0.1
				}
			}

			datapoints[i] = map[string]interface{}{
				"timestamp": timestamp,
				"value":     value,
			}
		}

		metrics := map[string]interface{}{
			"metric_type": metricType,
			"interval":    "1m",
			"datapoints":  datapoints,
		}

		jsonData, err := json.Marshal(metrics)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to marshal metrics data", "error", err)
			return nil, err
		}

		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				Text: string(jsonData),
			},
		}, nil
	})

	// --- 3. Binary Resources ---

	// Dashboard Screenshot Resource
	dashboardScreenshotResource := mcp.NewResourceTemplate(
		"kafka-mcp://{cluster}/dashboards/broker/{brokerId}/screenshot.png",
		"Broker Metrics Dashboard",
		mcp.WithTemplateDescription("PNG snapshot of broker-level metrics view"),
		mcp.WithTemplateMIMEType("image/png"),
	)

	s.AddResourceTemplate(dashboardScreenshotResource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Extract parameters from the URI
		params := extractParamsFromURI(req.Params.URI, "kafka-mcp://{cluster}/dashboards/broker/{brokerId}/screenshot.png")

		brokerId, ok := params["brokerId"]
		if !ok || brokerId == "" {
			return nil, fmt.Errorf("brokerId parameter is required")
		}

		slog.InfoContext(ctx, "Executing dashboard_screenshot resource", "brokerId", brokerId)

		// In a real implementation, this would generate or fetch an actual PNG image
		// For now, returning a small, base64-encoded PNG as a placeholder
		// This is a 1x1 transparent PNG pixel
		pngData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="

		return []mcp.ResourceContents{
			&mcp.BlobResourceContents{
				Blob:     pngData,
				MIMEType: "image/png",
			},
		}, nil
	})

	// Partition Distribution Chart Resource
	partitionDistributionResource := mcp.NewResourceTemplate(
		"kafka-mcp://{cluster}/visualizations/{topic}/partition_distribution.svg",
		"Partition Distribution",
		mcp.WithTemplateDescription("SVG depicting partition leadership distribution"),
		mcp.WithTemplateMIMEType("image/svg+xml"),
	)

	s.AddResourceTemplate(partitionDistributionResource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Extract parameters from the URI
		params := extractParamsFromURI(req.Params.URI, "kafka-mcp://{cluster}/visualizations/{topic}/partition_distribution.svg")

		topic, ok := params["topic"]
		if !ok || topic == "" {
			return nil, fmt.Errorf("topic parameter is required")
		}

		slog.InfoContext(ctx, "Executing partition_distribution resource", "topic", topic)

		// In a real implementation, this would generate an actual SVG based on partition data
		// For now, returning a minimal SVG as a placeholder
		svgData := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="200" height="100">
			<rect width="200" height="100" style="fill:rgb(240,240,240);stroke-width:1;stroke:rgb(0,0,0)" />
			<text x="10" y="50" font-family="Verdana" font-size="12">Topic: %s Partition Distribution</text>
		</svg>`, topic)

		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				Text: svgData,
			},
		}, nil
	})

	// --- Additional Resources ---

	// List Topics Resource
	listTopicsResource := mcp.NewResource(
		"list_topics",
		"List available Kafka topics",
		mcp.WithResourceDescription("List all topics in the Kafka cluster"),
		mcp.WithMIMEType("application/json"),
	)

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

	// Cluster Overview Resource
	clusterOverviewResource := mcp.NewResource(
		"cluster_overview",
		"Cluster Overview",
		mcp.WithResourceDescription("Summary information about the Kafka cluster"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(clusterOverviewResource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		slog.InfoContext(ctx, "Executing cluster_overview resource")

		overview, err := kafkaClient.GetClusterOverview(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get cluster overview", "error", err)
			return nil, err
		}

		jsonData, err := json.Marshal(overview)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to marshal cluster overview", "error", err)
			return nil, err
		}

		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				Text: string(jsonData),
			},
		}, nil
	})

	// List Consumer Groups Resource
	listConsumerGroupsResource := mcp.NewResource(
		"list_consumer_groups",
		"List Consumer Groups",
		mcp.WithResourceDescription("List all consumer groups in the Kafka cluster"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(listConsumerGroupsResource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		slog.InfoContext(ctx, "Executing list_consumer_groups resource")

		groups, err := kafkaClient.ListConsumerGroups(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to list consumer groups", "error", err)
			return nil, err
		}

		jsonData, err := json.Marshal(groups)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to marshal consumer groups data", "error", err)
			return nil, err
		}

		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				Text: string(jsonData),
			},
		}, nil
	})

	// List Brokers Resource
	listBrokersResource := mcp.NewResource(
		"list_brokers",
		"List Brokers",
		mcp.WithResourceDescription("List all brokers in the Kafka cluster"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(listBrokersResource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		slog.InfoContext(ctx, "Executing list_brokers resource")

		brokers, err := kafkaClient.ListBrokers(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to list brokers", "error", err)
			return nil, err
		}

		// Convert string list to JSON
		jsonData, err := json.Marshal(brokers)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to marshal broker list", "error", err)
			return nil, err
		}

		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				Text: string(jsonData),
			},
		}, nil
	})

	// Broker Log Resource
	brokerLogResource := mcp.NewResourceTemplate(
		"kafka-mcp://{cluster}/logs/broker/{brokerId}",
		"Broker Log",
		mcp.WithTemplateDescription("Access Kafka broker log files"),
		mcp.WithTemplateMIMEType("text/plain"),
	)

	s.AddResourceTemplate(brokerLogResource, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Extract parameters from the URI
		params := extractParamsFromURI(req.Params.URI, "kafka-mcp://{cluster}/logs/broker/{brokerId}")

		brokerId, ok := params["brokerId"]
		if !ok || brokerId == "" {
			return nil, fmt.Errorf("brokerId parameter is required")
		}

		// Parse query parameters
		queryParams := extractQueryParams(req.Params.URI)

		lines := 100
		if linesStr, ok := queryParams["lines"]; ok && linesStr != "" {
			fmt.Sscanf(linesStr, "%d", &lines)
			if lines <= 0 {
				lines = 100
			}
		}

		slog.InfoContext(ctx, "Executing broker_log resource", "brokerId", brokerId, "lines", lines)

		// In a real implementation, this would read actual log files
		// For now, returning simulated log data
		logData := fmt.Sprintf("Simulated log data for broker %s (latest %d lines)\n", brokerId, lines)
		logData += "[2023-07-10 12:00:01,123] INFO [KafkaServer id=1] Started (kafka.server.KafkaServer)\n"
		logData += "[2023-07-10 12:00:02,456] INFO [SocketServer brokerId=1] Started data-plane acceptor and processor(s) for endpoint : ListenerName(PLAINTEXT) (kafka.network.SocketServer)\n"
		logData += "[2023-07-10 12:00:03,789] INFO [BrokerToControllerChannelManager broker=1 name=forwarding]: Recorded new controller, from now on will use node 1 (kafka.server.BrokerToControllerRequestThread)\n"
		// Add more simulated log lines as needed

		// Build a safe file path for the broker log
		// In a real implementation, this would be a valid path to the actual log file
		logFilePath := filepath.Join("/var/log/kafka", fmt.Sprintf("server-%s.log", brokerId))
		_ = logFilePath // Unused but kept for reference

		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				Text: logData,
				// Note: TextResourceContents doesn't have a Path field in latest mcp-go version
			},
		}, nil
	})
}

// Helper function to extract path parameters from a URI based on a template
func extractParamsFromURI(uri, template string) map[string]string {
	params := make(map[string]string)

	// Split both URI and template into path segments
	uriParts := strings.Split(strings.Split(uri, "?")[0], "/")
	templateParts := strings.Split(template, "/")

	// Match corresponding segments
	for i := 0; i < len(templateParts) && i < len(uriParts); i++ {
		if strings.HasPrefix(templateParts[i], "{") && strings.HasSuffix(templateParts[i], "}") {
			// Extract the parameter name without the braces
			paramName := templateParts[i][1 : len(templateParts[i])-1]
			// Get the value from the URI
			params[paramName] = uriParts[i]
		}
	}

	return params
}

// Helper function to extract query parameters from a URI
func extractQueryParams(uri string) map[string]string {
	queryParams := make(map[string]string)

	parts := strings.Split(uri, "?")
	if len(parts) < 2 {
		return queryParams
	}

	queryString := parts[1]
	paramPairs := strings.Split(queryString, "&")

	for _, pair := range paramPairs {
		keyValue := strings.SplitN(pair, "=", 2)
		if len(keyValue) == 2 {
			queryParams[keyValue[0]] = keyValue[1]
		}
	}

	return queryParams
}
