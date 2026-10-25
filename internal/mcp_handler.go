package internal

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// healthHandler responds to health checks with a JSON status.
func healthHandler(writer http.ResponseWriter, _ *http.Request) {
	var (
		encoder  *json.Encoder
		response map[string]string
	)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	encoder = json.NewEncoder(writer)
	response = map[string]string{"status": "ok"}
	_ = encoder.Encode(response)
}

// getServer returns an MCP server to handle MCP requests.
func getServer(_ *http.Request) *mcp.Server {
	var (
		logger *slog.Logger
		server *mcp.Server
	)

	logger = slog.Default()

	// Create the MCP server and configure it to send logs to the logger.
	server = mcp.NewServer(
		&mcp.Implementation{Name: "kapala"},
		&mcp.ServerOptions{Logger: logger},
	)

	// Log every tool call the MCP server receives.
	server.AddReceivingMiddleware(ToolCallLoggingMiddleware(logger))

	// Register tools with the MCP server.
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "version",
			Description: "Get the build version.",
		},
		versionToolHandler,
	)

	return server
}

// handler returns an HTTP handler that routes and logs requests.
func handler(logger *slog.Logger) http.Handler {
	var (
		loggingHandler http.HandlerFunc
		mcpHandler     *mcp.StreamableHTTPHandler
		mux            *http.ServeMux
		serverOptions  *mcp.StreamableHTTPOptions
	)

	// Create an MCP handler.
	serverOptions = &mcp.StreamableHTTPOptions{Logger: logger}
	mcpHandler = mcp.NewStreamableHTTPHandler(getServer, serverOptions)

	// Create an HTTP router.
	mux = http.NewServeMux()

	// Attach the healthcheck and MCP handlers to the HTTP router.
	mux.HandleFunc("GET /health", healthHandler)
	mux.Handle("/mcp", mcpHandler)

	// Wrap the HTTP router with a logger that logs each HTTP request.
	loggingHandler = func(writer http.ResponseWriter, request *http.Request) {
		logger.InfoContext(
			request.Context(),
			"HTTP request",
			"method", request.Method,
			"path", request.URL.Path,
		)

		mux.ServeHTTP(writer, request)
	}

	return loggingHandler
}
