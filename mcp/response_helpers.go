// filepath: /Users/tuannvm/Projects/cli/kafka-mcp-server/mcp/response_helpers.go
package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// formatResponseHeader creates a consistent header for response with title and timestamp
func formatResponseHeader(title string) string {
	return fmt.Sprintf("# %s\n\n**Time**: %s\n\n",
		title,
		time.Now().UTC().Format(time.RFC3339))
}

// formatErrorResponse creates a standard error response for prompts
func formatErrorResponse(description string, err error, message string) (*mcp.GetPromptResult, error) {
	errorText := fmt.Sprintf("%s: %v", message, err)

	return &mcp.GetPromptResult{
		Description: description,
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleAssistant,
				Content: mcp.TextContent{
					Type: "text",
					Text: errorText,
				},
			},
		},
	}, nil
}

// handlePromptError logs an error and returns a formatted error response
func handlePromptError(ctx context.Context, description string, err error, message string) (*mcp.GetPromptResult, error) {
	slog.ErrorContext(ctx, message, "error", err)
	return formatErrorResponse(description, err, message)
}

// createSuccessResponse creates a standard success response with formatted text
func createSuccessResponse(description string, content string) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: description,
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleAssistant,
				Content: mcp.TextContent{
					Type: "text",
					Text: content,
				},
			},
		},
	}, nil
}

// formatHealthStatus returns a health status emoji and text
func formatHealthStatus(critical bool, warning bool) (string, string) {
	if critical {
		return "🚨", "Critical Issues Detected"
	} else if warning {
		return "⚠️", "Warnings Detected"
	}
	return "✅", "Healthy"
}

// ----- Resource Helper Functions -----

// getTimestamp returns the current timestamp in ISO 8601 format
func getTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// getHealthStatusString returns a string indicating the overall health status
// Used by resources to maintain consistent status values
func getHealthStatusString(hasOfflinePartitions, noController, hasOfflineBrokers, hasWarnings bool) string {
	if hasOfflinePartitions || noController || hasOfflineBrokers {
		return "critical"
	}
	if hasWarnings {
		return "warning"
	}
	return "healthy"
}

// handleResourceError logs an error and returns a formatted error message with nil data
func handleResourceError(ctx context.Context, err error, message string) ([]byte, error) {
	slog.ErrorContext(ctx, message, "error", err)
	return nil, fmt.Errorf("%s: %w", message, err)
}

// createBaseResourceResponse creates a base response map with common fields
func createBaseResourceResponse() map[string]interface{} {
	return map[string]interface{}{
		"timestamp": getTimestamp(),
	}
}

// addRecommendations adds recommendations to a response based on the condition
func addRecommendations(response map[string]interface{}, condition bool, recommendations []string) {
	if condition {
		response["recommendations"] = recommendations
	}
}

// Helper functions

// Commented out as currently unused
// extractURIPathParameter extracts a path parameter from a URI
/*
func extractURIPathParameter(name string) func(uri string) string {
	return func(uri string) string {
		pattern := fmt.Sprintf("{%s}", name)
		parts := strings.Split(uri, "/")

		for i, part := range parts {
			if part == pattern && i+1 < len(parts) {
				return parts[i+1]
			}
		}

		return ""
	}
}
*/

// extractURIQueryParameter extracts a query parameter from a URI
func extractURIQueryParameter(uri, name string) string {
	if parsedURL, err := url.Parse(uri); err == nil {
		values := parsedURL.Query()
		return values.Get(name)
	}
	return ""
}

// getStatus returns the first value if condition is true, otherwise the second value
func getStatus(condition bool, trueValue, falseValue string) string {
	if condition {
		return trueValue
	}
	return falseValue
}

// getErrorString returns the error string or empty string if error is nil
func getErrorString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
