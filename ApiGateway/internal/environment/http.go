package environment

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

const ContextUserIDKey = "user_id"

type ServerOptions struct {
	logger *slog.Logger
}

func (opts *ServerOptions) WithLogger(logger *slog.Logger) {
	opts.logger = logger
}

func (opts *ServerOptions) NewServer(handler http.Handler, addr string) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func ListenAndServeContext(ctx context.Context, srv *http.Server) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.ListenAndServe()
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
