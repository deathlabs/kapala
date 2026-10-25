package internal

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type VersionToolInput struct{}

type VersionToolOutput struct {
	Version string `mapstructure:"version" json:"version" yaml:"version"`
}

func VersionToolHandler(_ context.Context, _ *mcp.CallToolRequest, _ VersionToolInput) (*mcp.CallToolResult, VersionToolOutput, error) {
	var output = VersionToolOutput{Version: "hello world"}
	return nil, output, nil
}
