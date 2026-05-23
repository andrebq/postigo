package client

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/coder/websocket"
)

var (
	validNodenameRE = regexp.MustCompile(`^[a-zA-Z0-9]+[a-zA-Z0-9-]*$`)
)

func ExposeLocalPort(ctx context.Context, upstream string, nodename string) error {
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
	sc := bufio.NewScanner(conn)
	defer conn.Close()
	for {
		_, err := fmt.Fprintf(conn, "ping: %v\n", time.Now().Format(time.DateTime))
		if err != nil {
			return fmt.Errorf("error sending ping to server: %w", err)
		}
		if sc.Scan() {
			println("Got: ", sc.Text())
		} else {
			return fmt.Errorf("unable to get response from server: %w", err)
		}
		time.Sleep(time.Second)
	}
}
