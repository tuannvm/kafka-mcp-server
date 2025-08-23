// filepath: /Users/tuannvm/Projects/cli/kafka-mcp-server/mcp/prompts.go
package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tuannvm/kafka-mcp-server/kafka"
)

// RegisterPrompts registers prompts with the MCP server.
func RegisterPrompts(s *server.MCPServer, kafkaClient kafka.KafkaClient) {
	// Register cluster overview prompt
	clusterOverviewPrompt := mcp.Prompt{
		Name:        "kafka_cluster_overview",
		Description: "Generates a comprehensive, human-readable summary of Kafka cluster health including broker counts, controller status, topic/partition metrics, and replication health. Perfect for status reports, monitoring dashboards, and quick cluster assessments.",
		Arguments: []mcp.PromptArgument{},
	}
	s.AddPrompt(clusterOverviewPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		slog.InfoContext(ctx, "Executing cluster overview prompt")

		// Get overview data
		overview, err := kafkaClient.GetClusterOverview(ctx)
		if err != nil {
			return handlePromptError(ctx, "Error retrieving cluster overview", err, "Failed to get cluster overview")
		}

		// Format the response
		response := formatResponseHeader("Kafka Cluster Overview")

		response += fmt.Sprintf("- **Broker Count**: %d\n"+
			"- **Active Controller ID**: %d\n"+
			"- **Total Topics**: %d\n"+
			"- **Total Partitions**: %d\n"+
			"- **Under-Replicated Partitions**: %d\n"+
			"- **Offline Partitions**: %d\n\n",
			overview.BrokerCount,
			overview.ControllerID,
			overview.TopicCount,
			overview.PartitionCount,
			overview.UnderReplicatedPartitionsCount,
			overview.OfflinePartitionsCount)

		// Check for warning conditions
		critical := overview.OfflinePartitionsCount > 0 || overview.ControllerID == -1 || len(overview.OfflineBrokerIDs) > 0
		warning := overview.UnderReplicatedPartitionsCount > 0

		if len(overview.OfflineBrokerIDs) > 0 {
			brokerIDsStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(overview.OfflineBrokerIDs)), ", "), "[]")
			response += fmt.Sprintf("⚠️ **Warning**: %d brokers are offline (IDs: %s)\n\n",
				len(overview.OfflineBrokerIDs), brokerIDsStr)
		}

		if overview.UnderReplicatedPartitionsCount > 0 {
			response += fmt.Sprintf("⚠️ **Warning**: %d partitions are under-replicated\n\n",
				overview.UnderReplicatedPartitionsCount)
		}

		if overview.OfflinePartitionsCount > 0 {
			response += fmt.Sprintf("🚨 **Critical**: %d partitions are offline\n\n",
				overview.OfflinePartitionsCount)
		}

		if overview.ControllerID == -1 {
			response += "🚨 **Critical**: No active controller found in the cluster\n\n"
		}

		// Add overall health status
		emoji, statusText := formatHealthStatus(critical, warning)
		response += fmt.Sprintf("**Overall Status**: %s %s\n", emoji, statusText)

		return createSuccessResponse("Kafka Cluster Overview", response)
	})

	// Register health check prompt
	healthCheckPrompt := mcp.Prompt{
		Name:        "kafka_health_check",
		Description: "Performs a comprehensive health assessment of the Kafka cluster with detailed analysis of broker availability, controller status, partition health, and consumer group performance. Provides actionable recommendations for troubleshooting and maintenance activities.",
		Arguments: []mcp.PromptArgument{},
	}
	s.AddPrompt(healthCheckPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		slog.InfoContext(ctx, "Executing health check prompt")

		// Get overview data for basic health info
		overview, err := kafkaClient.GetClusterOverview(ctx)
		if err != nil {
			return handlePromptError(ctx, "Error running health check", err, "Failed to get cluster overview for health check")
		}

		// Begin building the health check report
		response := formatResponseHeader("Kafka Cluster Health Check Report")

		// Add broker status section
		response += "## Broker Status\n\n"
		if len(overview.OfflineBrokerIDs) > 0 {
			brokerIDsStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(overview.OfflineBrokerIDs)), ", "), "[]")
			response += fmt.Sprintf("- 🚨 **%d of %d brokers are OFFLINE** (IDs: %s)\n",
				len(overview.OfflineBrokerIDs), overview.BrokerCount, brokerIDsStr)
			response += "- **Recommendation**: Investigate why these brokers are offline and restart if necessary\n\n"
		} else {
			response += fmt.Sprintf("- ✅ **All %d brokers are online**\n\n", overview.BrokerCount)
		}

		// Add controller status
		response += "## Controller Status\n\n"
		if overview.ControllerID == -1 {
			response += "- 🚨 **No active controller found**\n"
			response += "- **Recommendation**: Check broker logs to identify controller election issues\n\n"
		} else {
			response += fmt.Sprintf("- ✅ **Active controller**: Broker %d\n\n", overview.ControllerID)
		}

		// Add partition health
		response += "## Partition Health\n\n"
		healthIssues := false

		if overview.UnderReplicatedPartitionsCount > 0 {
			response += fmt.Sprintf("- ⚠️ **%d under-replicated partitions detected**\n", overview.UnderReplicatedPartitionsCount)
			response += "- **Recommendation**: Check broker health and network connectivity\n"
			healthIssues = true
		}

		if overview.OfflinePartitionsCount > 0 {
			response += fmt.Sprintf("- 🚨 **%d offline partitions detected**\n", overview.OfflinePartitionsCount)
			response += "- **Recommendation**: Check if leader replicas are available for these partitions\n"
			healthIssues = true
		}

		if !healthIssues {
			response += fmt.Sprintf("- ✅ **All %d partitions are healthy**\n", overview.PartitionCount)
		}

		response += "\n"

		// Add consumer group status
		response += "## Consumer Group Status\n\n"

		// Get consumer groups
		groups, groupErr := kafkaClient.ListConsumerGroups(ctx)
		if groupErr != nil {
			response += fmt.Sprintf("- ⚠️ **Unable to check consumer groups**: %s\n\n", groupErr.Error())
		} else {
			// Track groups with high lag
			groupsWithHighLag := 0
			highLagThreshold := int64(10000) // Consider high lag if over 10K messages

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

			response += fmt.Sprintf("- **Total consumer groups**: %d\n", len(groups))

			if groupsWithHighLag > 0 {
				response += fmt.Sprintf("- ⚠️ **%d consumer groups have high lag** (>10K messages)\n", groupsWithHighLag)
				response += "- **Recommendation**: Check consumer performance and consider scaling consumers\n\n"
			} else {
				response += "- ✅ **No consumer groups with high lag detected**\n\n"
			}
		}

		// Add overall health status
		response += "## Overall Health Assessment\n\n"

		// Determine overall health status
		critical := overview.OfflinePartitionsCount > 0 || overview.ControllerID == -1 || len(overview.OfflineBrokerIDs) > 0
		warning := overview.UnderReplicatedPartitionsCount > 0 || ((groups != nil) && len(groups) > 0)

		emoji, statusText := formatHealthStatus(critical, warning)
		response += fmt.Sprintf("%s **%s**: ", emoji, strings.ToUpper(statusText))

		if critical {
			response += "The cluster has serious issues that require immediate attention.\n"
		} else if warning {
			response += "The cluster has issues that should be investigated soon.\n"
		} else {
			response += "All systems are operating normally.\n"
		}

		return createSuccessResponse("Kafka Health Check Report", response)
	})

	// Register under-replicated partitions prompt
	underReplicatedPrompt := mcp.Prompt{
		Name:        "kafka_under_replicated_partitions",
		Description: "Identifies and analyzes partitions with insufficient replication, where the in-sync replica (ISR) count is less than the configured replication factor. Provides detailed reporting on affected topics, missing replicas, potential causes, and step-by-step troubleshooting recommendations to restore data durability.",
		Arguments: []mcp.PromptArgument{},
	}
	s.AddPrompt(underReplicatedPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		slog.InfoContext(ctx, "Executing under-replicated partitions prompt")

		// Get overview for quick check
		overview, err := kafkaClient.GetClusterOverview(ctx)
		if err != nil {
			return handlePromptError(ctx, "Error retrieving under-replicated partitions", err, "Failed to get cluster overview for URP check")
		}

		// Begin building the response
		response := formatResponseHeader("Under-Replicated Partitions Report")

		if overview.UnderReplicatedPartitionsCount == 0 {
			response += "✅ **Good news!** No under-replicated partitions found.\n"
			return createSuccessResponse("No under-replicated partitions found", response)
		}

		response += fmt.Sprintf("⚠️ **Found %d under-replicated partitions**\n\n", overview.UnderReplicatedPartitionsCount)

		// Get list of all topics
		topics, err := kafkaClient.ListTopics(ctx)
		if err != nil {
			return handlePromptError(ctx, "Error listing topics", err, "Failed to list topics")
		}

		// Add table header
		response += "| Topic | Partition | Leader | Replica Count | ISR Count | Missing Replicas |\n"
		response += "|-------|-----------|--------|---------------|-----------|------------------|\n"

		// Track whether we found any URPs
		foundURPs := false

		// For each topic, get details and check for URPs
		for _, topic := range topics {
			topicInfo, err := kafkaClient.DescribeTopic(ctx, topic)
			if err != nil {
				slog.WarnContext(ctx, "Error getting topic details", "topic", topic, "error", err)
				continue
			}

			for _, partition := range topicInfo.Partitions {
				if len(partition.ISR) < len(partition.Replicas) {
					foundURPs = true

					// Format missing replicas
					missingReplicas := []int32{}
					replicaMap := make(map[int32]bool)

					// Convert ISR to map for quicker lookup
					for _, isr := range partition.ISR {
						replicaMap[isr] = true
					}

					// Find which replicas are not in ISR
					for _, replica := range partition.Replicas {
						if !replicaMap[replica] {
							missingReplicas = append(missingReplicas, replica)
						}
					}

					missingReplicasStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(missingReplicas)), ", "), "[]")

					// Add row to table
					response += fmt.Sprintf("| %s | %d | %d | %d | %d | %s |\n",
						topic, partition.PartitionID, partition.Leader, len(partition.Replicas),
						len(partition.ISR), missingReplicasStr)
				}
			}
		}

		// Add explanatory content if we found URPs
		if foundURPs {
			response += "\n## Possible Causes\n\n"
			response += "Under-replicated partitions occur when one or more replicas are not in sync with the leader. Common causes include:\n\n"
			response += "- **Broker failure or network partition**\n"
			response += "- **High load on brokers**\n"
			response += "- **Insufficient disk space**\n"
			response += "- **Network bandwidth limitations**\n"
			response += "- **Misconfigured topic replication factor**\n\n"

			response += "## Recommendations\n\n"
			response += "1. **Check broker health** for any offline or struggling brokers\n"
			response += "2. **Verify network connectivity** between brokers\n"
			response += "3. **Monitor disk space** on broker nodes\n"
			response += "4. **Review broker logs** for detailed error messages\n"
			response += "5. **Consider increasing replication timeouts** if network is slow\n"
		} else {
			// This is unlikely, but handle case where overview reported URPs but we didn't find any
			response += "\n⚠️ Overview reported under-replicated partitions, but none were found during detailed scan. The condition may have resolved itself.\n"
		}

		return createSuccessResponse("Under-Replicated Partitions Report", response)
	})

	// Register consumer lag report prompt
	consumerLagPrompt := mcp.Prompt{
		Name:        "kafka_consumer_lag_report",
		Description: "Generates a comprehensive consumer lag analysis report covering all consumer groups in the cluster. Analyzes group states, member assignments, partition lag metrics, and provides performance optimization recommendations. Supports customizable lag thresholds for alerting and includes actionable insights for scaling decisions.",
		Arguments: []mcp.PromptArgument{
			{
				Name:        "threshold",
				Description: "Message lag threshold for highlighting consumer groups with performance issues (default: 1000 messages). Groups exceeding this threshold will be flagged for attention.",
				Required:    false,
			},
		},
	}
	s.AddPrompt(consumerLagPrompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		slog.InfoContext(ctx, "Executing consumer lag report prompt")

		// Get threshold from arguments
		var lagThreshold int64 = 1000 // Default threshold
		if thresholdArg, ok := req.Params.Arguments["threshold"]; ok && thresholdArg != "" {
			threshold, err := strconv.ParseInt(thresholdArg, 10, 64)
			if err == nil && threshold >= 0 {
				lagThreshold = threshold
			} else {
				slog.WarnContext(ctx, "Invalid lag threshold, using default", "threshold", thresholdArg, "default", lagThreshold)
			}
		}

		// Get list of consumer groups
		groups, err := kafkaClient.ListConsumerGroups(ctx)
		if err != nil {
			return handlePromptError(ctx, "Error listing consumer groups", err, "Failed to list consumer groups")
		}

		// Begin building the response
		response := formatResponseHeader("Kafka Consumer Lag Report")
		response += fmt.Sprintf("**Lag Threshold**: %d messages\n\n", lagThreshold)

		if len(groups) == 0 {
			response += "No active consumer groups found.\n"
			return createSuccessResponse("No active consumer groups", response)
		}

		response += fmt.Sprintf("Found %d consumer group(s)\n\n", len(groups))

		// Track groups with high lag for summary
		groupsWithHighLag := 0
		highLagDetails := []map[string]interface{}{}

		// Consumer group summary table
		response += "## Consumer Group Summary\n\n"
		response += "| Group ID | State | Members | Topics | Total Lag | High Lag |\n"
		response += "|----------|-------|---------|--------|-----------|----------|\n"

		for _, groupInfo := range groups {
			// Describe each group to get offset/lag info
			descResult, descErr := kafkaClient.DescribeConsumerGroup(ctx, groupInfo.GroupID, true) // includeOffsets = true

			var state = "Unknown"
			members := 0
			topics := []string{}
			totalLag := int64(0)
			hasHighLag := false

			if descErr == nil && descResult.ErrorCode == 0 {
				state = descResult.State
				members = len(descResult.Members)

				// Process offset/lag information
				topicSet := make(map[string]bool)

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

				// Convert topic set to slice
				for topic := range topicSet {
					topics = append(topics, topic)
				}

				if hasHighLag {
					groupsWithHighLag++
				}
			}

			// Format topics list for display
			topicsStr := strings.Join(topics, ", ")
			if len(topicsStr) > 30 {
				topicsStr = topicsStr[:27] + "..."
			}

			// Add row to table
			highLagStatus := "✅ No"
			if hasHighLag {
				highLagStatus = "⚠️ Yes"
			}

			response += fmt.Sprintf("| %s | %s | %d | %s | %d | %s |\n",
				groupInfo.GroupID, state, members, topicsStr, totalLag, highLagStatus)
		}

		// Add high lag details if any exist
		if len(highLagDetails) > 0 {
			response += "\n## High Lag Details\n\n"
			response += "| Group ID | Topic | Partition | Current Offset | Log End Offset | Lag |\n"
			response += "|----------|-------|-----------|----------------|----------------|-----|\n"

			// Sort details by lag (descending) - in a real implementation we'd sort the slice
			// For simplicity, we're just iterating through the unsorted slice
			for _, detail := range highLagDetails {
				response += fmt.Sprintf("| %s | %s | %d | %d | %d | %d |\n",
					detail["group_id"], detail["topic"], detail["partition"].(int),
					detail["current_offset"].(int64), detail["log_end_offset"].(int64),
					detail["lag"].(int64))
			}
		}

		// Add recommendations if high lag was found
		if groupsWithHighLag > 0 {
			response += "\n## Recommendations\n\n"
			response += "Consumer groups with high lag may indicate performance issues or insufficient consumer capacity. Consider:\n\n"
			response += "1. **Increase consumer instances** to process messages faster\n"
			response += "2. **Check for consumer errors or bottlenecks** in consumer application logs\n"
			response += "3. **Verify producer rate** isn't abnormally high\n"
			response += "4. **Review consumer configuration** for performance tuning\n"
			response += "5. **Consider topic partitioning** to allow more parallel processing\n"
		} else {
			response += "\n✅ All consumer groups are keeping up with producers.\n"
		}

		return createSuccessResponse("Consumer Lag Report", response)
	})
}
