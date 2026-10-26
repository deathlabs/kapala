package cmd

import (
	"encoding/json"

	"github.com/deathlabs/kapala/internal/configuration"
	"github.com/deathlabs/kapala/internal/embeddings"
	"github.com/spf13/cobra"
)

type embedOutput struct {
	Dimensions int    `mapstructure:"dimensions" json:"dimensions" yaml:"dimensions"`
	Model      string `mapstructure:"model"      json:"model"      yaml:"model"`
}

var (
	embedCmd = &cobra.Command{
		Use:   "embed <text>",
		Short: "Create an embedding for text.",
		Args:  cobra.ExactArgs(1),
		RunE:  embedText,
	}
)

func init() {
	rootCmd.AddCommand(embedCmd)
}

func embedText(cmd *cobra.Command, args []string) error {
	var (
		client     *embeddings.Client
		embeddings [][]float64
		encoder    *json.Encoder
		err        error
		output     embedOutput
		settings   configuration.Embeddings
	)

	settings = configuration.EmbeddingSettings()
	client, err = configuration.NewEmbeddingClient(settings)
	if err != nil {
		return err
	}

	embeddings, err = client.Embed(cmd.Context(), args)
	if err != nil {
		return err
	}

	output = embedOutput{Dimensions: len(embeddings[0]), Model: client.Model()}
	encoder = json.NewEncoder(cmd.OutOrStdout())
	err = encoder.Encode(output)
	if err != nil {
		return err
	}

	return nil
}
