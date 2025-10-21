// Package mcp provides the MCP server functionality for Kafka.
package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tuannvm/kafka-mcp-server/internal/kafka"
)

// ResourceContentsFunc defines a function that returns the contents of a resource
type ResourceContentsFunc func(ctx context.Context, uri string) ([]byte, error)

// ResourceHandlerFunc is a wrapper around ResourceContentsFunc to match the server.ResourceHandlerFunc signature
func resourceHandlerFuncWrapper(contentsFunc ResourceContentsFunc) server.ResourceHandlerFunc {
	return func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		data, err := contentsFunc(ctx, req.Params.URI)
		if err != nil {
			return nil, err
		}

		// Convert to text resource contents
		textContent := mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}
		return []mcp.ResourceContents{textContent}, nil
	}
}

// RegisterResources registers MCP resources with the server.
func RegisterResources(s *server.MCPServer, kafkaClient kafka.KafkaClient) {
	// Register core resources
	s.AddResource(mcp.Resource{
		URI:         "kafka-mcp://overview",
		Name:        "Kafka Cluster Overview",
		Description: "Comprehensive summary of Kafka cluster health including broker counts, controller status, topic/partition metrics, and replication health. Use this resource for quick cluster status assessment and monitoring dashboards.",
		MIMEType:    "application/json",
	}, resourceHandlerFuncWrapper(clusterOverviewResource(kafkaClient)))

	s.AddResource(mcp.Resource{
		URI:         "kafka-mcp://health-check",
		Name:        "Kafka Cluster Health Check",
		Description: "Detailed health assessment of the Kafka cluster covering broker availability, controller status, partition health, and consumer group performance. Provides actionable insights for troubleshooting and maintenance.",
		MIMEType:    "application/json",
	}, resourceHandlerFuncWrapper(healthCheckResource(kafkaClient)))

	s.AddResource(mcp.Resource{
		URI:         "kafka-mcp://under-replicated-partitions",
		Name:        "Under-Replicated Partitions Report",
		Description: "Comprehensive analysis of partitions with insufficient replication, including affected topics, missing replicas, and troubleshooting recommendations. Critical for identifying and resolving data durability issues.",
		MIMEType:    "application/json",
	}, resourceHandlerFuncWrapper(underReplicatedPartitionsResource(kafkaClient)))

	s.AddResource(mcp.Resource{
		URI:         "kafka-mcp://consumer-lag-report",
		Name:        "Consumer Group Lag Analysis",
		Description: "Detailed analysis of consumer group performance including lag metrics, group states, partition assignments, and performance recommendations. Supports threshold-based alerting and performance optimization.",
		MIMEType:    "application/json",
	}, resourceHandlerFuncWrapper(consumerLagReportResource(kafkaClient)))
}

// clusterOverviewResource returns a resource for the cluster overview
func clusterOverviewResource(kafkaClient kafka.KafkaClient) ResourceContentsFunc {
	return func(ctx context.Context, uri string) ([]byte, error) {
		slog.InfoContext(ctx, "Fetching cluster overview resource", "uri", uri)

		// Get overview data
		overview, err := kafkaClient.GetClusterOverview(ctx)
		if err != nil {
			return handleResourceError(ctx, err, "Failed to get cluster overview")
		}

		// Create a structured response
		response := createBaseResourceResponse()

		// Add cluster specific fields
		response["broker_count"] = overview.BrokerCount
		response["controller_id"] = overview.ControllerID
		response["topic_count"] = overview.TopicCount
		response["partition_count"] = overview.PartitionCount
		response["under_replicated_partitions"] = overview.UnderReplicatedPartitionsCount
		response["offline_partitions"] = overview.OfflinePartitionsCount
		response["offline_broker_ids"] = overview.OfflineBrokerIDs
		response["health_status"] = getHealthStatusString(
			overview.OfflinePartitionsCount > 0,
			overview.ControllerID == -1,
			len(overview.OfflineBrokerIDs) > 0,
			overview.UnderReplicatedPartitionsCount > 0,
		)

		return json.Marshal(response)
	}
}

