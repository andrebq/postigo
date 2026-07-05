package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/andrebq/postigo/internal/auth"
	"github.com/andrebq/postigo/internal/authz"
	"github.com/andrebq/postigo/internal/kdb"
)

func RegisterKey(ctx context.Context, db *kdb.DB, kid string, exposes []string, dials []string) error {
	keyCol, err := kdb.GetCollection[authz.AuthorizedKey](ctx, db, "node_keyset")
	if err != nil {
		return nil
	}
	// TODO: perform validation here
	for range 5 {
		var updated bool
		updated, err = keyCol.CAS(ctx, kid, func(old *authz.AuthorizedKey) (*authz.AuthorizedKey, error) {
			old.KID = kid
			old.Exposes = dedup(old.Exposes, exposes)
			old.DialTo = dedup(old.DialTo, dials)
			return old, nil
		})
		if updated {
			break
		}
	}
	return err
}

func dedup[T comparable, E ~[]T](a, b E) E {
	found := make(map[T]struct{}, max(len(a), len(b)))
	var ret E
	for _, item := range a {
		if _, exists := found[item]; exists {
			continue
		}
		found[item] = struct{}{}
		ret = append(ret, item)
	}
	for _, item := range b {
		if _, exists := found[item]; exists {
			continue
		}
		found[item] = struct{}{}
		ret = append(ret, item)
	}
	return ret
}

func Run(ctx context.Context,
	db *kdb.DB,
	tcpBind string) error {

	keyCol, err := kdb.GetCollection[authz.AuthorizedKey](ctx, db, "node_keyset")
	if err != nil {
		return nil
	}

	mux := http.NewServeMux()
	tm := newTrafficManager()
	go func() {
		err := tm.route(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Error while routing connections", "error", err)
		}
	}()
	mux.Handle("/ws/expose/{nodename}", authz.WrapFunc[auth.NodeClaims](tm.handleExpose, authz.KeyFromDB[auth.NodeClaims](keyCol), authz.AcceptAll))
	mux.Handle("/ws/dial/{nodename}", authz.WrapFunc[auth.NodeClaims](tm.handleDial, authz.KeyFromDB[auth.NodeClaims](keyCol), authz.AcceptAll))
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
