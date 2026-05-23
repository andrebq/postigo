package client

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/hashicorp/yamux"
)

var (
	validNodenameRE = regexp.MustCompile(`^[a-zA-Z0-9]+[a-zA-Z0-9-]*$`)
)

func TCPDialer(address string, maxTimeout time.Duration) func(context.Context) (io.ReadWriteCloser, error) {
	return func(c context.Context) (io.ReadWriteCloser, error) {
		dl, ok := c.Deadline()
		if !ok {
			dl = time.Now().Add(maxTimeout)
		} else {
			if time.Until(dl) > maxTimeout {
				dl = time.Now().Add(maxTimeout)
			}
		}
		finalTimeout := time.Until(dl)
		return net.DialTimeout("tcp", address, finalTimeout)
	}
}

func ExposeLocalPort(ctx context.Context,
	upstream string,
	nodename string,
	dial func(ctx context.Context) (io.ReadWriteCloser, error)) error {
	upstream = strings.TrimRight(upstream, "/")
	nodename = strings.TrimSpace(nodename)
	if !validNodenameRE.MatchString(nodename) {
		return fmt.Errorf("invalid nodename, should match: %v", validNodenameRE.String())
	}
	connurl := fmt.Sprintf("%v/%v", upstream, nodename)
	ws, res, err := websocket.Dial(ctx, connurl, &websocket.DialOptions{})
	if err != nil {
		return fmt.Errorf("unable to dial upstream server: %w", err)
	}
	if res.StatusCode >= 400 {
		return fmt.Errorf("unexpected status code: %v", res.StatusCode)
	}
	// for now, just loop over ws sending messages to the server
	conn := websocket.NetConn(ctx, ws, websocket.MessageBinary)
	defer conn.Close()
	session, err := yamux.Server(conn, yamux.DefaultConfig())
	if err != nil {
		return fmt.Errorf("unable to establish multiplexed connection: %w", err)
	}
	for {
		stream, err := session.AcceptStreamWithContext(ctx)
		if err != nil {
			// TODO: how to handle graceful shutdown?
			return fmt.Errorf("listener closed: %v", err)
		}
		slog.DebugContext(ctx, "Multiplexed stream acquired", "streamId", stream.StreamID())
		go handleStream(ctx, stream, dial)
	}
}

func handleStream(ctx context.Context, stream *yamux.Stream, dialer func(context.Context) (io.ReadWriteCloser, error)) error {
	rwc, err := dialer(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Error while dialing up local address", "error", err)
		return fmt.Errorf("unable to dial to local endpoint: %v", err)
	}
	errCh := make(chan error, 2)
	go bgCopy(stream, rwc, errCh)
	go bgCopy(rwc, stream, errCh)
	select {
	case <-ctx.Done():
		rwc.Close()
		stream.Close()
		slog.ErrorContext(ctx, "Stream copy interrupted by context", "error", ctx.Err())
		return ctx.Err()
	case err = <-errCh:
		rwc.Close()
		stream.Close()
		slog.ErrorContext(ctx, "Stream copy interrupted by copying error", "error", err)
		return err
	}
}

func bgCopy(out io.Writer, in io.Reader, errCh chan error) {
	var err error
	defer func() {
		if err == nil {
			err = io.EOF
		}
		errCh <- err
	}()
	_, err = io.Copy(out, in)
}
