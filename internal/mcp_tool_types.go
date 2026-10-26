package internal

import (
	"github.com/deathlabs/kapala/internal/embeddings"
	"github.com/deathlabs/kapala/internal/index"
)

type retrievalTools struct {
	client    *embeddings.Client
	notesRoot string
	store     *index.Store
}

type semanticSearchInput struct {
	Query string `mapstructure:"query" json:"query" jsonschema:"A natural-language query to match against indexed note chunks." yaml:"query"`
	TopK  int    `mapstructure:"top_k" json:"top_k,omitempty" jsonschema:"Maximum results to return. Defaults to 5 and cannot exceed 50." yaml:"top_k,omitempty"`
}

type semanticSearchOutput struct {
	Results []semanticSearchResult `mapstructure:"results" json:"results" yaml:"results"`
}

type semanticSearchResult struct {
	Content string  `mapstructure:"content" json:"content" yaml:"content"`
	Ordinal int     `mapstructure:"ordinal" json:"ordinal" yaml:"ordinal"`
	Path    string  `mapstructure:"path"    json:"path"    yaml:"path"`
	Score   float64 `mapstructure:"score"   json:"score"   yaml:"score"`
}

type scoredChunk struct {
	chunk index.Chunk
	score float64
}

type listIndexedFilesInput struct{}

type listIndexedFilesOutput struct {
	Files []string `mapstructure:"files" json:"files" yaml:"files"`
}

type getFileContentInput struct {
	Path string `mapstructure:"path" json:"path" jsonschema:"An exact normalized relative Markdown path returned by list_indexed_files." yaml:"path"`
}

type getFileContentOutput struct {
	Content string `mapstructure:"content" json:"content" yaml:"content"`
	Path    string `mapstructure:"path"    json:"path"    yaml:"path"`
}
