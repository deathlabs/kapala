package internal

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
)

func Run(ctx context.Context, port string) error {
	var (
		address    string
		err        error
		errChannel chan error
		httpServer *http.Server
		logger     *slog.Logger
	)

	logger = slog.Default()

	address = ":" + port
	httpServer = &http.Server{
		Addr:    address,
		Handler: handler(logger),
	}

	errChannel = make(chan error, 1)

	logger.Info("starting HTTP server", "address", address)
	go func() {
		errChannel <- httpServer.ListenAndServe()
	}()

	select {
	case err = <-errChannel:
		if errors.Is(err, http.ErrServerClosed) {
			logger.Info("HTTP server stopped")
			return nil
		}

		logger.Error("HTTP server stopped", "error", err)
		return err
	case <-ctx.Done():
		logger.Info("stopping HTTP server")
		err = httpServer.Shutdown(context.Background())
		if err != nil {
			logger.Error("failed to stop HTTP server", "error", err)
			return err
		}

		err = <-errChannel
		if errors.Is(err, http.ErrServerClosed) {
			logger.Info("HTTP server stopped")
			return nil
		}

		logger.Error("HTTP server stopped", "error", err)
		return err
	}
}
