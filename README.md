# Kafka MCP Server

A comprehensive Model Context Protocol (MCP) server for Apache Kafka implemented in Go, using [franz-go](https://github.com/twmb/franz-go) and [mcp-go](https://github.com/mark3labs/mcp-go).

This server provides the most complete implementation available for interacting with Kafka via the MCP protocol, enabling AI assistants to perform a wide range of Kafka operations through a standardized interface.

[![Go Report Card](https://goreportcard.com/badge/github.com/tuannvm/kafka-mcp-server)](https://goreportcard.com/report/github.com/tuannvm/kafka-mcp-server)

## Overview

The Kafka MCP Server bridges the gap between AI assistants and Apache Kafka, allowing them to:

- Produce and consume messages
- List and describe topics
- Manage consumer groups
- Monitor cluster health
- And much more

All through the standardized Model Context Protocol (MCP).

## Key Features

- **Comprehensive Kafka Integration**: The most complete implementation of Kafka operations via MCP
- **Advanced Security**: Full support for SASL (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512) and TLS authentication
- **Enterprise-Ready**: Designed for production use with extensive configuration options
- **Fault Tolerance**: Robust error handling and recovery mechanisms
- **Intelligent Prompts**: Pre-configured prompts for common Kafka operations
- **High Performance**: Built with Go for maximum efficiency and throughput

## Getting Started

### Prerequisites

- Go 1.21 or later
- Docker (for running integration tests)
- Access to a Kafka cluster

### Installation

```bash
# Clone the repository
git clone https://github.com/tuannvm/kafka-mcp-server.git
cd kafka-mcp-server

# Build the server
go build -o kafka-mcp-server ./cmd/server
```

### Quick Start

```bash
# Basic setup with local Kafka
export KAFKA_BROKERS="localhost:9092"
./kafka-mcp-server

# Enterprise setup with security
export KAFKA_BROKERS="kafka-broker-1:9092,kafka-broker-2:9092"
export KAFKA_SASL_MECHANISM="scram-sha-512"
export KAFKA_SASL_USER="kafka-user"
export KAFKA_SASL_PASSWORD="kafka-password"
export KAFKA_TLS_ENABLE="true"
./kafka-mcp-server
```

## MCP Tools

The server exposes a rich set of tools that enable AI assistants to interact with Kafka. Here's a detailed overview:

### produce_message

Produces a single message to a Kafka topic with optional key.

**Sample Prompt:**
> "Send a new order update to the orders topic with order ID 12345."

**Example:**
```json
{
  "topic": "orders",
  "key": "12345",
  "value": "{\"order_id\":\"12345\",\"status\":\"shipped\"}"
}
```

**Response:**
```json
"Message produced successfully to topic orders"
```

### consume_messages

Consumes a batch of messages from one or more Kafka topics.

**Sample Prompt:**
> "Retrieve the latest messages from the customer-events topic so I can see recent customer activity."

**Example:**
```json
{
  "topics": ["customer-events"],
  "max_messages": 5
}
```

**Response:**
```json
[
  {
    "topic": "customer-events",
    "partition": 0,
    "offset": 1042,
    "timestamp": 1650123456789,
    "key": "customer-123",
    "value": "{\"customer_id\":\"123\",\"action\":\"login\",\"timestamp\":\"2023-04-16T12:34:56Z\"}"
  },
  // Additional messages...
]
```

### list_brokers

Lists the configured Kafka broker addresses the server is connected to.

**Sample Prompt:**
> "What Kafka brokers do we have available in our cluster?"

**Example:**
```json
{}
```

**Response:**
```json
[
  "kafka-broker-1:9092",
  "kafka-broker-2:9092",
  "kafka-broker-3:9092"
]
```

### describe_topic

Provides detailed metadata for a specific Kafka topic.

**Sample Prompt:**
> "Show me the configuration and partition details for our orders topic."

**Example:**
```json
{
  "topic_name": "orders"
}
```

**Response:**
```json
{
  "name": "orders",
  "partitions": [
    {
      "partitionID": 0,
      "leader": 1,
      "replicas": [1, 2, 3],
      "ISR": [1, 2, 3],
      "errorCode": 0
    },
    {
      "partitionID": 1,
      "leader": 2,
      "replicas": [2, 3, 1],
      "ISR": [2, 3, 1],
      "errorCode": 0
    }
  ],
  "isInternal": false
}
```

### list_consumer_groups

Enumerates active consumer groups known by the Kafka cluster.

**Sample Prompt:**
> "What consumer groups are currently active in our Kafka cluster?"

**Example:**
```json
{}
```

**Response:**
```json
[
  {
    "groupID": "order-processor",
    "state": "Stable",
    "errorCode": 0
  },
  {
    "groupID": "analytics-pipeline",
    "state": "Stable",
    "errorCode": 0
  }
]
```

### describe_consumer_group

Shows details for a specific consumer group, including state, members, and partition offsets.

**Sample Prompt:**
> "Tell me about the order-processor consumer group. Are there any lagging consumers?"

**Example:**
```json
{
  "group_id": "order-processor",
  "include_offsets": true
}
```

**Response:**
```json
{
  "groupID": "order-processor",
  "state": "Stable",
  "members": [
    {
      "memberID": "consumer-1-uuid",
      "clientID": "consumer-1",
      "clientHost": "10.0.0.101",
      "assignments": [
        {"topic": "orders", "partitions": [0, 2, 4]}
      ]
    },
    {
      "memberID": "consumer-2-uuid",
      "clientID": "consumer-2",
      "clientHost": "10.0.0.102",
      "assignments": [
        {"topic": "orders", "partitions": [1, 3, 5]}
      ]
    }
  ],
  "offsets": [
    {
      "topic": "orders",
      "partition": 0,
      "commitOffset": 10045,
      "lag": 5
    },
    // More partitions...
  ],
  "errorCode": 0
}
```

### describe_configs

Fetches configuration entries for a specific resource (topic or broker).

**Sample Prompt:**
> "What's the retention configuration for our clickstream topic?"

**Example:**
```json
{
  "resource_type": "topic",
  "resource_name": "clickstream",
  "config_keys": ["retention.ms", "retention.bytes"]
}
```

**Response:**
```json
{
  "configs": [
    {
      "name": "retention.ms",
      "value": "604800000",
      "source": "DYNAMIC_TOPIC_CONFIG",
      "isSensitive": false,
      "isReadOnly": false
    },
    {
      "name": "retention.bytes",
      "value": "1073741824",
      "source": "DYNAMIC_TOPIC_CONFIG",
      "isSensitive": false,
      "isReadOnly": false
    }
  ]
}
```

### cluster_overview

Aggregates high-level cluster health data, including controller, brokers, topics, and partition status.

**Sample Prompt:**
> "Give me an overview of our Kafka cluster health."

**Example:**
```json
{}
```

**Response:**
```json
{
  "brokerCount": 3,
  "controllerID": 1,
  "topicCount": 24,
  "partitionCount": 120,
  "underReplicatedPartitionsCount": 0,
  "offlinePartitionsCount": 0,
  "offlineBrokerIDs": []
}
```

## MCP Resources

### list_topics

Lists the names of available topics in the Kafka cluster.

**Sample Prompt:**
> "What topics do we have in our Kafka cluster?"

**Example:**
```json
{}
```

**Response:**
```json
[
  "orders",
  "customers",
  "shipments",
  "inventory",
  "clickstream"
]
```

## MCP Prompts

The server includes a rich set of pre-configured prompts for common Kafka operations and diagnostics:

- **list_brokers**: Shows all broker IDs, hostnames, and ports
- **list_topics**: Lists all Kafka topics with partition counts
- **describe_topic**: Displays detailed information about a specific Kafka topic
- **cluster_overview**: Summarizes Kafka cluster health
- **list_consumer_groups**: Lists all active consumer groups
- **reset_offsets**: Provides instructions for resetting consumer group offsets
- **create_topic**: Creates a new Kafka topic with specified parameters
- **increase_partitions**: Increases the number of partitions for an existing topic
- **produce**: Publishes a message to a Kafka topic
- **consume**: Fetches messages from a Kafka topic
- **health_check**: Runs a comprehensive health check on the Kafka cluster
- **under_replicated**: Lists topics and partitions where ISR < replication factor
- **consumer_lag_report**: Provides a detailed report on consumer lag
- **monitor_consumer_group**: Provides commands for monitoring a consumer group
- **kafka_troubleshooting**: General troubleshooting guide for common Kafka issues

## End-to-End Example

Here's a complete interaction example showing how an AI assistant might use these tools to help a user:

**User Query:** "We need to investigate why our order processing pipeline is falling behind. Can you help me check the consumer lag?"

**AI Assistant's workflow:**

1. First, check the consumer groups
   ```
   > Using list_consumer_groups tool
   > Discovers "order-processor" group
   ```

2. Then, examine the consumer group details
   ```
   > Using describe_consumer_group tool with group_id "order-processor" and include_offsets=true
   > Discovers lag on several partitions
   ```

3. Get cluster overview to check general health
   ```
   > Using cluster_overview tool
   > Confirms no under-replicated partitions or offline brokers
   ```

4. Generate a consumer lag report
   ```
   > Using consumer_lag_report prompt with threshold=1000
   > Provides detailed analysis of lag across consumer groups
   ```

5. Provide actionable recommendations:
   ```
   Based on my analysis, your order-processor consumer group has significant lag on topic orders, partitions 2 and 3, with lag values of 15,420 and 12,305 messages respectively.

   The cluster itself is healthy with no under-replicated partitions, suggesting this is an issue with the consumer application rather than Kafka infrastructure.

   I recommend:
   1. Check the consumer instances processing those partitions for slowdowns or errors
   2. Consider scaling up your consumer group by adding more instances
   3. Review the consumer configuration, particularly max.poll.records and max.poll.interval.ms
   4. Examine any processing bottlenecks in the consumer logic
   ```

This seamless workflow demonstrates how the Kafka MCP tools enable AI assistants to perform sophisticated diagnostics and provide actionable insights.

## Configuration

The server can be configured using the following environment variables:

| Variable | Description | Default |
| :------- | :---------- | :------ |
| `KAFKA_BROKERS` | Comma-separated list of Kafka broker addresses | `localhost:9092` |
| `KAFKA_CLIENT_ID` | Kafka client ID used for connections | `kafka-mcp-server` |
| `MCP_TRANSPORT` | MCP transport method (stdio/http) | `stdio` |
| `KAFKA_SASL_MECHANISM` | SASL mechanism: `plain`, `scram-sha-256`, `scram-sha-512`, or `""` (disabled) | `""` |
| `KAFKA_SASL_USER` | Username for SASL authentication | `""` |
| `KAFKA_SASL_PASSWORD` | Password for SASL authentication | `""` |
| `KAFKA_TLS_ENABLE` | Enable TLS for Kafka connection (`true` or `false`) | `false` |
| `KAFKA_TLS_INSECURE_SKIP_VERIFY` | Skip TLS certificate verification (`true` or `false`) | `false` |

> **Security Note:** When using `KAFKA_TLS_INSECURE_SKIP_VERIFY=true`, the server will skip TLS certificate verification. This should only be used in development or testing environments, or when using self-signed certificates.

## Security Considerations

The server is designed with security in mind:

- **Authentication**: Full support for SASL PLAIN, SCRAM-SHA-256, and SCRAM-SHA-512
- **Encryption**: TLS support for secure communication with Kafka brokers
- **Input Validation**: Thorough validation of all user inputs to prevent injection attacks
- **Error Handling**: Secure error handling that doesn't expose sensitive information

## Development

### Testing

Comprehensive test coverage ensures reliability:

```bash
# Run all tests (requires Docker for integration tests)
go test ./...

# Run tests excluding integration tests
go test -short ./...

# Run integration tests with specific Kafka brokers
export KAFKA_BROKERS="your-broker:9092"
export SKIP_KAFKA_TESTS="false"
go test ./kafka -v -run Test
```

### Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
