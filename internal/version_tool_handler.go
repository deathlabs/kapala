package internal

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type versionToolInput struct{}

type versionToolOutput struct {
	Version string `mapstructure:"version" json:"version" yaml:"version"`
}

func versionToolHandler(_ context.Context, _ *mcp.CallToolRequest, _ versionToolInput) (*mcp.CallToolResult, versionToolOutput, error) {
	var output = versionToolOutput{Version: "hello world"}
	return nil, output, nil
}