// healthCheckResource returns a resource for detailed cluster health assessment
func healthCheckResource(kafkaClient kafka.KafkaClient) ResourceContentsFunc {
	return func(ctx context.Context, uri string) ([]byte, error) {
		slog.InfoContext(ctx, "Fetching health check resource", "uri", uri)

		// Get overview for basic health info
		overview, err := kafkaClient.GetClusterOverview(ctx)
		if err != nil {
			return handleResourceError(ctx, err, "Failed to get cluster overview for health check")
		}

		// Get consumer groups
		groups, groupErr := kafkaClient.ListConsumerGroups(ctx)

		// Track groups with high lag
		groupsWithHighLag := 0
		highLagThreshold := int64(10000) // Consider high lag if over 10K messages

		if groupErr == nil {
			for _, groupInfo := range groups {
				// Get detailed info including lag
				descResult, descErr := kafkaClient.DescribeConsumerGroup(ctx, groupInfo.GroupID, true)
				if descErr == nil && descResult.ErrorCode == 0 {
					for _, offsetInfo := range descResult.Offsets {
						if offsetInfo.Lag > highLagThreshold {
							groupsWithHighLag++
							break // Only count each group once
						}
					}
				}
			}
		}

		// Create a structured response
		response := createBaseResourceResponse()

		// Add broker status
		response["broker_status"] = map[string]interface{}{
			"total_brokers":      overview.BrokerCount,
			"offline_brokers":    len(overview.OfflineBrokerIDs),
			"offline_broker_ids": overview.OfflineBrokerIDs,
			"status":             getStatus(len(overview.OfflineBrokerIDs) > 0, "critical", "healthy"),
		}

		// Add controller status
		response["controller_status"] = map[string]interface{}{
			"controller_id": overview.ControllerID,
			"status":        getStatus(overview.ControllerID == -1, "critical", "healthy"),
		}

		// Add partition status
		response["partition_status"] = map[string]interface{}{
			"total_partitions":            overview.PartitionCount,
			"under_replicated_partitions": overview.UnderReplicatedPartitionsCount,
			"offline_partitions":          overview.OfflinePartitionsCount,
			"status": getStatus(
				overview.OfflinePartitionsCount > 0 || overview.UnderReplicatedPartitionsCount > 0,
				"critical",
				"healthy",
			),
		}

		// Add consumer status
		response["consumer_status"] = map[string]interface{}{
			"total_groups":         len(groups),
			"groups_with_high_lag": groupsWithHighLag,
			"status":               getStatus(groupsWithHighLag > 0, "warning", "healthy"),
			"error":                getErrorString(groupErr),
		}

		// Add overall status
		response["overall_status"] = getHealthStatusString(
			overview.OfflinePartitionsCount > 0,
			overview.ControllerID == -1,
			len(overview.OfflineBrokerIDs) > 0,
			overview.UnderReplicatedPartitionsCount > 0 || groupsWithHighLag > 0,
		)

		return json.Marshal(response)
	}
}

// underReplicatedPartitionsResource returns a resource for under-replicated partitions report
func underReplicatedPartitionsResource(kafkaClient kafka.KafkaClient) ResourceContentsFunc {
	return func(ctx context.Context, uri string) ([]byte, error) {
		slog.InfoContext(ctx, "Fetching under-replicated partitions resource", "uri", uri)

		// Get overview for quick check
		overview, err := kafkaClient.GetClusterOverview(ctx)
		if err != nil {
			return handleResourceError(ctx, err, "Failed to get cluster overview for URP check")
		}

		// Create base response
		response := createBaseResourceResponse()
		response["under_replicated_partition_count"] = overview.UnderReplicatedPartitionsCount
		response["details"] = []map[string]interface{}{}

		// If no URPs, return early
		if overview.UnderReplicatedPartitionsCount == 0 {
			return json.Marshal(response)
		}

		// Get list of all topics
		topics, err := kafkaClient.ListTopics(ctx)
		if err != nil {
			return handleResourceError(ctx, err, "Failed to list topics")
		}

		// Track URPs
		urpDetails := []map[string]interface{}{}

		// For each topic, get details and check for URPs
		for _, topic := range topics {
			topicInfo, err := kafkaClient.DescribeTopic(ctx, topic)
			if err != nil {
				slog.WarnContext(ctx, "Error getting topic details", "topic", topic, "error", err)
				continue
			}

			for _, partition := range topicInfo.Partitions {
				if len(partition.ISR) < len(partition.Replicas) {
					// Format missing replicas
					missingReplicas := []int{}
					replicaMap := make(map[int]bool)

					// Convert ISR to map for quicker lookup
					for _, isr := range partition.ISR {
						replicaMap[int(isr)] = true
					}

					// Find which replicas are not in ISR
					for _, replica := range partition.Replicas {
						if !replicaMap[int(replica)] {
							missingReplicas = append(missingReplicas, int(replica))
						}
					}

					// Add to details
					urpDetails = append(urpDetails, map[string]interface{}{
						"topic":            topic,
						"partition":        partition.PartitionID,
						"leader":           partition.Leader,
						"replica_count":    len(partition.Replicas),
						"isr_count":        len(partition.ISR),
						"replicas":         partition.Replicas,
						"isr":              partition.ISR,
						"missing_replicas": missingReplicas,
					})
				}
			}
		}

		// Add details to response
		response["details"] = urpDetails

		// Add recommendations if URPs were found
		recommendations := []string{
			"Check broker health for any offline or struggling brokers",
			"Verify network connectivity between brokers",
			"Monitor disk space on broker nodes",
			"Review broker logs for detailed error messages",
			"Consider increasing replication timeouts if network is slow",
		}

		addRecommendations(response, len(urpDetails) > 0, recommendations)

		return json.Marshal(response)
	}
}

