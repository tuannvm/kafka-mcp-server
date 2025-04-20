## Metadata Tools in MCP‑Kafka Servers  
MCP‑Kafka servers typically expose a suite of metadata‑oriented “tools” that LLMs can invoke to inspect and manage Kafka clusters. Commonly supported operations include:  
- **list_brokers**: Returns broker IDs, hostnames, and port configurations.  
- **list_topics**: Retrieves all topic names along with partition counts and replication factors.  
- **describe_topic**: Provides per‑partition leader and replica assignments, ISR (in‑sync replicas), and retention settings.  
- **list_consumer_groups**: Enumerates active consumer groups and their member assignments.  
- **describe_consumer_group**: Shows each group’s partition offsets, lag metrics, and rebalance state.  
- **describe_configs**: Fetches topic‑ or broker‑level configuration entries (e.g., `min.insync.replicas`, `unclean.leader.election.enable`).  
- **cluster_overview**: Aggregates high‑level cluster health data, such as under‑replicated partitions, offline partitions, and controller status.
