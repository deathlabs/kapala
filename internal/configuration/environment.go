// Package configuration loads application settings from the environment.
package configuration

import (
	"errors"
	"os"

	"github.com/deathlabs/kapala/internal/embeddings"
	"github.com/joho/godotenv"
)

const defaultEmbeddingModel = "text-embedding-3-small"

// Embeddings contains the settings for the embeddings API.
type Embeddings struct {
	APIKey   string `mapstructure:"api_key"  json:"api_key"  yaml:"api_key"`
	Endpoint string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Model    string `mapstructure:"model"    json:"model"    yaml:"model"`
}

// LoadEnvironment loads a local .env file without replacing exported values.
func LoadEnvironment() error {
	return ignoreMissingEnvironment(godotenv.Load())
}

func ignoreMissingEnvironment(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return err
}

// EmbeddingSettings returns embeddings settings from environment variables.
func EmbeddingSettings() Embeddings {
	return Embeddings{
		APIKey:   os.Getenv("OPENAI_API_KEY"),
		Endpoint: os.Getenv("OPENAI_EMBEDDING_ENDPOINT"),
		Model:    embeddingModel(),
	}
}

func embeddingModel() string {
	return valueOrDefault(os.Getenv("OPENAI_EMBEDDING_MODEL"), defaultEmbeddingModel)
}

func valueOrDefault(value string, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}

// NewEmbeddingClient creates a client from embeddings settings.
func NewEmbeddingClient(settings Embeddings) (*embeddings.Client, error) {
	var (
		client  *embeddings.Client
		err     error
		options []embeddings.Option
	)

	options = []embeddings.Option{embeddings.WithModel(settings.Model)}
	if settings.Endpoint != "" {
		options = append(options, embeddings.WithEndpoint(settings.Endpoint))
	}

	client, err = embeddings.NewClient(settings.APIKey, options...)
	if err != nil {
		return nil, err
	}

	return client, nil
}
