// Package index stores embedded Markdown chunks in a JSON file.
package index

import (
	"sync"
	"time"
)

const (
	// Version identifies the current index schema.
	Version = 1
)

// Chunk contains searchable content and its embedding.
type Chunk struct {
	Content   string    `mapstructure:"content"   json:"content"   yaml:"content"`
	Embedding []float64 `mapstructure:"embedding" json:"embedding" yaml:"embedding"`
	ID        string    `mapstructure:"id"        json:"id"        yaml:"id"`
	Ordinal   int       `mapstructure:"ordinal"   json:"ordinal"   yaml:"ordinal"`
	Path      string    `mapstructure:"path"      json:"path"      yaml:"path"`
}

// Data is the durable semantic index schema.
type Data struct {
	Chunks    []Chunk   `mapstructure:"chunks"     json:"chunks"     yaml:"chunks"`
	Model     string    `mapstructure:"model"      json:"model"      yaml:"model"`
	UpdatedAt time.Time `mapstructure:"updated_at" json:"updated_at" yaml:"updated_at"`
	Version   int       `mapstructure:"version"    json:"version"    yaml:"version"`
}

// Store provides atomic updates to an index file.
type Store struct {
	data  Data
	model string
	mutex sync.RWMutex
	path  string
}
