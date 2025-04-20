// Package mcp provides the MCP server functionality for Kafka.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tuannvm/kafka-mcp-server/kafka"
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
		URI:         "kafka-mcp://{cluster}/overview",
		Name:        "Cluster Overview",
		Description: "Summary of Kafka cluster health and metrics",
		MIMEType:    "application/json",
	}, resourceHandlerFuncWrapper(clusterOverviewResource(kafkaClient)))

	s.AddResource(mcp.Resource{
		URI:         "kafka-mcp://{cluster}/health-check",
		Name:        "Health Check",
		Description: "Comprehensive health assessment of the Kafka cluster",
		MIMEType:    "application/json",
	}, resourceHandlerFuncWrapper(healthCheckResource(kafkaClient)))

	s.AddResource(mcp.Resource{
		URI:         "kafka-mcp://{cluster}/under-replicated-partitions",
		Name:        "Under-Replicated Partitions",
		Description: "Detailed report of under-replicated partitions",
		MIMEType:    "application/json",
	}, resourceHandlerFuncWrapper(underReplicatedPartitionsResource(kafkaClient)))

	s.AddResource(mcp.Resource{
		URI:         "kafka-mcp://{cluster}/consumer-lag-report",
		Name:        "Consumer Lag Report",
		Description: "Analysis of consumer group lag across the cluster",
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
			slog.ErrorContext(ctx, "Failed to get cluster overview", "error", err)
			return nil, fmt.Errorf("failed to get cluster overview: %w", err)
		}

		// Create a structured response
		response := map[string]interface{}{
			"broker_count":                overview.BrokerCount,
			"controller_id":               overview.ControllerID,
			"topic_count":                 overview.TopicCount,
			"partition_count":             overview.PartitionCount,
			"under_replicated_partitions": overview.UnderReplicatedPartitionsCount,
			"offline_partitions":          overview.OfflinePartitionsCount,
			"offline_broker_ids":          overview.OfflineBrokerIDs,
			"health_status": getHealthStatus(
				overview.OfflinePartitionsCount > 0,
				overview.ControllerID == -1,
				len(overview.OfflineBrokerIDs) > 0,
				overview.UnderReplicatedPartitionsCount > 0,
			),
			"timestamp": getCurrentTimestamp(),
		}

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
			slog.ErrorContext(ctx, "Failed to get cluster overview for health check", "error", err)
			return nil, fmt.Errorf("failed to get cluster overview for health check: %w", err)
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
		response := map[string]interface{}{
			"timestamp": getCurrentTimestamp(),
			"broker_status": map[string]interface{}{
				"total_brokers":      overview.BrokerCount,
				"offline_brokers":    len(overview.OfflineBrokerIDs),
				"offline_broker_ids": overview.OfflineBrokerIDs,
				"status":             getStatus(len(overview.OfflineBrokerIDs) > 0, "critical", "healthy"),
			},
			"controller_status": map[string]interface{}{
				"controller_id": overview.ControllerID,
				"status":        getStatus(overview.ControllerID == -1, "critical", "healthy"),
			},
			"partition_status": map[string]interface{}{
				"total_partitions":            overview.PartitionCount,
				"under_replicated_partitions": overview.UnderReplicatedPartitionsCount,
				"offline_partitions":          overview.OfflinePartitionsCount,
				"status": getPartitionHealthStatus(
					overview.UnderReplicatedPartitionsCount,
					overview.OfflinePartitionsCount,
				),
			},
			"consumer_status": map[string]interface{}{
				"total_groups":         len(groups),
				"groups_with_high_lag": groupsWithHighLag,
				"status":               getStatus(groupsWithHighLag > 0, "warning", "healthy"),
				"error":                getErrorString(groupErr),
			},
			"overall_status": getHealthStatus(
				overview.OfflinePartitionsCount > 0,
				overview.ControllerID == -1,
				len(overview.OfflineBrokerIDs) > 0,
				overview.UnderReplicatedPartitionsCount > 0 || groupsWithHighLag > 0,
			),
		}

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
			slog.ErrorContext(ctx, "Failed to get cluster overview for URP check", "error", err)
			return nil, fmt.Errorf("failed to get cluster overview for URP check: %w", err)
		}

		// Create base response
		response := map[string]interface{}{
			"timestamp":                        getCurrentTimestamp(),
			"under_replicated_partition_count": overview.UnderReplicatedPartitionsCount,
			"details":                          []map[string]interface{}{},
		}

		// If no URPs, return early
		if overview.UnderReplicatedPartitionsCount == 0 {
			return json.Marshal(response)
		}

		// Get list of all topics
		topics, err := kafkaClient.ListTopics(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to list topics", "error", err)
			return nil, fmt.Errorf("failed to list topics: %w", err)
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
		if len(urpDetails) > 0 {
			response["recommendations"] = []string{
				"Check broker health for any offline or struggling brokers",
				"Verify network connectivity between brokers",
				"Monitor disk space on broker nodes",
				"Review broker logs for detailed error messages",
				"Consider increasing replication timeouts if network is slow",
			}
		}

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
			slog.ErrorContext(ctx, "Failed to list consumer groups", "error", err)
			return nil, fmt.Errorf("failed to list consumer groups: %w", err)
		}

		// Prepare response
		response := map[string]interface{}{
			"timestamp":        getCurrentTimestamp(),
			"lag_threshold":    lagThreshold,
			"group_count":      len(groups),
			"group_summary":    []map[string]interface{}{},
			"high_lag_details": []map[string]interface{}{},
		}

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
		if groupsWithHighLag > 0 {
			response["recommendations"] = []string{
				"Increase consumer instances to process messages faster",
				"Check for consumer errors or bottlenecks in consumer application logs",
				"Verify producer rate isn't abnormally high",
				"Review consumer configuration for performance tuning",
				"Consider topic partitioning to allow more parallel processing",
			}
		}

		return json.Marshal(response)
	}
}

