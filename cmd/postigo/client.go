package main

import (
	"fmt"
	"os"
	"time"

	"github.com/andrebq/postigo/internal/auth"
	"github.com/andrebq/postigo/internal/client"
	"github.com/urfave/cli/v2"
)

func clientCmd() *cli.Command {
	upstream := "ws://localhost:9000/ws"
	nodename := ""
	var ks auth.KeySigner
	return &cli.Command{
		Name: "client",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "hub",
				Usage:       "Address of server where the websocket tunnel is running, should include the /ws suffix when using default paths",
				Destination: &upstream,
				Value:       upstream,
			},
			&cli.StringFlag{
				Name:        "nodename",
				Usage:       "Name of the node being used",
				Destination: &nodename,
				Value:       nodename,
				Required:    true,
			},
		},
		Before: func(ctx *cli.Context) error {
			var err error
			ks, err = auth.LoadNodeKey(os.Getenv, os.Setenv)
			if err != nil && auth.IsKeyNotSet(err) {
				var rndErr error
				ks, rndErr = auth.RandomNodeKey()
				if rndErr != nil {
					return fmt.Errorf("missing env key %v and random key could not be generated %w", err, rndErr)
				}
			} else if err != nil {
				return err
			}
			return nil
		},
		Subcommands: []*cli.Command{
			exposeLocalCmd(&upstream, &nodename, &ks),
			exposeRemoteCmd(&upstream, &nodename, &ks),
		},
	}
}

func exposeLocalCmd(upstream *string,
	nodename *string,
	ks *auth.KeySigner) *cli.Command {
	var localAddr string
	var connTimeout time.Duration
	return &cli.Command{
		Name: "expose-local",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "tcp",
				Usage:       "Address being exposed, must be TCP but can be other nodes in the network",
				Destination: &localAddr,
				Required:    true,
			},
			&cli.DurationFlag{
				Name:        "conn-timeout",
				Usage:       "Max timeout when connecting to local host",
				Destination: &connTimeout,
				Value:       connTimeout,
				Required:    true,
			},
		},
		Action: func(ctx *cli.Context) error {
			return client.ExposeLocalPort(ctx.Context, *upstream, *nodename, *ks, client.TCPDialer(localAddr, connTimeout))
		},
	}
}

func exposeRemoteCmd(upstream *string,
	nodename *string,
	ks *auth.KeySigner) *cli.Command {
	var localAddr string
	return &cli.Command{
		Name: "expose-remote",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "tcp",
				Usage:       "Local address which we should listen for connections",
				Destination: &localAddr,
				Required:    true,
			},
		},
		Action: func(ctx *cli.Context) error {
			return client.ListenAndServeTCP(ctx.Context, localAddr, *upstream, *nodename, *ks)
		},
	}
}
