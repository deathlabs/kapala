package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/deathlabs/kapala/internal"
	"github.com/deathlabs/kapala/internal/configuration"
	"github.com/spf13/cobra"
)

var (
	indexPath string
	notesDir  string
	port      string
	rootCmd   = &cobra.Command{
		Use:     "kapala",
		Short:   "A tool for sharing your brain with AI.",
		Version: version,
		RunE:    startCmd,
	}
	version string
)

func init() {
	rootCmd.Flags().StringVar(&indexPath, "index", "", "Path to the semantic index")
	rootCmd.Flags().StringVar(&notesDir, "notes-dir", "", "Directory of Markdown notes to monitor")
	rootCmd.Flags().StringVar(&port, "port", "8080", "Port")
}

func Execute() {
	var (
		ctx     context.Context
		err     error
		handler *slog.JSONHandler
		logger  *slog.Logger
		stop    context.CancelFunc
	)

	handler = slog.NewJSONHandler(os.Stderr, nil)
	logger = slog.New(handler)
	slog.SetDefault(logger)
	err = configuration.LoadEnvironment()
	if err != nil {
		logger.Error("failed to load environment", "error", err)
		os.Exit(1)
	}

	ctx, stop = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err = rootCmd.ExecuteContext(ctx)
	if err != nil {
		logger.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func startCmd(cmd *cobra.Command, _ []string) error {
	return internal.Run(cmd.Context(), port, notesDir, indexPath)
}
