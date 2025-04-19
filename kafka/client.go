package kafka

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"

	"os"

	"github.com/tuannvm/kafka-mcp-server/config"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	"github.com/twmb/franz-go/pkg/sasl"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

// Message represents a consumed Kafka message data.
type Message struct {
	Topic     string `json:"topic"`
	Partition int32  `json:"partition"`
	Offset    int64  `json:"offset"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Timestamp int64  `json:"timestamp"` // Unix milliseconds
}

// TopicMetadata holds detailed information about a topic.
type TopicMetadata struct {
	TopicName    string              `json:"topic_name"`
	Partitions   []PartitionMetadata `json:"partitions"`
	IsInternal   bool                `json:"is_internal"`
	ErrorCode    int16               `json:"error_code,omitempty"` // Include error code if any
	ErrorMessage string              `json:"error_message,omitempty"`
}

// PartitionMetadata holds details about a single partition.
type PartitionMetadata struct {
	PartitionID  int32   `json:"partition_id"`
	Leader       int32   `json:"leader"`
	Replicas     []int32 `json:"replicas"`
	ISR          []int32 `json:"isr"` // In-Sync Replicas
	ErrorCode    int16   `json:"error_code,omitempty"`
	ErrorMessage string  `json:"error_message,omitempty"`
}

// ConsumerGroupInfo holds basic information about a consumer group.
type ConsumerGroupInfo struct {
	GroupID      string `json:"group_id"`
	State        string `json:"state"` // e.g., "Stable", "PreparingRebalance", "Empty"
	ErrorCode    int16  `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// DescribeConsumerGroupResult holds detailed information about a consumer group.
type DescribeConsumerGroupResult struct {
	GroupID      string                `json:"group_id"`
	State        string                `json:"state"`
	ProtocolType string                `json:"protocol_type"`
	Protocol     string                `json:"protocol"`
	Members      []GroupMemberInfo     `json:"members"`
	Offsets      []PartitionOffsetInfo `json:"offsets,omitempty"` // Optional, fetched separately
	// CoordinatorID int32            `json:"coordinator_id"` // Removed, not directly available in DescribeGroupsResponseGroup
	ErrorCode    int16  `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// GroupMemberInfo holds information about a single member within a consumer group.
type GroupMemberInfo struct {
	MemberID   string `json:"member_id"`
	ClientID   string `json:"client_id"`
	ClientHost string `json:"client_host"`
}

// PartitionOffsetInfo holds offset and lag information for a partition assigned to a group.
type PartitionOffsetInfo struct {
	Topic        string `json:"topic"`
	Partition    int32  `json:"partition"`
	CommitOffset int64  `json:"commit_offset"`
	Lag          int64  `json:"lag"` // Calculated: LogEndOffset - CommitOffset
	Metadata     string `json:"metadata,omitempty"`
	ErrorCode    int16  `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// ConfigEntry represents a single configuration key-value pair for a resource.
type ConfigEntry struct {
	Name        string `json:"name"`
	Value       string `json:"value,omitempty"` // Value might be nil if sensitive or default
	IsDefault   bool   `json:"is_default"`
	IsReadOnly  bool   `json:"is_read_only"`
	IsSensitive bool   `json:"is_sensitive"`
	Source      string `json:"source"` // e.g., "DEFAULT_CONFIG", "TOPIC_CONFIG", "BROKER_CONFIG"
	// Synonyms can be included if needed
}

// DescribeConfigsResult holds the configuration entries for a specific resource.
type DescribeConfigsResult struct {
	ResourceType string        `json:"resource_type"`
	ResourceName string        `json:"resource_name"`
	Configs      []ConfigEntry `json:"configs"`
	ErrorCode    int16         `json:"error_code,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

// ClusterOverviewResult holds high-level cluster health information.
type ClusterOverviewResult struct {
	ControllerID                   int32   `json:"controller_id"`
	BrokerCount                    int     `json:"broker_count"`
	TopicCount                     int     `json:"topic_count"` // Excluding internal topics
	PartitionCount                 int     `json:"partition_count"`
	UnderReplicatedPartitionsCount int     `json:"under_replicated_partitions_count"`
	OfflinePartitionsCount         int     `json:"offline_partitions_count"`
	OfflineBrokerIDs               []int32 `json:"offline_broker_ids,omitempty"` // Brokers that failed metadata request
	ErrorCode                      int16   `json:"error_code,omitempty"`         // Overall error, e.g., if metadata fails completely
	ErrorMessage                   string  `json:"error_message,omitempty"`
}

// Client wraps the franz-go Kafka client.
type Client struct {
	kgoClient *kgo.Client
}

// NewClient creates and initializes a new Kafka client based on the provided configuration.
func NewClient(cfg config.Config) (*Client, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.KafkaBrokers...),
		kgo.ClientID(cfg.KafkaClientID),
		// Corrected BasicLogger usage: write to os.Stderr
		kgo.WithLogger(kgo.BasicLogger(os.Stderr, kgo.LogLevelInfo, nil)),
		// Add consumer group if specified (needed for ConsumeMessages)
		// TODO: Make group ID configurable? For now, use client ID.
		kgo.ConsumerGroup(cfg.KafkaClientID),
		kgo.ConsumeTopics(),     // Initialize with empty topics, will be added in ConsumeMessages
		kgo.DisableAutoCommit(), // Recommend disabling auto-commit for MCP tools
	}

	// --- TLS Configuration ---
	if cfg.TLSEnable {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.TLSInsecureSkipVerify,
			// TODO: Add RootCAs, Certificates if client cert auth is needed
		}
		opts = append(opts, kgo.DialTLSConfig(tlsConfig))
		slog.Info("TLS enabled for Kafka client", "insecureSkipVerify", cfg.TLSInsecureSkipVerify)
	}

	// --- SASL Configuration ---
	var saslMethod sasl.Mechanism
	switch cfg.SASLMechanism {
	case "plain":
		if cfg.SASLUser == "" || cfg.SASLPassword == "" {
			return nil, fmt.Errorf("SASL PLAIN requires KAFKA_SASL_USER and KAFKA_SASL_PASSWORD")
		}
		saslMethod = plain.Auth{
			User: cfg.SASLUser,
			Pass: cfg.SASLPassword,
		}.AsMechanism()
		slog.Info("SASL PLAIN mechanism configured")
	case "scram-sha-256", "scram-sha-512":
		if cfg.SASLUser == "" || cfg.SASLPassword == "" {
			return nil, fmt.Errorf("SASL SCRAM requires KAFKA_SASL_USER and KAFKA_SASL_PASSWORD")
		}
		var scramAuth sasl.Mechanism
		if cfg.SASLMechanism == "scram-sha-256" {
			scramAuth = scram.Auth{
				User: cfg.SASLUser,
				Pass: cfg.SASLPassword,
			}.AsSha256Mechanism()
		} else {
			scramAuth = scram.Auth{
				User: cfg.SASLUser,
				Pass: cfg.SASLPassword,
			}.AsSha512Mechanism()
		}
		saslMethod = scramAuth
		slog.Info("SASL SCRAM mechanism configured", "type", cfg.SASLMechanism)
	case "":
		// SASL disabled, do nothing
		slog.Info("SASL disabled")
	default:
		return nil, fmt.Errorf("unsupported SASL mechanism: %s", cfg.SASLMechanism)
	}

	if saslMethod != nil {
		opts = append(opts, kgo.SASL(saslMethod))
	}

	// --- Create Client ---
	cl, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client: %w", err)
	}

	// Test connectivity on startup (optional, adds delay but catches config errors early)
	if err := cl.Ping(context.Background()); err != nil {
		cl.Close() // Close the client if ping fails
		return nil, fmt.Errorf("failed to ping Kafka brokers: %w", err)
	}
	slog.Info("Successfully pinged Kafka brokers")

	return &Client{kgoClient: cl}, nil
}

// ProduceMessage sends a message to the specified Kafka topic.
func (c *Client) ProduceMessage(ctx context.Context, topic string, key, value []byte) error {
	record := &kgo.Record{
		Topic: topic,
		Key:   key,
		Value: value,
	}
	// ProduceSync waits for the record to be acknowledged by Kafka.
	// For higher throughput, consider Produce which is asynchronous.
	err := c.kgoClient.ProduceSync(ctx, record).FirstErr()
	return err // Will be nil if producing was successful
}

// ConsumeMessages polls for messages from the specified topics.
// It requires the client to have been initialized with consumer group options.
// It returns a slice of consumed messages or an error.
func (c *Client) ConsumeMessages(ctx context.Context, topics []string, maxMessages int) ([]Message, error) {
	// Note: This function assumes the kgo.Client was already configured with
	// kgo.ConsumerGroup(...) during its creation in NewClient.

	// Add the requested topics to the subscription.
	// Note: These topics will remain in the subscription until the client is closed
	// or they are explicitly removed later. This might be desired behavior for an MCP server.
	c.kgoClient.AddConsumeTopics(topics...)

	slog.InfoContext(ctx, "Polling for messages", "topics", topics, "maxMessages", maxMessages)

	// Poll for fetches.
	fetches := c.kgoClient.PollFetches(ctx)
	if fetches.IsClientClosed() {
		return nil, fmt.Errorf("kafka client is closed")
	}

	// Check for general fetch errors
	errs := fetches.Errors()
	if len(errs) > 0 {
		firstErr := errs[0]
		return nil, fmt.Errorf("fetch error on topic %q (partition %d): %w", firstErr.Topic, firstErr.Partition, firstErr.Err)
	}

	consumedMessages := make([]Message, 0, maxMessages)
	iter := fetches.RecordIter()
	count := 0
	for !iter.Done() && count < maxMessages {
		rec := iter.Next()

		consumedMessages = append(consumedMessages, Message{
			Topic:     rec.Topic,
			Partition: rec.Partition,
			Offset:    rec.Offset,
			Key:       string(rec.Key),
			Value:     string(rec.Value),
			Timestamp: rec.Timestamp.UnixMilli(),
		})
		count++
	}

	slog.InfoContext(ctx, "Finished polling", "messagesConsumed", len(consumedMessages))

	// Note: Committing offsets is crucial for consumer groups.
	// kgo.Client handles auto-committing by default if group ID is set at init.
	// If manual commit is needed (e.g., after processing), use:
	// err := c.kgoClient.CommitRecords(ctx, fetches.Records()...)
	// Handle commit error appropriately.

	return consumedMessages, nil
}

// ListTopics retrieves a list of topic names from the Kafka cluster.
func (c *Client) ListTopics(ctx context.Context) ([]string, error) {
	// Use kmsg.NewMetadataRequest() to create the request struct
	req := kmsg.NewMetadataRequest()
	req.Topics = nil // Request metadata for all topics

	// RequestSharded returns *kgo.ShardedRequestPromise
	// We need to await its result.
	shardedResp := c.kgoClient.RequestSharded(ctx, &req)

	topicSet := make(map[string]struct{}) // Use map for deduplication

	// Iterate through responses from each broker shard
	// The response type in the shard is *kmsg.MetadataResponse
	for _, shard := range shardedResp {
		if shard.Err != nil {
			// Get broker ID from shard metadata if available
			brokerID := shard.Meta.NodeID // Access NodeID directly
			return nil, fmt.Errorf("metadata request failed for broker %d: %w", brokerID, shard.Err)
		}

		// Cast the response to the correct type
		resp, ok := shard.Resp.(*kmsg.MetadataResponse)
		if !ok {
			// This shouldn't happen if the request was kmsg.MetadataRequest
			return nil, fmt.Errorf("unexpected response type for metadata request from broker %d", shard.Meta.NodeID)
		}

		// Process topics from this broker's response
		for _, topic := range resp.Topics {
			if topic.ErrorCode != 0 {
				// Log topic-specific errors using kerr
				errMsg := kerr.ErrorForCode(topic.ErrorCode)
				// Dereference topic name pointer for logging, handle nil case
				topicName := "unknown"
				if topic.Topic != nil {
					topicName = *topic.Topic
				}
				slog.WarnContext(ctx, "Error fetching metadata for topic", "topic", topicName, "broker", shard.Meta.NodeID, "error_code", topic.ErrorCode, "error", errMsg.Error())
				continue // Skip topics with errors
			}
			// Dereference topic name pointer for map key, ensure not nil
			if topic.Topic != nil {
				// Ignore internal topics if desired (e.g., __consumer_offsets)
				// if strings.HasPrefix(*topic.Topic, "__") { continue }
				topicSet[*topic.Topic] = struct{}{}
			}
		}
	}

	topics := make([]string, 0, len(topicSet))
	for topic := range topicSet {
		topics = append(topics, topic)
	}

	return topics, nil
}

// DescribeTopic retrieves detailed metadata for a specific topic.
func (c *Client) DescribeTopic(ctx context.Context, topicName string) (*TopicMetadata, error) {
	req := kmsg.NewMetadataRequest()
	topicReq := kmsg.NewMetadataRequestTopic()
	topicReq.Topic = kmsg.StringPtr(topicName)
	req.Topics = append(req.Topics, topicReq)
	req.AllowAutoTopicCreation = false // Don't create if it doesn't exist

	shardedResp := c.kgoClient.RequestSharded(ctx, &req)

	var finalMetadata *TopicMetadata
	var firstError error

	for _, shard := range shardedResp {
		if shard.Err != nil {
			slog.ErrorContext(ctx, "Metadata request shard error", "broker", shard.Meta.NodeID, "error", shard.Err)
			if firstError == nil {
				firstError = fmt.Errorf("metadata request failed for broker %d: %w", shard.Meta.NodeID, shard.Err)
			}
			continue // Try other brokers if possible
		}

		resp, ok := shard.Resp.(*kmsg.MetadataResponse)
		if !ok {
			slog.ErrorContext(ctx, "Unexpected response type for metadata request", "broker", shard.Meta.NodeID)
			if firstError == nil {
				firstError = fmt.Errorf("unexpected metadata response type from broker %d", shard.Meta.NodeID)
			}
			continue
		}

		// Find the requested topic in the response
		for _, topic := range resp.Topics {
			if topic.Topic != nil && *topic.Topic == topicName {
				meta := &TopicMetadata{
					TopicName:  topicName,
					IsInternal: topic.IsInternal,
					ErrorCode:  topic.ErrorCode,
				}
				if topic.ErrorCode != 0 {
					errMsg := kerr.ErrorForCode(topic.ErrorCode)
					meta.ErrorMessage = errMsg.Error()
					// If we get a topic-level error, return it immediately
					return meta, fmt.Errorf("error describing topic '%s': %w", topicName, errMsg)
				}

				meta.Partitions = make([]PartitionMetadata, 0, len(topic.Partitions))
				for _, p := range topic.Partitions {
					pMeta := PartitionMetadata{
						PartitionID: p.Partition,
						Leader:      p.Leader,
						Replicas:    p.Replicas,
						ISR:         p.ISR,
						ErrorCode:   p.ErrorCode,
					}
					if p.ErrorCode != 0 {
						pErrMsg := kerr.ErrorForCode(p.ErrorCode)
						pMeta.ErrorMessage = pErrMsg.Error()
						slog.WarnContext(ctx, "Error describing partition", "topic", topicName, "partition", p.Partition, "error", pErrMsg)
					}
					meta.Partitions = append(meta.Partitions, pMeta)
				}
				finalMetadata = meta
				// Found the topic, no need to check other shards for this topic's data
				goto foundTopic
			}
		}
	}

foundTopic:
	if finalMetadata != nil {
		return finalMetadata, nil
	}

	// If loop finished and finalMetadata is still nil, the topic wasn't found or there was an error
	if firstError != nil {
		return nil, firstError // Return the first broker communication error encountered
	}

	// If no communication errors but topic not found, return a specific error
	return nil, fmt.Errorf("topic '%s' not found in metadata response", topicName)
}

// ListConsumerGroups retrieves a list of consumer groups known by the cluster.
func (c *Client) ListConsumerGroups(ctx context.Context) ([]ConsumerGroupInfo, error) {
	req := kmsg.NewListGroupsRequest()
	// Requesting groups in all states
	// req.StatesFilter = []string{"Stable", "PreparingRebalance", "CompletingRebalance", "Dead", "Empty"}

	// RequestSharded sends the request to all brokers and aggregates results.
	// For ListGroups, it's often sufficient to ask one broker (usually the coordinator),
	// but RequestSharded handles finding a suitable broker.
	shardedResp := c.kgoClient.RequestSharded(ctx, &req)

	var allGroups []ConsumerGroupInfo
	var firstError error
	processedGroups := make(map[string]struct{}) // Deduplicate groups listed by multiple brokers

	for _, shard := range shardedResp {
		if shard.Err != nil {
			slog.ErrorContext(ctx, "ListGroups request shard error", "broker", shard.Meta.NodeID, "error", shard.Err)
			if firstError == nil {
				firstError = fmt.Errorf("list groups request failed for broker %d: %w", shard.Meta.NodeID, shard.Err)
			}
			continue // Try other brokers
		}

		resp, ok := shard.Resp.(*kmsg.ListGroupsResponse)
		if !ok {
			slog.ErrorContext(ctx, "Unexpected response type for ListGroups request", "broker", shard.Meta.NodeID)
			if firstError == nil {
				firstError = fmt.Errorf("unexpected list groups response type from broker %d", shard.Meta.NodeID)
			}
			continue
		}

		if resp.ErrorCode != 0 {
			errMsg := kerr.ErrorForCode(resp.ErrorCode)
			slog.ErrorContext(ctx, "ListGroups request failed with error code", "broker", shard.Meta.NodeID, "error_code", resp.ErrorCode, "error", errMsg)
			if firstError == nil {
				firstError = fmt.Errorf("list groups request failed on broker %d: %w", shard.Meta.NodeID, errMsg)
			}
			continue // Potentially try other brokers if it's a transient error
		}

		for _, group := range resp.Groups {
			if _, exists := processedGroups[group.Group]; !exists {
				allGroups = append(allGroups, ConsumerGroupInfo{
					GroupID: group.Group,
					State:   group.GroupState, // Use GroupState field (added in v11)
					// Note: ListGroupsResponse itself doesn't have per-group error codes
				})
				processedGroups[group.Group] = struct{}{}
			}
		}
	}

	// If we collected groups, return them even if some brokers failed
	if len(allGroups) > 0 {
		return allGroups, nil
	}

	// If no groups were collected, return the first error encountered
	if firstError != nil {
		return nil, firstError
	}

	// No errors, but no groups found
	return []ConsumerGroupInfo{}, nil
}

// DescribeConsumerGroup retrieves detailed information about a specific consumer group,
// including members and optionally their committed offsets and lag.
func (c *Client) DescribeConsumerGroup(ctx context.Context, groupID string, includeOffsets bool) (*DescribeConsumerGroupResult, error) {
	// 1. Describe the Group to get state, protocol, members
	descReq := kmsg.NewDescribeGroupsRequest()
	descReq.Groups = []string{groupID}
	descReq.IncludeAuthorizedOperations = false

	// DescribeGroups needs to be sent to the group coordinator.
	// kgo.Client handles finding the coordinator automatically.
	resp, err := c.kgoClient.Request(ctx, &descReq)
	if err != nil {
		return nil, fmt.Errorf("DescribeGroups request failed for group '%s': %w", groupID, err)
	}

	descResp, ok := resp.(*kmsg.DescribeGroupsResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type for DescribeGroups request for group '%s'", groupID)
	}

	if len(descResp.Groups) != 1 {
		// Check if the group just doesn't exist (common case)
		if len(descResp.Groups) == 0 {
			return nil, fmt.Errorf("consumer group '%s' not found or not described", groupID)
		}
		return nil, fmt.Errorf("expected exactly one group in DescribeGroups response for group '%s', got %d", groupID, len(descResp.Groups))
	}

	group := descResp.Groups[0]
	result := &DescribeConsumerGroupResult{
		GroupID:      group.Group,
		State:        group.State,
		ProtocolType: group.ProtocolType,
		Protocol:     group.Protocol,
		// CoordinatorID: group.Coordinator, // Field removed
		ErrorCode: group.ErrorCode,
	}

	if group.ErrorCode != 0 {
		errMsg := kerr.ErrorForCode(group.ErrorCode)
		result.ErrorMessage = errMsg.Error()
		// Return partial result with error if group description failed
		return result, fmt.Errorf("error describing group '%s': %w", groupID, errMsg)
	}

	result.Members = make([]GroupMemberInfo, 0, len(group.Members))
	for _, member := range group.Members {
		result.Members = append(result.Members, GroupMemberInfo{
			MemberID:   member.MemberID,
			ClientID:   member.ClientID,
			ClientHost: member.ClientHost,
		})
	}

	// 2. Optionally Fetch Offsets and Calculate Lag
	if includeOffsets {
		offsets, err := c.fetchGroupOffsetsAndLag(ctx, groupID)
		if err != nil {
			// Log the error but return the group description info anyway
			slog.ErrorContext(ctx, "Failed to fetch offsets/lag for group", "group", groupID, "error", err)
			// Append warning to error message if it exists
			warningMsg := fmt.Sprintf("(Warning: Failed to fetch offsets: %v)", err)
			if result.ErrorMessage != "" {
				result.ErrorMessage = result.ErrorMessage + " " + warningMsg
			} else {
				result.ErrorMessage = warningMsg
			}
		} else {
			result.Offsets = offsets
		}
	}

	return result, nil
}

// fetchGroupOffsetsAndLag is a helper to get committed offsets and calculate lag.
func (c *Client) fetchGroupOffsetsAndLag(ctx context.Context, groupID string) ([]PartitionOffsetInfo, error) {
	// 1. Fetch committed offsets
	offsetReq := kmsg.NewOffsetFetchRequest()
	offsetReq.Group = groupID
	offsetReq.Topics = nil // Request all topics/partitions for the group

	// OffsetFetch also needs to go to the coordinator.
	offsetRespGeneric, err := c.kgoClient.Request(ctx, &offsetReq)
	if err != nil {
		return nil, fmt.Errorf("OffsetFetch request failed: %w", err)
	}
	offsetResp, ok := offsetRespGeneric.(*kmsg.OffsetFetchResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type for OffsetFetch request")
	}
	if offsetResp.ErrorCode != 0 {
		errMsg := kerr.ErrorForCode(offsetResp.ErrorCode)
		// Handle group-level errors like coordinator loading or not coordinator
		return nil, fmt.Errorf("OffsetFetch request failed for group '%s' with error code %d: %w", groupID, offsetResp.ErrorCode, errMsg)
	}

	if len(offsetResp.Topics) == 0 {
		return []PartitionOffsetInfo{}, nil // No offsets committed or group is inactive/empty
	}

	// 2. Prepare to fetch Log End Offsets (LEOs) to calculate lag
	leoReqTopics := make(map[string]*kmsg.ListOffsetsRequestTopic)  // topic -> request topic struct
	partitionMap := make(map[string]map[int32]*PartitionOffsetInfo) // topic -> partition -> info

	for _, topic := range offsetResp.Topics {
		if _, exists := partitionMap[topic.Topic]; !exists {
			partitionMap[topic.Topic] = make(map[int32]*PartitionOffsetInfo)
		}
		var reqTopic *kmsg.ListOffsetsRequestTopic
		if rt, exists := leoReqTopics[topic.Topic]; !exists {
			// Correctly create and assign pointer
			newTopic := kmsg.NewListOffsetsRequestTopic()
			newTopic.Topic = topic.Topic
			reqTopic = &newTopic // Assign address of the new struct
			leoReqTopics[topic.Topic] = reqTopic
		} else {
			reqTopic = rt
		}

		for _, p := range topic.Partitions {
			// Safely dereference metadata pointer
			metadataStr := ""
			if p.Metadata != nil {
				metadataStr = *p.Metadata
			}

			pInfo := PartitionOffsetInfo{
				Topic:        topic.Topic,
				Partition:    p.Partition,
				CommitOffset: p.Offset,
				Metadata:     metadataStr,
				ErrorCode:    p.ErrorCode,
				Lag:          -1, // Default to -1 (unknown/error)
			}
			if p.ErrorCode != 0 {
				pErrMsg := kerr.ErrorForCode(p.ErrorCode)
				pInfo.ErrorMessage = pErrMsg.Error()
			} else if p.Offset == -1 {
				// Offset -1 means no offset committed for this partition
				pInfo.Lag = 0 // Lag is effectively 0 if nothing is committed
			} else {
				// Only request LEO if commit offset is valid
				reqPartition := kmsg.NewListOffsetsRequestTopicPartition()
				reqPartition.Partition = p.Partition
				reqPartition.Timestamp = -1 // -1 requests the Log End Offset (LEO)
				reqTopic.Partitions = append(reqTopic.Partitions, reqPartition)
			}
			partitionMap[topic.Topic][p.Partition] = &pInfo
		}
	}

	// 3. Fetch LEOs using kmsg.ListOffsetsRequest and RequestSharded
	if len(leoReqTopics) > 0 {
		leoReq := kmsg.NewListOffsetsRequest()
		leoReq.ReplicaID = -1 // Standard client request
		for _, topicReq := range leoReqTopics {
			if len(topicReq.Partitions) > 0 { // Only add topics if they have partitions needing LEO
				leoReq.Topics = append(leoReq.Topics, *topicReq)
			}
		}

		// ListOffsets should be sharded as partitions can be on different brokers
		shardedLeoResp := c.kgoClient.RequestSharded(ctx, &leoReq)
		var leoErrors []error
		leoResults := make(map[string]map[int32]kmsg.ListOffsetsResponseTopicPartition) // topic -> partition -> response partition

		for _, shard := range shardedLeoResp {
			if shard.Err != nil {
				err := fmt.Errorf("ListOffsets shard request failed for broker %d: %w", shard.Meta.NodeID, shard.Err)
				slog.ErrorContext(ctx, err.Error())
				leoErrors = append(leoErrors, err)
				continue
			}
			leoResp, ok := shard.Resp.(*kmsg.ListOffsetsResponse)
			if !ok {
				err := fmt.Errorf("unexpected response type for ListOffsets from broker %d", shard.Meta.NodeID)
				slog.ErrorContext(ctx, err.Error())
				leoErrors = append(leoErrors, err)
				continue
			}

			for _, topic := range leoResp.Topics {
				if _, exists := leoResults[topic.Topic]; !exists {
					leoResults[topic.Topic] = make(map[int32]kmsg.ListOffsetsResponseTopicPartition)
				}
				for _, p := range topic.Partitions {
					leoResults[topic.Topic][p.Partition] = p // Store the partition response
				}
			}
		}

		// 4. Calculate Lag using LEO results
		for topicName, partitions := range partitionMap {
			for partitionID, pInfo := range partitions {
				if pInfo.CommitOffset >= 0 && pInfo.ErrorCode == 0 { // Only calculate if commit was valid
					if leoTopicResults, topicOk := leoResults[topicName]; topicOk {
						if leoPartitionResult, pOk := leoTopicResults[partitionID]; pOk {
							if leoPartitionResult.ErrorCode == 0 {
								pInfo.Lag = leoPartitionResult.Offset - pInfo.CommitOffset
								if pInfo.Lag < 0 {
									pInfo.Lag = 0 // Avoid negative lag
								}
							} else {
								leoErrMsg := kerr.ErrorForCode(leoPartitionResult.ErrorCode)
								slog.WarnContext(ctx, "Error getting LEO for partition", "group", groupID, "topic", topicName, "partition", partitionID, "error", leoErrMsg)
								pInfo.ErrorMessage = fmt.Sprintf("Lag calculation failed: %s", leoErrMsg.Error())
							}
						} else {
							slog.WarnContext(ctx, "LEO result missing for partition", "group", groupID, "topic", topicName, "partition", partitionID)
							pInfo.ErrorMessage = "Lag calculation failed: LEO result missing"
						}
					} else {
						// This case might happen if ListOffsets failed entirely for the brokers holding these partitions
						slog.WarnContext(ctx, "LEO results missing for topic", "group", groupID, "topic", topicName)
						pInfo.ErrorMessage = "Lag calculation failed: LEO results missing for topic"
					}
				}
			}
		}
		if len(leoErrors) > 0 {
			// Optionally return an aggregate error, but for now, lag is just -1 or has specific error message
		}
	}

	// 5. Flatten the results
	finalOffsets := make([]PartitionOffsetInfo, 0)
	for _, partitions := range partitionMap {
		for _, pInfo := range partitions {
			finalOffsets = append(finalOffsets, *pInfo)
		}
	}

	return finalOffsets, nil
}

// DescribeConfigs fetches configuration entries for a given resource (topic or broker).
func (c *Client) DescribeConfigs(ctx context.Context, resourceType kmsg.ConfigResourceType, resourceName string, configKeys []string) (*DescribeConfigsResult, error) {
	req := kmsg.NewDescribeConfigsRequest()
	res := kmsg.NewDescribeConfigsRequestResource()
	res.ResourceType = resourceType
	res.ResourceName = resourceName
	if len(configKeys) > 0 {
		res.ConfigNames = configKeys
	} // If empty, requests all non-default configs
	req.Resources = append(req.Resources, res)
	req.IncludeSynonyms = false      // Keep it simple for now
	req.IncludeDocumentation = false // Keep it simple for now

	// DescribeConfigs should be sent to the appropriate broker (controller for broker configs, any broker for topic configs).
	// RequestSharded handles routing correctly.
	shardedResp := c.kgoClient.RequestSharded(ctx, &req)

	var finalResult *DescribeConfigsResult
	var firstError error

	for _, shard := range shardedResp {
		if shard.Err != nil {
			slog.ErrorContext(ctx, "DescribeConfigs request shard error", "broker", shard.Meta.NodeID, "error", shard.Err)
			if firstError == nil {
				firstError = fmt.Errorf("DescribeConfigs request failed for broker %d: %w", shard.Meta.NodeID, shard.Err)
			}
			continue // Try other brokers if possible
		}

		resp, ok := shard.Resp.(*kmsg.DescribeConfigsResponse)
		if !ok {
			slog.ErrorContext(ctx, "Unexpected response type for DescribeConfigs request", "broker", shard.Meta.NodeID)
			if firstError == nil {
				firstError = fmt.Errorf("unexpected DescribeConfigs response type from broker %d", shard.Meta.NodeID)
			}
			continue
		}

		// Expecting one resource result per shard usually, but loop just in case
		for _, resResult := range resp.Resources {
			// Match the resource we requested
			if resResult.ResourceType == resourceType && resResult.ResourceName == resourceName {
				result := &DescribeConfigsResult{
					ResourceType: resourceType.String(), // Convert enum to string
					ResourceName: resourceName,
					ErrorCode:    resResult.ErrorCode,
				}
				if resResult.ErrorCode != 0 {
					errMsg := kerr.ErrorForCode(resResult.ErrorCode)
					result.ErrorMessage = errMsg.Error()
					// Return immediately if the resource itself had an error
					return result, fmt.Errorf("error describing configs for %s '%s': %w", resourceType.String(), resourceName, errMsg)
				}

				result.Configs = make([]ConfigEntry, 0, len(resResult.Configs))
				for _, cfg := range resResult.Configs {
					// Safely dereference value pointer
					valStr := ""
					if cfg.Value != nil {
						valStr = *cfg.Value
					}
					result.Configs = append(result.Configs, ConfigEntry{
						Name:        cfg.Name,
						Value:       valStr,
						IsDefault:   cfg.IsDefault,
						IsReadOnly:  cfg.ReadOnly,
						IsSensitive: cfg.IsSensitive,
						Source:      cfg.Source.String(), // Convert enum to string
					})
				}
				finalResult = result
				// Found the resource, no need to check other shards for this resource
				goto foundResource
			}
		}
	}

foundResource:
	if finalResult != nil {
		return finalResult, nil
	}

	// If loop finished and finalResult is still nil, the resource wasn't found or there was an error
	if firstError != nil {
		return nil, firstError // Return the first broker communication error encountered
	}

	// If no communication errors but resource not found in any response
	return nil, fmt.Errorf("resource %s '%s' not found in DescribeConfigs response", resourceType.String(), resourceName)
}

// GetClusterOverview retrieves high-level cluster health information.
func (c *Client) GetClusterOverview(ctx context.Context) (*ClusterOverviewResult, error) {
	req := kmsg.NewMetadataRequest()
	req.Topics = nil // Request metadata for all topics
	req.AllowAutoTopicCreation = false

	shardedResp := c.kgoClient.RequestSharded(ctx, &req)

	overview := &ClusterOverviewResult{
		ControllerID: -1, // Default if not found
	}
	var firstError error
	offlineBrokers := make(map[int32]struct{})
	topicSet := make(map[string]struct{}) // To count unique non-internal topics

	for _, shard := range shardedResp {
		if shard.Err != nil {
			slog.ErrorContext(ctx, "Cluster overview metadata shard error", "broker", shard.Meta.NodeID, "error", shard.Err)
			offlineBrokers[shard.Meta.NodeID] = struct{}{}
			if firstError == nil {
				firstError = fmt.Errorf("metadata request failed for broker %d: %w", shard.Meta.NodeID, shard.Err)
			}
			continue // Continue processing results from other brokers
		}

		resp, ok := shard.Resp.(*kmsg.MetadataResponse)
		if !ok {
			slog.ErrorContext(ctx, "Unexpected response type for cluster overview metadata", "broker", shard.Meta.NodeID)
			if firstError == nil {
				firstError = fmt.Errorf("unexpected metadata response type from broker %d", shard.Meta.NodeID)
			}
			continue
		}

		// Update controller ID if found (should be consistent across responses)
		if resp.ControllerID != -1 {
			overview.ControllerID = resp.ControllerID
		}

		// Count brokers (implicitly includes brokers that responded)
		// Note: This might slightly undercount if a broker is up but fails the metadata request specifically.
		// A more accurate count might require a separate DescribeCluster request if available/needed.
		overview.BrokerCount = len(resp.Brokers)

		// Process topics and partitions
		for _, topic := range resp.Topics {
			// Skip topics with errors at the topic level
			if topic.ErrorCode != 0 {
				errMsg := kerr.ErrorForCode(topic.ErrorCode)
				topicName := "unknown"
				if topic.Topic != nil {
					topicName = *topic.Topic
				}
				slog.WarnContext(ctx, "Skipping topic in overview due to error", "topic", topicName, "error", errMsg)
				continue
			}

			if topic.Topic != nil {
				// Count unique non-internal topics
				if !topic.IsInternal {
					topicSet[*topic.Topic] = struct{}{}
				}

				for _, p := range topic.Partitions {
					overview.PartitionCount++

					// Check for offline partitions (e.g., no leader)
					if p.ErrorCode != 0 || p.Leader == -1 {
						overview.OfflinePartitionsCount++
						if p.ErrorCode != 0 {
							pErrMsg := kerr.ErrorForCode(p.ErrorCode)
							slog.DebugContext(ctx, "Offline partition detected", "topic", *topic.Topic, "partition", p.Partition, "error", pErrMsg)
						} else {
							slog.DebugContext(ctx, "Offline partition detected (no leader)", "topic", *topic.Topic, "partition", p.Partition)
						}
					}

					// Check for under-replicated partitions (ISR < Replicas)
					// Only check if the partition is not already offline
					if p.ErrorCode == 0 && p.Leader != -1 && len(p.ISR) < len(p.Replicas) {
						overview.UnderReplicatedPartitionsCount++
						slog.DebugContext(ctx, "Under-replicated partition detected", "topic", *topic.Topic, "partition", p.Partition, "isr_count", len(p.ISR), "replica_count", len(p.Replicas))
					}
				}
			}
		}
	}

	overview.TopicCount = len(topicSet)

	// Populate offline broker IDs
	if len(offlineBrokers) > 0 {
		overview.OfflineBrokerIDs = make([]int32, 0, len(offlineBrokers))
		for id := range offlineBrokers {
			overview.OfflineBrokerIDs = append(overview.OfflineBrokerIDs, id)
		}
	}

	// If metadata failed entirely for all brokers, return the first error
	if overview.ControllerID == -1 && firstError != nil {
		overview.ErrorMessage = fmt.Sprintf("Failed to retrieve complete cluster metadata: %v", firstError)
		// overview.ErrorCode could be set based on firstError if needed
		return overview, firstError
	}

	// If some brokers failed, include a warning in the message
	if firstError != nil {
		overview.ErrorMessage = fmt.Sprintf("Warning: Failed to retrieve metadata from some brokers: %v", firstError)
	}

	return overview, nil
}

// Close gracefully shuts down the Kafka client.
func (c *Client) Close() {
	if c.kgoClient != nil {
		c.kgoClient.Close()
	}
}
