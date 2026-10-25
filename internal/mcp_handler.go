package internal

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func healthHandler(writer http.ResponseWriter, _ *http.Request) {
	var (
		encoder  *json.Encoder
		response map[string]string
	)

	writer.WriteHeader(http.StatusOK)
	encoder = json.NewEncoder(writer)
	response = map[string]string{"status": "ok"}
	_ = encoder.Encode(response)
}

func getServer(_ *http.Request) *mcp.Server {
	var (
		logger *slog.Logger
		server *mcp.Server
	)

	logger = slog.Default()

	server = mcp.NewServer(
		&mcp.Implementation{Name: "kapala"},
		&mcp.ServerOptions{Logger: logger},
	)

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

func handler(logger *slog.Logger) http.Handler {
	var (
		mcpHandler    *mcp.StreamableHTTPHandler
		mux           *http.ServeMux
		serverOptions *mcp.StreamableHTTPOptions
	)

	serverOptions = &mcp.StreamableHTTPOptions{Logger: logger}
	mcpHandler = mcp.NewStreamableHTTPHandler(getServer, serverOptions)
	mux = http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.Handle("/mcp", mcpHandler)

	return mux
}
