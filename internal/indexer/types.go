// Package indexer converts changed Markdown files into indexed chunks.
package indexer

import (
	"github.com/deathlabs/kapala/internal/embeddings"
	"github.com/deathlabs/kapala/internal/index"
)

// Result describes an index update.
type Result struct {
	Chunks int    `mapstructure:"chunks" json:"chunks" yaml:"chunks"`
	Path   string `mapstructure:"path"   json:"path"   yaml:"path"`
}

// Service updates the semantic index for file events.
type Service struct {
	client *embeddings.Client
	root   string
	store  *index.Store
}
