package cmd

import (
	"log/slog"
	"os"

	"github.com/deathlabs/kapala/internal"
	"github.com/spf13/cobra"
)

var (
	port    string
	rootCmd = &cobra.Command{
		Use:     "kapala",
		Short:   "A tool for sharing your brain with AI.",
		Version: version,
		RunE:    startCmd,
	}
	version string
)

func init() {
	rootCmd.Flags().StringVar(&port, "port", "8080", "Port")
}

func Execute() {
	var (
		err     error
		handler *slog.JSONHandler
		logger  *slog.Logger
	)

	handler = slog.NewJSONHandler(os.Stderr, nil)
	logger = slog.New(handler)
	slog.SetDefault(logger)

	err = rootCmd.Execute()
	if err != nil {
		logger.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func startCmd(cmd *cobra.Command, _ []string) error {
	return internal.Run(cmd.Context(), port)
}
