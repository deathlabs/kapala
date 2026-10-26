package internal

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/deathlabs/kapala/internal/configuration"
	"github.com/deathlabs/kapala/internal/embeddings"
	"github.com/deathlabs/kapala/internal/index"
	"github.com/deathlabs/kapala/internal/indexer"
	"github.com/deathlabs/kapala/internal/watcher"
)

// Run serves MCP requests and optionally monitors a directory of Markdown notes.
func Run(ctx context.Context, port string, notesDir string, indexPath string) error {
	var (
		address    string
		cancel     context.CancelFunc
		client     *embeddings.Client
		err        error
		errChannel chan error
		httpServer *http.Server
		indexStore *index.Store
		indexing   *indexer.Service
		logger     *slog.Logger
		retrieval  *retrievalTools
		runContext context.Context
		settings   configuration.Embeddings
		watchError chan error
	)

	logger = slog.Default()
	retrieval = &retrievalTools{}

	address = ":" + port
	httpServer = &http.Server{
		Addr:    address,
		Handler: handler(logger, retrieval),
	}

	runContext, cancel = context.WithCancel(ctx)
	defer cancel()

	if notesDir != "" {
		settings = configuration.EmbeddingSettings()
		client, err = configuration.NewEmbeddingClient(settings)
		if err != nil {
			return err
		}
		if indexPath == "" {
			indexPath = filepath.Join(notesDir, "index.json")
		}

		indexStore, err = index.NewStore(indexPath, settings.Model)
		if err != nil {
			return err
		}
		indexing, err = indexer.New(notesDir, client, indexStore)
		if err != nil {
			return err
		}
		retrieval.client = client
		retrieval.notesRoot = notesDir
		retrieval.store = indexStore
	}

	errChannel = make(chan error, 1)
	logger.Info("starting HTTP server", "address", address)
	go func() {
		errChannel <- httpServer.ListenAndServe()
	}()

	if notesDir != "" {
		watchError = make(chan error, 1)
		go func() {
			watchError <- watcher.Watch(runContext, notesDir, func(event watcher.Event) error {
				var (
					handleError error
					result      indexer.Result
				)

				result, handleError = indexing.Handle(runContext, event)
				if handleError != nil {
					return handleError
				}

				logger.InfoContext(
					runContext,
					"semantic index updated",
					"chunks", result.Chunks,
					"operation", event.Operation,
					"path", result.Path,
				)

				return nil
			})
		}()
	}

	select {
	case err = <-errChannel:
		if errors.Is(err, http.ErrServerClosed) {
			logger.Info("HTTP server stopped")
			return nil
		}

		logger.Error("HTTP server stopped", "error", err)
		return err
	case err = <-watchError:
		cancel()
		if err != nil {
			logger.Error("file watcher stopped", "error", err)
		}

		return stopHTTPServer(httpServer, errChannel, err, logger)
	case <-ctx.Done():
		cancel()
		logger.Info("stopping HTTP server")

		return stopHTTPServer(httpServer, errChannel, nil, logger)
	}
}

func stopHTTPServer(httpServer *http.Server, errChannel chan error, result error, logger *slog.Logger) error {
	var (
		err error
	)

	err = httpServer.Shutdown(context.Background())
	if err != nil {
		logger.Error("failed to stop HTTP server", "error", err)
		return errors.Join(result, err)
	}

	err = <-errChannel
	if !errors.Is(err, http.ErrServerClosed) {
		logger.Error("HTTP server stopped", "error", err)
		return errors.Join(result, err)
	}

	logger.Info("HTTP server stopped")

	return result
}