// Helper functions

// getHealthStatus returns a string indicating the overall health status
func getHealthStatus(hasOfflinePartitions, noController, hasOfflineBrokers, hasWarnings bool) string {
	if hasOfflinePartitions || noController || hasOfflineBrokers {
		return "critical"
	}
	if hasWarnings {
		return "warning"
	}
	return "healthy"
}

// getPartitionHealthStatus returns a string indicating partition health
func getPartitionHealthStatus(urpCount, offlineCount int) string {
	if offlineCount > 0 {
		return "critical"
	}
	if urpCount > 0 {
		return "warning"
	}
	return "healthy"
}

// getCurrentTimestamp returns the current timestamp in ISO 8601 format
func getCurrentTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// extractURIPathParameter extracts a path parameter from a URI
func extractURIPathParameter(name string) func(uri string) string {
	return func(uri string) string {
		pattern := fmt.Sprintf("{%s}", name)
		parts := strings.Split(uri, "/")

		for i, part := range parts {
			if part == pattern && i+1 < len(parts) {
				return parts[i+1]
			}
		}

		return ""
	}
}

// extractURIQueryParameter extracts a query parameter from a URI
func extractURIQueryParameter(uri, name string) string {
	if parsedURL, err := url.Parse(uri); err == nil {
		values := parsedURL.Query()
		return values.Get(name)
	}
	return ""
}

// extractURIComponent extracts a specific component from a URI
func extractURIComponent(uri, pattern, defaultValue string) string {
	if value := strings.TrimSpace(extractURIPathParameter(pattern)(uri)); value != "" {
		return value
	}
	return defaultValue
}

// getStatus returns the first value if condition is true, otherwise the second value
func getStatus(condition bool, trueValue, falseValue string) string {
	if condition {
		return trueValue
	}
	return falseValue
}

// getErrorString returns the error string or empty string if error is nil
func getErrorString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
