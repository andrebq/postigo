package client

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/andrebq/postigo/internal/auth"
	"github.com/andrebq/postigo/internal/ioutil"
	"github.com/coder/websocket"
	"github.com/hashicorp/yamux"
)

func ListenAndServeTCP(ctx context.Context, localAddr, upstream, nodename string, ks auth.KeySigner) error {
	lst, err := net.Listen("tcp", localAddr)
	if err != nil {
		return fmt.Errorf("unable to setup local listener: %w", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-ctx.Done()
		lst.Close()
	}()
	for {
		conn, err := lst.Accept()
		if err != nil {
			return fmt.Errorf("accept failed: %w", err)
		}
		// TODO: close conn before we exit from ListenAndServe?
		go func() {
			err := DialRemote(ctx, upstream, nodename, conn, ks)
			if err != nil {
				conn.Close()
				slog.ErrorContext(ctx, "error dialing remote", "error", err)
			}
		}()
	}
}

func DialRemote(ctx context.Context, upstream string, nodename string, rwc io.ReadWriteCloser, ks auth.KeySigner) error {
	defer rwc.Close()
	upstream = strings.TrimRight(upstream, "/")
	nodename = strings.TrimSpace(nodename)
	if !validNodenameRE.MatchString(nodename) {
		return fmt.Errorf("invalid nodename, should match: %v", validNodenameRE.String())
	}
	connurl := fmt.Sprintf("%v/dial/%v", upstream, nodename)
	tk, err := auth.DialNodeToken(ks, nodename, time.Minute)
	if err != nil {
		return fmt.Errorf("unable to sign token: %w", err)
	}
	ws, err := wsDial(ctx, connurl, tk)
	if err != nil {
		return err
	}
	// for now, just loop over ws sending messages to the server
	conn := websocket.NetConn(ctx, ws, websocket.MessageBinary)
	defer conn.Close()

	session, err := yamux.Client(conn, ioutil.MuxerConfig())
	if err != nil {
		return fmt.Errorf("unable to start yamux session: %w", err)
	}
	stream, err := session.OpenStream()
	if err != nil {
		return fmt.Errorf("unable to open stream: %w", err)
	}

	return handleStream(ctx, stream, rwc)
}

func wsDial(ctx context.Context, connurl string, token string) (*websocket.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	hdr := http.Header{}
	hdr.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	ws, res, err := websocket.Dial(ctx, connurl, &websocket.DialOptions{
		HTTPHeader: hdr,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to dial upstream server: %w", err)
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("unexpected status code: %v", res.StatusCode)
	}
	return ws, nil
}
