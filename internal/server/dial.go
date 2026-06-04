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

func (tm *trafficManager) handleDial(w http.ResponseWriter, req *http.Request) {
	nodename := req.PathValue("nodename")
	c, err := websocket.Accept(w, req, nil)
	if err != nil {
		slog.ErrorContext(req.Context(), "unable to accept connection", "error", err, "remoteAddress", req.RemoteAddr, "url", req.URL.Path)
		return
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
		slog.DebugContext(req.Context(), "Unable to offer yamux session", "error", err)
		return
	}
	defer session.Close()
	for {
		stream, err := session.AcceptStreamWithContext(ctx)
		if err != nil {
			slog.DebugContext(req.Context(), "Unable to accept stream", "error", err)
			return
		}
		go tm.dialNode(ctx, nodename, stream)
	}
}
