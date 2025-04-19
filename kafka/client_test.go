package kafka

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/tuannvm/kafka-mcp-server/config"
	"github.com/twmb/franz-go/pkg/kgo"
)

var testKafkaContainer *kafka.KafkaContainer
var testKafkaBrokers []string

// setupKafkaContainer starts a Kafka container for integration tests.
func setupKafkaContainer(ctx context.Context) error {
	// Skip if running in CI without Docker, or if explicitly disabled
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_KAFKA_TESTS") == "true" {
		return fmt.Errorf("skipping Kafka container setup")
	}

	// Use default Kafka version provided by testcontainers-go
	kc, err := kafka.Run(ctx,
		"confluentinc/confluent-local:7.5.0",
		kafka.WithClusterID("test-cluster"),
	)
	if err != nil {
		return fmt.Errorf("failed to start Kafka container: %w", err)
	}
	testKafkaContainer = kc

	brokers, err := kc.Brokers(ctx)
	if err != nil {
		teardownKafkaContainer(context.Background()) // Attempt cleanup
		return fmt.Errorf("failed to get Kafka brokers: %w", err)
	}
	testKafkaBrokers = brokers
	fmt.Printf("Kafka container started with brokers: %s\n", strings.Join(brokers, ","))
	return nil
}

// teardownKafkaContainer stops the Kafka container.
func teardownKafkaContainer(ctx context.Context) {
	if testKafkaContainer != nil {
		fmt.Println("Terminating Kafka container...")
		if err := testKafkaContainer.Terminate(ctx); err != nil {
			fmt.Printf("Failed to terminate Kafka container: %v\n", err)
		}
		testKafkaContainer = nil
		testKafkaBrokers = nil
	}
}

// TestMain manages the Kafka container lifecycle for the test suite.
func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // Give ample time for container setup
	defer cancel()

	// Set a flag to indicate if Kafka container is available
	var kafkaAvailable bool = false

	// Try to set up Kafka container
	err := setupKafkaContainer(ctx)
	if err != nil {
		fmt.Printf("WARNING: Could not set up Kafka container, integration tests will be skipped: %v\n", err)
		// No need to fail - we'll just skip the tests that need Kafka
	} else {
		kafkaAvailable = true
	}

	// Run tests - they should check testKafkaBrokers/testKafkaContainer being nil
	exitCode := m.Run()

	// Teardown only if we successfully set up the container
	if kafkaAvailable {
		teardownKafkaContainer(context.Background())
	}

	os.Exit(exitCode)
}

// Helper function to create a test config pointing to the container
func getTestConfig() config.Config {
	return config.Config{
		KafkaBrokers:  testKafkaBrokers,
		KafkaClientID: "test-mcp-client",
		// Add SASL/TLS config here if testing those features
	}
}

func TestNewClient_Connection(t *testing.T) {
	if testKafkaContainer == nil {
		t.Skip("Skipping test: Kafka container not available")
	}
	require.NotEmpty(t, testKafkaBrokers, "Kafka brokers should be set by TestMain")

	cfg := getTestConfig()
	client, err := NewClient(cfg)

	require.NoError(t, err, "NewClient should connect successfully")
	require.NotNil(t, client, "Client should not be nil")
	defer client.Close()

	// Ping is already done in NewClient, but we can assert no error occurred
	assert.NoError(t, err)
}

func TestProduceConsume_RoundTrip(t *testing.T) {
	if testKafkaContainer == nil {
		t.Skip("Skipping test: Kafka container not available")
	}
	require.NotEmpty(t, testKafkaBrokers, "Kafka brokers should be set by TestMain")

	cfg := getTestConfig()
	client, err := NewClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	ctx := context.Background()
	topic := "test-produce-consume"
	key := "test-key-1"
	value := "hello kafka " + time.Now().String()

	// Produce
	err = client.ProduceMessage(ctx, topic, []byte(key), []byte(value))
	require.NoError(t, err, "ProduceMessage should succeed")

	// Consume
	// Give Kafka a moment to process the message
	time.Sleep(2 * time.Second)

	// ConsumeMessages requires the client to be configured as a consumer group
	// NewClient already configures ConsumerGroup(cfg.KafkaClientID)
	messages, err := client.ConsumeMessages(ctx, []string{topic}, 1)
	require.NoError(t, err, "ConsumeMessages should succeed")
	require.Len(t, messages, 1, "Should consume exactly one message")

	consumedMsg := messages[0]
	assert.Equal(t, topic, consumedMsg.Topic)
	assert.Equal(t, key, consumedMsg.Key)
	assert.Equal(t, value, consumedMsg.Value)
	assert.True(t, consumedMsg.Offset >= 0, "Offset should be non-negative")
	assert.True(t, consumedMsg.Timestamp > 0, "Timestamp should be positive")

	// Optional: Commit offsets if auto-commit is disabled (it is in NewClient)
	err = client.kgoClient.CommitUncommittedOffsets(ctx)
	assert.NoError(t, err, "Committing offsets should succeed")
}

func TestListTopics(t *testing.T) {
	if testKafkaContainer == nil {
		t.Skip("Skipping test: Kafka container not available")
	}
	require.NotEmpty(t, testKafkaBrokers, "Kafka brokers should be set by TestMain")

	cfg := getTestConfig()
	// Use a separate client ID to avoid consumer group interactions if not needed
	cfg.KafkaClientID = "test-list-topics-client"
	// Re-create client without consumer group for listing
	clientOpts := []kgo.Opt{
		kgo.SeedBrokers(cfg.KafkaBrokers...),
		kgo.ClientID(cfg.KafkaClientID),
		kgo.WithLogger(kgo.BasicLogger(os.Stderr, kgo.LogLevelWarn, nil)), // Reduce log level for this test
	}
	kgoClient, err := kgo.NewClient(clientOpts...)
	require.NoError(t, err)
	client := &Client{kgoClient: kgoClient}
	defer client.Close()

	ctx := context.Background()

	// Produce to a topic first to ensure it exists
	topicToCreate := "topic-for-listing-test"
	err = client.ProduceMessage(ctx, topicToCreate, nil, []byte("dummy message"))
	require.NoError(t, err)
	time.Sleep(2 * time.Second) // Give Kafka time to create/register the topic

	topics, err := client.ListTopics(ctx)
	require.NoError(t, err, "ListTopics should succeed")
	require.NotEmpty(t, topics, "List of topics should not be empty")

	// Check if our created topic is in the list
	found := false
	for _, topic := range topics {
		if topic == topicToCreate {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected to find topic '%s' in the list %v", topicToCreate, topics)
}
