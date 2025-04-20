# Kafka MCP Server Execution Plan

This plan outlines the steps to build the Kafka MCP server based on the project README.

## Phase 1: Project Setup & Core Components

1. **Setup Project Skeleton:**  
     
   * Create the main project directory: `kafka-mcp-server`.  
   * Initialize Go module: `go mod init github.com/yourorg/kafka-mcp-server` (replace `yourorg` appropriately).  
   * Create subdirectories: `cmd/kafka-mcp-server/`, `config/`, `kafka/`, `mcp/`, `server/`, `middleware/`.  
   * Add initial `README.md` (already present) and `.gitignore`.

   

2. **Configuration Management (`config/`):**  
     
   * Define a `Config` struct in `config/config.go` to hold Kafka and MCP server settings.  
   * Implement a `LoadConfig()` function to read settings from environment variables (e.g., using `os.Getenv` or a library like `viper` or `godotenv`). Include defaults.  
   * Define necessary environment variables (e.g., `KAFKA_BROKERS`, `KAFKA_CLIENT_ID`, `MCP_TRANSPORT`).

   

3. **Kafka Client Wrapper (`kafka/`):**  
     
   * Add `franz-go` dependency: `go get github.com/twmb/franz-go`.  
   * Create `kafka/client.go`.  
   * Define a `Client` struct wrapping `*kgo.Client`.  
   * Implement `NewClient(cfg config.Config)` function to initialize the `franz-go` client using loaded configuration.  
   * Implement initial wrapper functions:  
     * `ProduceMessage(ctx context.Context, topic string, key, value []byte) error`  
     * `Close()` method for graceful shutdown.  
   * Add basic error handling and logging.

## Phase 2: MCP Integration

4. **MCP Server Setup (`server/`):**  
     
   * Add `mcp-go` dependency: `go get github.com/mark3labs/mcp-go`.  
   * Create `server/server.go`.  
   * Implement `NewMCPServer(name, version string)` function returning `*mcp.Server`.  
   * Implement `Start(ctx context.Context, s *mcp.Server, cfg config.Config)` function to handle starting the server based on `MCP_TRANSPORT` (initially support `stdio`).

   

5. **MCP Tools & Resources (`mcp/`):**  
     
   * Create `mcp/tools.go` and `mcp/resources.go`.  
   * Implement `RegisterTools(s *mcp.Server, kafkaClient *kafka.Client)` in `mcp/tools.go`.  
     * Define the `produce_message` tool using `mcp.NewTool`.  
     * Implement the handler function for `produce_message`, calling `kafkaClient.ProduceMessage`.  
   * Implement `RegisterResources(s *mcp.Server, kafkaClient *kafka.Client)` in `mcp/resources.go`.  
     * Define the `list_topics` resource (initially, might return a static list or require adding `ListTopics` to the Kafka client wrapper).  
     * Implement the handler for `list_topics`.  
   * Add necessary Kafka client wrapper methods (e.g., `ListTopics`) as needed.

   

6. **Main Entrypoint (`cmd/kafka-mcp-server/main.go`):**  
     
   * Create `cmd/kafka-mcp-server/main.go`.  
   * Implement the `main` function:  
     * Load configuration using `config.LoadConfig()`.  
     * Initialize the Kafka client using `kafka.NewClient()`. Handle errors. Defer `kafkaClient.Close()`.  
     * Initialize the MCP server using `server.NewMCPServer()`.  
     * Register tools and resources using `mcp.RegisterTools()` and `mcp.RegisterResources()`.  
     * Set up graceful shutdown using signals (SIGINT, SIGTERM) and context cancellation.  
     * Start the MCP server using `server.Start()`. Handle errors.

## Phase 3: Enhancements & Production Readiness

7. **Testing:**  
     
   * Write unit tests for `config` loading.  
   * Write unit tests for `kafka` client wrapper functions (consider using mocks or an embedded Kafka cluster like `testcontainers-go`).  
   * Write integration tests for MCP tool handlers.

   

8. **Advanced Features & Refinements:**  
     
   * Implement `ConsumeMessages` tool in `mcp/tools.go` and the corresponding `ConsumeMessages` method in `kafka/client.go`. Consider streaming strategies.  
   * Implement `list_topics` resource properly by adding `ListTopics` to `kafka/client.go`.  
   * Add support for SASL/SSL in `config/` and `kafka/client.go`.  
   * Implement optional middleware (`middleware/`) for logging, metrics, or error handling.  
   * Add support for HTTP transport in `server/server.go`.  
   * Add more admin tools/resources as needed.

   

9. **Documentation & Examples:**  
     
   * Update `README.md` with detailed usage instructions, environment variables, and tool/resource contracts.  
   * Create an `examples/` directory with sample client interactions or scripts.

   

10. **CI/CD & Docker:**  
      
    * Create a `Dockerfile` for building a container image.  
    * Set up a basic CI pipeline (e.g., GitHub Actions) to build, lint, and test the code on push/PR.  
    * Consider adding GoReleaser for automated releases.

This plan provides a structured approach to developing the Kafka MCP server. Each phase builds upon the previous one, starting with the core setup and gradually adding features and robustness.  
