# MCP Prompts

The server includes the following pre-configured prompts for Kafka operations and diagnostics:

## kafka_cluster_overview

Generates a comprehensive, human-readable summary of Kafka cluster health including broker counts, controller status, topic/partition metrics, and replication health. Perfect for status reports, monitoring dashboards, and quick cluster assessments.

**Arguments:**
- None required - uses the configured cluster connection

**Example Response:**
```
# Kafka Cluster Overview

**Time**: 2023-08-15T12:34:56Z

- **Broker Count**: 3
- **Active Controller ID**: 1
- **Total Topics**: 24
- **Total Partitions**: 120
- **Under-Replicated Partitions**: 0
- **Offline Partitions**: 0

**Overall Status**: ✅ Healthy
```

## kafka_health_check

Performs a comprehensive health assessment of the Kafka cluster with detailed analysis of broker availability, controller status, partition health, and consumer group performance. Provides actionable recommendations for troubleshooting and maintenance activities.

**Arguments:**
- None required - uses the configured cluster connection

**Example Response:**
```
# Kafka Cluster Health Check Report

**Time**: 2023-08-15T12:34:56Z

## Broker Status

- ✅ **All 3 brokers are online**

## Controller Status

- ✅ **Active controller**: Broker 1

## Partition Health

- ✅ **All 120 partitions are online**
- ✅ **No under-replicated partitions detected**

## Consumer Group Health

- ✅ **5 consumer groups are active**
- ✅ **No consumer groups with significant lag detected**

## Overall Health Assessment

✅ **HEALTHY**: All systems are operating normally.
```

## kafka_under_replicated_partitions

Identifies and analyzes partitions with insufficient replication, where the in-sync replica (ISR) count is less than the configured replication factor. Provides detailed reporting on affected topics, missing replicas, potential causes, and step-by-step troubleshooting recommendations to restore data durability.

**Arguments:**
- None required - uses the configured cluster connection

**Example Response:**
```
# Under-Replicated Partitions Report

**Time**: 2023-08-15T12:34:56Z

⚠️ **Found 2 under-replicated partitions**

| Topic | Partition | Leader | Replica Count | ISR Count | Missing Replicas |
|:------|----------:|-------:|--------------:|----------:|:-----------------|
| orders | 3 | 1 | 3 | 2 | 3 |
| clickstream | 5 | 2 | 3 | 2 | 3 |

## Possible Causes

Under-replicated partitions occur when one or more replicas are not in sync with the leader. Common causes include:

- **Broker failure or network partition**
- **High load on brokers**
- **Insufficient disk space**
- **Network bandwidth limitations**
- **Misconfigured topic replication factor**

## Recommendations

1. **Check broker health** for any offline or struggling brokers
2. **Verify network connectivity** between brokers
3. **Monitor disk space** on broker nodes
4. **Review broker logs** for detailed error messages
5. **Consider increasing replication timeouts** if network is slow
```

## kafka_consumer_lag_report

Generates a comprehensive consumer lag analysis report covering all consumer groups in the cluster. Analyzes group states, member assignments, partition lag metrics, and provides performance optimization recommendations. Supports customizable lag thresholds for alerting and includes actionable insights for scaling decisions.

**Arguments:**
- `threshold` (optional): Message lag threshold for highlighting consumer groups with performance issues (default: 1000 messages). Groups exceeding this threshold will be flagged for attention.

**Example Response:**
```
# Kafka Consumer Lag Report

**Time**: 2023-08-15T12:34:56Z

**Lag Threshold**: 1000 messages

Found 3 consumer group(s)

## Consumer Group Summary

| Group ID | State | Members | Topics | Total Lag | High Lag |
|:---------|:------|--------:|-------:|----------:|:---------|
| order-processor | Stable | 2 | 1 | 15,420 | ⚠️ Yes |
| analytics-pipeline | Stable | 3 | 2 | 520 | No |
| monitoring | Stable | 1 | 3 | 0 | No |

## High Lag Details

### Group: order-processor

| Topic | Partition | Current Offset | Log End Offset | Lag |
|:------|----------:|--------------:|--------------:|----:|
| orders | 2 | 1,045,822 | 1,061,242 | 15,420 |

## Recommendations

1. **Check consumer instances** for errors or slowdowns
2. **Scale up consumer groups** with high lag
3. **Review consumer configuration** settings
4. **Examine processing bottlenecks** in consumer application logic
```