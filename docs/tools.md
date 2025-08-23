# MCP Tools

The server exposes tools for Kafka interaction:

## produce_message

Produces a single message to a specified Kafka topic. Use this tool when you need to send data, events, or notifications to a Kafka topic. The message can include an optional key for partitioning and routing.

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

## consume_messages

Consumes messages from one or more Kafka topics in a single batch operation. Use this tool to retrieve recent messages for analysis, monitoring, or processing. Messages are consumed from the latest available offsets.

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

## list_brokers

Lists all configured Kafka broker addresses that the server is connecting to. Use this tool to verify connectivity and understand the cluster topology. Returns the broker hostnames and ports as configured.

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

## describe_topic

Provides comprehensive metadata and configuration details for a specific Kafka topic. Returns information about partitions, replication factors, leaders, replicas, and in-sync replicas (ISRs). Use this tool to understand topic structure, troubleshoot replication issues, or verify topic configuration.

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

## list_consumer_groups

Enumerates all consumer groups known by the Kafka cluster, including their current states. Use this tool to discover active consumer applications, monitor consumer group health, or identify unused consumer groups. Returns group IDs, states, and error codes.

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

## describe_consumer_group

Provides detailed information about a specific consumer group including its current state, active members, partition assignments, and optionally offset and lag information. Use this tool to troubleshoot consumer lag, monitor group membership, or analyze partition distribution across consumers.

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

## describe_configs

Retrieves configuration settings for Kafka resources such as topics or brokers. Use this tool to examine retention policies, replication settings, segment sizes, cleanup policies, and other configuration parameters. Helps troubleshoot performance issues and verify configuration compliance.

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

## cluster_overview

Provides a comprehensive health summary of the entire Kafka cluster including broker status, controller information, topic and partition counts, and replication health metrics. Use this tool for cluster monitoring, health checks, and getting a quick overview of cluster state and potential issues.

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

## list_topics

Retrieves a complete list of all topics in the Kafka cluster along with their metadata including partition counts, replication factors, and internal topic flags. Use this tool to discover available topics, understand cluster topology, or inventory data streams in the cluster.

**Sample Prompt:**
> "What topics are available in our Kafka cluster?"

**Example:**
```json
{}
```

**Response:**
```json
[
  {
    "name": "orders",
    "partition_count": 6,
    "replication_factor": 3,
    "is_internal": false
  },
  {
    "name": "customer-events",
    "partition_count": 12,
    "replication_factor": 3,
    "is_internal": false
  },
  {
    "name": "__consumer_offsets",
    "partition_count": 50,
    "replication_factor": 3,
    "is_internal": true
  }
]
```