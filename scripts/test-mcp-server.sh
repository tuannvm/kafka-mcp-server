#!/bin/zsh

# Test script for Kafka MCP Server
SERVER_BIN="./bin/kafka-mcp-server"

echo "Testing Kafka MCP Server..."

# Step 1: Initialize the MCP connection
echo -e "\n1. Initializing MCP connection..."
INIT_RESULT=$(echo '{"jsonrpc":"2.0","id":"init1","method":"initialize","params":{"protocolVersion":"2024-11-05"}}' | $SERVER_BIN)
echo "$INIT_RESULT"

# Step 2: List available tools
echo -e "\n2. Listing available tools..."
TOOLS_RESULT=$(echo '{"jsonrpc":"2.0","id":"tools1","method":"tools/list","params":{}}' | $SERVER_BIN)
echo "$TOOLS_RESULT"

# Step 3: List all Kafka topics with partition counts and replication factors (newly added tool)
echo -e "\n3. Listing all Kafka topics..."
TOPICS_RESULT=$(echo '{"jsonrpc":"2.0","id":"call1","method":"tools/call","params":{"name":"list_topics"}}' | $SERVER_BIN)
echo "$TOPICS_RESULT"

# Step 4: Get cluster overview (available tool)
echo -e "\n4. Getting cluster overview..."
OVERVIEW_RESULT=$(echo '{"jsonrpc":"2.0","id":"call2","method":"tools/call","params":{"name":"cluster_overview"}}' | $SERVER_BIN)
echo "$OVERVIEW_RESULT"

# Step 5: Try to produce a test message (available tool)
echo -e "\n5. Trying to produce a test message..."
PRODUCE_RESULT=$(echo '{"jsonrpc":"2.0","id":"call3","method":"tools/call","params":{"name":"produce_message","arguments":{"topic":"test-topic","value":"Test message from MCP client"}}}' | $SERVER_BIN)
echo "$PRODUCE_RESULT"

echo -e "\nTest completed."
