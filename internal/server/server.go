package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"
)

func Run(ctx context.Context, tcpBind string) error {
	mux := http.NewServeMux()
	tm := newTrafficManager()
	go func() {
		err := tm.route(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Error while routing connections", "error", err)
		}
	}()
	mux.HandleFunc("/ws/expose/{nodename}", tm.handleExpose)
	mux.HandleFunc("/ws/dial/{nodename}", tm.handleDial)
	h := http.Server{
		Addr:    tcpBind,
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}
	errCh := make(chan error, 1)
	go func() {
		slog.InfoContext(ctx, "Starting postigo server", "addr", h.Addr)
		errCh <- h.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		slog.InfoContext(ctx, "Shutdown started")
		sctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		h.Shutdown(sctx)
		return ctx.Err()
	case err := <-errCh:
		slog.ErrorContext(ctx, "Unable to start server perform clear shutdown of server", "error", err)
		return err
	}
}
