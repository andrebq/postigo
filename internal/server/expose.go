package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/andrebq/postigo/internal/ioutil"
	"github.com/coder/websocket"
	"github.com/hashicorp/yamux"
)

func (tm *trafficManager) handleExpose(w http.ResponseWriter, req *http.Request) {
	nodename := req.PathValue("nodename")
	c, err := websocket.Accept(w, req, nil)
	if err != nil {
		// ...
	}
	defer c.CloseNow()

	// Set the context as needed. Use of r.Context() is not recommended
	// to avoid surprising behavior (see http.Hijacker).
	ctx, cancel := context.WithCancelCause(context.Background())
	defer bindContext(req.Context(), cancel, errors.New("request complete"))
	conn := websocket.NetConn(ctx, c, websocket.MessageBinary)
	defer conn.Close()

	session, err := yamux.Server(conn, ioutil.MuxerConfig())
	if err != nil {
		slog.ErrorContext(req.Context(), "Unable to setup yamux session", "error", err)
		return
	}
	defer session.Close()
	regid, done := tm.registerNode(nodename, session)
	defer tm.unregisterNode(regid)
	select {
	case <-done:
		return
	case <-ctx.Done():
		return
	}
}

func bindContext(parent context.Context, cancel func(error), reason error) {
	<-parent.Done()
	if reason == nil {
		reason = parent.Err()
	}
	cancel(reason)
}
