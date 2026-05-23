package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/coder/websocket"
)

func handleExpose(w http.ResponseWriter, req *http.Request) {
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

	sc := bufio.NewScanner(conn)
	for sc.Scan() {
		println(fmt.Sprintf("<from %v<: %v", req.RemoteAddr, strings.TrimSpace(sc.Text())))
		fmt.Fprintf(conn, "OK\n")
	}

	c.Close(websocket.StatusNormalClosure, "")
}

func bindContext(parent context.Context, cancel func(error), reason error) {
	select {
	case <-parent.Done():
		if reason == nil {
			reason = parent.Err()
		}
		cancel(reason)
	}
}
