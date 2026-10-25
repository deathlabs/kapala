package internal

import (
	"log/slog"
	"net/http"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

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
		VersionToolHandler,
	)

	return server
}

func handler(logger *slog.Logger) http.Handler {
	var (
		handler       *mcp.StreamableHTTPHandler
		mux           *http.ServeMux
		serverOptions *mcp.StreamableHTTPOptions
	)

	serverOptions = &mcp.StreamableHTTPOptions{Logger: logger}
	handler = mcp.NewStreamableHTTPHandler(getServer, serverOptions)
	mux = http.NewServeMux()
	mux.Handle("/mcp", handler)

	return mux
}
