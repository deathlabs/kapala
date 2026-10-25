package internal

import (
	"context"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolCallLoggingMiddleware returns middleware that logs every tool call.
func ToolCallLoggingMiddleware(logger *slog.Logger) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {

		return func(ctx context.Context, method string, request mcp.Request) (result mcp.Result, err error) {
			var (
				callToolRequest *mcp.CallToolRequest
				finished        bool // Only True if next() returns normally.
				isToolCall      bool
				session         mcp.Session
				sessionID       string
				startedAt       time.Time
				toolName        string
			)

			// Use a type assertion to filter MCP requests for tool calls.
			callToolRequest, isToolCall = request.(*mcp.CallToolRequest)
			if !isToolCall {
				return next(ctx, method, request)
			}

			// Check if Params is nil before reading the tool name.
			if callToolRequest.Params != nil {
				toolName = callToolRequest.Params.Name
			}

			// Get the session ID to aggregate logs.
			session = callToolRequest.GetSession()
			if session != nil {
				sessionID = session.ID()
			}

			logger.InfoContext(
				ctx,
				"started tool call",
				"tool", toolName,
				"session", sessionID,
			)
			startedAt = time.Now()

			// Defer logging the outcome so every exit is only logged once.
			defer func() {
				var (
					attrs        []any
					isToolResult bool
					toolResult   *mcp.CallToolResult
				)

				attrs = []any{
					"tool", toolName,
					"session", sessionID,
					"duration_ms", time.Since(startedAt).Milliseconds(),
				}

				// When the handler panicked ("finished" is still false):
				if !finished {
					logger.ErrorContext(
						ctx,
						"tool call did not finish",
						attrs...)
					return
				}

				// When the handler returned a protocol-level error:
				if err != nil {
					logger.ErrorContext(
						ctx,
						"failed to call tool",
						append(attrs, "error", err)...)
					return
				}

				// When the tool reported a failure through IsError:
				toolResult, isToolResult = result.(*mcp.CallToolResult)
				if isToolResult && toolResult.IsError {
					logger.WarnContext(
						ctx,
						"tool returned error result",
						attrs...)
					return
				}

				// When the tool call succeeded:
				logger.InfoContext(
					ctx,
					"completed tool call",
					attrs...)
			}()

			// Run the next handler, then mark the call as finished.
			result, err = next(ctx, method, request)
			finished = true

			return result, err
		}
	}
}
