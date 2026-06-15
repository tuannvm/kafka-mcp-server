// Package kafka provides Kafka client functionality for the MCP server.
package kafka

import (
	"context"
)

// KafkaClient defines the interface for Kafka operations.
// This interface makes it easier to create mock implementations for testing.
type KafkaClient interface {
	// ProduceMessage sends a message to the specified Kafka topic.
	ProduceMessage(ctx context.Context, topic string, key, value []byte) error

	// ConsumeMessages polls for messages from the specified topics.
	// It returns a slice of consumed messages or an error.
	ConsumeMessages(ctx context.Context, topics []string, maxMessages int) ([]Message, error)

	// ListTopics retrieves a list of topic names from the Kafka cluster.
	ListTopics(ctx context.Context) ([]string, error)

	// ListBrokers retrieves a list of broker addresses from the Kafka cluster.
	ListBrokers(ctx context.Context) ([]string, error) // Added ListBrokers method

	// DescribeTopic retrieves detailed metadata for a specific topic.
	DescribeTopic(ctx context.Context, topicName string) (*TopicMetadata, error)

	// ListConsumerGroups retrieves a list of consumer groups known by the cluster.
	ListConsumerGroups(ctx context.Context) ([]ConsumerGroupInfo, error)

	// DescribeConsumerGroup retrieves detailed information about a specific consumer group,
	// including members and optionally their committed offsets and lag.
	DescribeConsumerGroup(ctx context.Context, groupID string, includeOffsets bool) (*DescribeConsumerGroupResult, error)

	// DescribeConfigs retrieves configuration for a Kafka resource
	DescribeConfigs(ctx context.Context, resourceType ConfigResourceType, resourceName string, configKeys []string) (*DescribeConfigsResult, error)

	// GetClusterOverview retrieves high-level cluster health information.
	GetClusterOverview(ctx context.Context) (*ClusterOverviewResult, error)

	// StringToResourceType converts a string resource type to its corresponding ConfigResourceType
	StringToResourceType(resourceTypeStr string) (ConfigResourceType, error)

	// CreateTopic creates a new topic in the Kafka cluster.
	CreateTopic(ctx context.Context, topic string, partitions int32, replicationFactor int16) error

	// DeleteTopic deletes an existing topic in the Kafka cluster.
	DeleteTopic(ctx context.Context, topic string) error

	// ResetConsumerGroupOffsets resets the committed offsets for a consumer group on a specific topic.
	// The group must have no active consumers. Offset can be a specific value, -1 for latest, or -2 for earliest.
	ResetConsumerGroupOffsets(ctx context.Context, groupID string, topic string, partition int32, offset int64) error

	// Close gracefully shuts down the Kafka client.
	Close()
}

// Ensure Client implements KafkaClient
var _ KafkaClient = (*Client)(nil)
