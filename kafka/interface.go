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

	// Close gracefully shuts down the Kafka client.
	Close()
}

// Ensure Client implements KafkaClient
var _ KafkaClient = (*Client)(nil)
