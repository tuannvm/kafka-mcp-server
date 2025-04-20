## Useful Natural‑Language Prompts for Kafka‑MCP‑Server  

Below are categories of prompts you can support in a Kafka‑MCP‑Server. Each maps to one or more MCP “tools” (e.g., `list_brokers`, `describe_topic`, `cluster_overview`) and lets users or LLM agents interact with Kafka via plain language.

### 1. Cluster Metadata & Health Checks  
- “List all brokers with their hostnames and ports.”  
- “Show me an overview of cluster health: under‑replicated and offline partitions, controller status, and broker count.”  
- “Which topics are under‑replicated right now?”  

### 2. Topic Queries & Management  
- “List all topics with partition counts and replication factors.”  
- “Describe topic `<topic>`: show leader, replicas, and ISR per partition.”  
- “Create a new topic named `<name>` with `<n>` partitions and replication factor `<r>`.”  
- “Increase partitions on `<topic>` to `<n>`.”  
- “Update `<topic>` config: set `min.insync.replicas` to `<value>`.”  

### 3. Consumer Group Operations  
- “List all consumer groups and their members.”  
- “Describe consumer group `<group-id>`: show current offsets, end offsets, and lag per partition.”  
- “Reset offsets for group `<group-id>` on `<topic>` to `<earliest|latest|<offset>>`.”  

### 4. Message Production & Consumption  
- “Publish this message to `<topic>`: `<message-payload>`.”  
- “Fetch the last `<n>` messages from `<topic>`.”  
- “Consume messages from `<topic>` starting at offset `<offset>` for `<count>` records.”  

### 5. Security & Access Control  
- “List all ACLs configured for this cluster.”  
- “Show ACLs for resource `<resource>`, principal `<principal>`, and host `<host>`.”  
- “Add an ACL to allow `<principal>` to `<operation>` on topic `<topic>`.”  
- “Remove the ACL allowing `<principal>` to `<operation>` on `<resource>`.”  

### 6. Metrics & Monitoring  
- “Get throughput and latency metrics for `<topic>` over the last `<duration>`.”  
- “Show broker CPU and memory usage for broker `<broker-id>` in the past `<duration>`.”  
- “Alert if any consumer group lag exceeds `<threshold>` messages.”  

### 7. Custom Domain Tools  
- “Publish customer feedback: `<text>`.”  
- “Retrieve sales metrics for product `<product-id>`.”  
- “Summarize today’s error logs from the broker.”  

By mapping these prompts to MCP tool invocations—such as `resources/read`, `tools/invoke` with `list_topics`, or `cluster_overview`—your Kafka‑MCP‑Server enables intuitive, conversational control and monitoring of Kafka clusters without requiring users to write code.
