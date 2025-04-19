package kafka

import (
	"context"
	"crypto/tls" // Added for TLS config
	"fmt"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	"github.com/twmb/franz-go/pkg/sasl" // Added for SASL interface
	"github.com/twmb/franz-go/pkg/sasl/plain" // Added for SASL PLAIN
	"github.com/twmb/franz-go/pkg/sasl/scram" // Added for SASL SCRAM
	"github.com/tuannvm/kafka-mcp-server/config"
	"os" // Added for os.Stderr
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
		kgo.ConsumeTopics(), // Initialize with empty topics, will be added in ConsumeMessages
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

// Close gracefully shuts down the Kafka client.
func (c *Client) Close() {
	if c.kgoClient != nil {
		c.kgoClient.Close()
	}
}
