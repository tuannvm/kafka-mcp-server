# CLAUDE.md — Kafka MCP Server (Personal Fork)

This file is the source of truth for Claude Code when working in this repository.
It describes the project goals, architecture, development workflow, and contribution strategy
relative to the upstream repo at `tuannvm/kafka-mcp-server`.

---

## Project Vision

The goal of this fork is to deepen understanding of the Model Context Protocol (MCP) by building
on top of an existing, real-world Kafka MCP server implementation. The upstream repo is solid but
relatively minimal — this fork aims to extend it with enterprise-grade features, better observability,
and richer agentic workflows that make it genuinely useful to companies adopting AI tooling.

The broader thesis: MCP is the emerging standard for how LLMs talk to infrastructure. Kafka is one
of the most widely deployed pieces of data infrastructure in the enterprise. Bridging the two in a
production-quality way is a meaningful contribution to the ecosystem and a valuable skill to develop.

---

## Upstream Reference

- **Repo**: https://github.com/tuannvm/kafka-mcp-server
- **Language**: Go
- **Key libraries**: `franz-go` (Kafka client), `mcp-go` (MCP server framework)
- **Transport**: stdio (default) and HTTP with OAuth 2.1
- **What upstream already does well**:
  - Core Kafka tools: `produce_message`, `consume_messages`, `list_topics`, `describe_topic`,
    `list_brokers`, `list_consumer_groups`, `describe_consumer_group`, `describe_configs`
  - MCP Resources: cluster overview, health check, under-replicated partitions, consumer lag report
  - MCP Prompts: pre-configured workflows for diagnostics
  - SASL auth (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512) and TLS
  - OAuth 2.1 for HTTP transport

---

## What This Fork Adds (Planned Contributions)

These are the gaps identified in upstream that represent real learning opportunities and
real value for companies using Kafka with LLM agents.

### Phase 1 — Understand the Codebase
- [ ] Clone and run the server locally against a Docker Kafka cluster
- [ ] Exercise every tool via Claude Desktop or Claude Code
- [ ] Read `kafka/interface.go`, `mcp/tools.go`, `mcp/resources.go`, `mcp/prompts.go`
- [ ] Write notes on how a new tool gets registered end-to-end

### Phase 2 — Missing Tools (PRs to upstream or fork additions)
- [ ] `alter_topic_config` — let an agent change retention, compaction settings on a topic
- [ ] `reset_consumer_group_offsets` — critical for incident recovery workflows
- [ ] `get_partition_leaders` — useful for diagnosing uneven load across brokers
- [ ] `search_messages` — consume from a topic and filter by key/value pattern (agents love this)
- [ ] `create_topic` / `delete_topic` — careful, but useful for dev/test environments

### Phase 3 — Observability & Agentic Workflows
- [ ] Add structured logging (JSON) that agents can parse in their reasoning
- [ ] Add a `kafka_incident_runbook` prompt: a guided agent workflow for diagnosing consumer lag
  spikes end-to-end (check lag → check broker health → check partition leaders → recommend action)
- [ ] Add a `kafka_schema_inspector` tool if the cluster uses Schema Registry (Confluent)
- [ ] Explore streaming responses for `consume_messages` (currently batch-only)

### Phase 4 — Contribute Back
- [ ] Open issues on upstream for gaps discovered
- [ ] Submit PRs for tools that upstream is missing
- [ ] Write a blog post or README section explaining the project from an MCP learning perspective

---

## Development Setup

### Prerequisites
- Go 1.24+
- Docker (for running a local Kafka cluster)
- Claude Desktop or Claude Code (for testing MCP tools interactively)

### Running a Local Kafka Cluster
```bash
docker run -d \
  --name kafka \
  -p 9092:9092 \
  -e KAFKA_CFG_NODE_ID=1 \
  -e KAFKA_CFG_PROCESS_ROLES=broker,controller \
  -e KAFKA_CFG_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093 \
  -e KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
  -e KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=1@localhost:9093 \
  -e KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER \
  bitnami/kafka:latest
```

### Building and Running
```bash
# Clone and build
git clone https://github.com/YOUR_USERNAME/kafka-mcp-server.git
cd kafka-mcp-server
make build

# Run against local Kafka
KAFKA_BROKERS=localhost:9092 MCP_TRANSPORT=stdio ./bin/kafka-mcp-server

# Run from source (faster iteration)
make run-dev
```