// consumerLagReportResource returns a resource for consumer lag analysis
func consumerLagReportResource(kafkaClient kafka.KafkaClient) ResourceContentsFunc {
	return func(ctx context.Context, uri string) ([]byte, error) {
		slog.InfoContext(ctx, "Fetching consumer lag report resource", "uri", uri)

		// Get threshold from query parameters (default to 1000)
		var lagThreshold int64 = 1000
		if thresholdStr := extractURIQueryParameter(uri, "threshold"); thresholdStr != "" {
			if threshold, err := strconv.ParseInt(thresholdStr, 10, 64); err == nil && threshold >= 0 {
				lagThreshold = threshold
			} else {
				slog.WarnContext(ctx, "Invalid lag threshold, using default", "threshold", thresholdStr, "default", lagThreshold)
			}
		}

		// Get list of consumer groups
		groups, err := kafkaClient.ListConsumerGroups(ctx)
		if err != nil {
			return handleResourceError(ctx, err, "Failed to list consumer groups")
		}

		// Prepare response
		response := createBaseResourceResponse()
		response["lag_threshold"] = lagThreshold
		response["group_count"] = len(groups)
		response["group_summary"] = []map[string]interface{}{}
		response["high_lag_details"] = []map[string]interface{}{}

		if len(groups) == 0 {
			return json.Marshal(response)
		}

		// Track groups with high lag
		groupSummary := []map[string]interface{}{}
		highLagDetails := []map[string]interface{}{}
		groupsWithHighLag := 0

		for _, groupInfo := range groups {
			// Describe each group to get offset/lag info
			descResult, descErr := kafkaClient.DescribeConsumerGroup(ctx, groupInfo.GroupID, true)

			groupState := "Unknown"
			memberCount := 0
			topicSet := make(map[string]bool)
			totalLag := int64(0)
			hasHighLag := false

			if descErr == nil && descResult.ErrorCode == 0 {
				groupState = descResult.State
				memberCount = len(descResult.Members)

				// Process offset/lag information
				for _, offsetInfo := range descResult.Offsets {
					topicSet[offsetInfo.Topic] = true
					totalLag += offsetInfo.Lag

					if offsetInfo.Lag > lagThreshold {
						hasHighLag = true

						// Add to high lag details
						highLagDetails = append(highLagDetails, map[string]interface{}{
							"group_id":       groupInfo.GroupID,
							"topic":          offsetInfo.Topic,
							"partition":      offsetInfo.Partition,
							"current_offset": offsetInfo.CommitOffset,
							"log_end_offset": offsetInfo.CommitOffset + offsetInfo.Lag, // Approximate
							"lag":            offsetInfo.Lag,
						})
					}
				}

				if hasHighLag {
					groupsWithHighLag++
				}
			}

			// Convert topic set to slice
			topics := []string{}
			for topic := range topicSet {
				topics = append(topics, topic)
			}

			// Add group summary
			groupSummary = append(groupSummary, map[string]interface{}{
				"group_id":     groupInfo.GroupID,
				"state":        groupState,
				"member_count": memberCount,
				"topics":       topics,
				"total_lag":    totalLag,
				"has_high_lag": hasHighLag,
			})
		}

		// Update response
		response["group_summary"] = groupSummary
		response["high_lag_details"] = highLagDetails
		response["groups_with_high_lag"] = groupsWithHighLag

		// Add recommendations if high lag was found
		recommendations := []string{
			"Increase consumer instances to process messages faster",
			"Check for consumer errors or bottlenecks in consumer application logs",
			"Verify producer rate isn't abnormally high",
			"Review consumer configuration for performance tuning",
			"Consider topic partitioning to allow more parallel processing",
		}

		addRecommendations(response, groupsWithHighLag > 0, recommendations)

		return json.Marshal(response)
	}
}
