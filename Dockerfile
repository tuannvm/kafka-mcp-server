# Use the official Golang image to create a build artifact.
# This is the builder stage.
FROM golang:1.24-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
# -ldflags="-w -s" reduces the size of the binary by removing debug information.
# CGO_ENABLED=0 disables CGO for static linking, useful for alpine base images.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /kafka-mcp-server ./cmd/

# --- Start final stage --- #

# Use a minimal base image like Alpine Linux
FROM alpine:latest

# Add ca-certificates in case TLS connections need system CAs
RUN apk --no-cache add ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /kafka-mcp-server .

# Expose ports if using HTTP transport in the future (optional for stdio)
# EXPOSE 8080

# Command to run the executable
# The server reads configuration from environment variables.
ENTRYPOINT ["/app/kafka-mcp-server"]