### Testing
```bash
make test                  # All tests (requires Docker)
make test-no-kafka         # Unit tests only, no Kafka needed
SKIP_KAFKA_TESTS=true go test ./...  # Same as above
```

### Linting
```bash
make lint
```

---

## Project Architecture

```
kafka-mcp-server/
├── cmd/server/main.go      # Entry point, graceful shutdown
├── config/                 # Env-based config (KAFKA_BROKERS, SASL, TLS, etc.)
├── kafka/
│   ├── interface.go        # KafkaClient interface — ADD NEW METHODS HERE FIRST
│   ├── client.go           # franz-go implementation
│   └── *_test.go           # Integration tests using testcontainers
└── mcp/
    ├── tools.go            # Tool handlers — register new tools here
    ├── resources.go        # Resource handlers (cluster health, lag reports)
    └── prompts.go          # Pre-configured prompt workflows
```

### How to Add a New Tool (Step-by-Step)

1. **Define the method on the interface** (`kafka/interface.go`)
   ```go
   ResetConsumerGroupOffsets(ctx context.Context, group string, topic string, partition int32, offset int64) error
   ```

2. **Implement it on the client** (`kafka/client.go`)
   Use `franz-go` primitives. Check the franz-go docs for the right admin request type.

3. **Register the MCP tool** (`mcp/tools.go`)
   ```go
   s.AddTool(mcp.NewTool("reset_consumer_group_offsets",
       mcp.WithDescription("Reset consumer group offsets for a topic partition"),
       mcp.WithString("group_id", mcp.Required(), mcp.Description("Consumer group ID")),
       mcp.WithString("topic", mcp.Required(), mcp.Description("Topic name")),
       mcp.WithNumber("partition", mcp.Required(), mcp.Description("Partition number")),
       mcp.WithNumber("offset", mcp.Required(), mcp.Description("Target offset (-1 for latest, -2 for earliest)")),
   ), handleResetOffsets(kafkaClient))
   ```

4. **Write a handler function** in `mcp/tools.go`
   Return structured JSON so agents can reason about the result clearly.

5. **Write tests** — at minimum a unit test with a mock KafkaClient, ideally an integration test.

6. **Document it** in `docs/tools.md`.

---

## MCP Concepts to Internalize

Understanding these will make you effective at both building and explaining MCP to companies:

- **Tools** = actions the LLM can invoke (produce, consume, describe). These are the most important.
- **Resources** = data the LLM can read without triggering side effects (health reports, overviews).
  Useful for giving an agent context before it acts.
- **Prompts** = templated workflows that chain tools together. Think of them as runbooks for agents.
- **Transport** = how the MCP client talks to the server. `stdio` is for local (Claude Desktop, Cursor).
  `HTTP` is for remote/multi-tenant deployments. Knowing both matters for enterprise use cases.

---

## Key Learning Goals

Beyond shipping code, this project exists to build intuition for:

1. **MCP tool schema design** — how do you write tool descriptions and parameter schemas that LLMs
   understand and use correctly? Bad schemas lead to confused agents.

2. **Agentic workflow patterns** — when should an agent use a Resource vs call a Tool? How do you
   chain tools to accomplish multi-step Kafka diagnostics?

3. **Enterprise integration concerns** — auth (SASL, OAuth), TLS, network config, multi-cluster
   support. These are what real companies ask about.

4. **Go proficiency** — franz-go is idiomatic, high-performance Go. Reading and contributing to this
   codebase is good Go practice beyond typical web API work.

---

## Issues to Open on Upstream

As gaps are discovered, track them here before filing:

| Feature | Priority | Status |
|---|---|---|
| `reset_consumer_group_offsets` tool | High | Not started |
| `alter_topic_config` tool | Medium | Not started |
| `search_messages` (filter consume) | Medium | Not started |
| Structured JSON logging | Low | Not started |
| Schema Registry integration | Low | Not started |

---

## Resources

- [MCP Specification](https://spec.modelcontextprotocol.io)
- [mcp-go library](https://github.com/mark3labs/mcp-go)
- [franz-go docs](https://pkg.go.dev/github.com/twmb/franz-go)
- [Upstream repo](https://github.com/tuannvm/kafka-mcp-server)
- [Anthropic MCP docs](https://docs.anthropic.com/en/docs/build-with-claude/computer-use)