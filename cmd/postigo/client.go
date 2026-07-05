package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/andrebq/postigo/internal/auth"
	"github.com/andrebq/postigo/internal/client"
	"github.com/urfave/cli/v2"
)

func clientCmd() *cli.Command {
	upstream := "ws://localhost:9000"
	nodename := ""
	var ks auth.KeySigner
	dataFlag, opendb := dataDirFlag()
	return &cli.Command{
		Name: "client",
		Flags: []cli.Flag{
			dataFlag,
			&cli.StringFlag{
				Name:        "hub",
				Usage:       "Address of server where the websocket tunnel is running, should not include the /ws/ suffix when using default paths, but should include any prefix if using virtual hosts",
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
			_, secretDB, err := opendb()
			if err != nil {
				return err
			}
			ks, err = auth.LoadNodeKey(os.Getenv, os.Setenv)
			if err != nil && auth.IsKeyNotSet(err) {
				// try to load from database
				var err2 error
				ks, err2 = auth.LoadNodeKeyFromDB(ctx.Context, secretDB, true)
				if err2 != nil {
					slog.Info("Unable to load key from secrets database", "err", err)
					ks, err2 = auth.RandomNodeKey()
					if err2 != nil {
						return fmt.Errorf("missing env key %v and random key could not be generated %w", err, err2)
					}
				}
			} else if err != nil {
				return err
			}
			slog.Info("Node public key", "kid", ks.KID())
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
	connTimeout := time.Minute * 10
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
				Required:    false,
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
	var remoteNode string
	return &cli.Command{
		Name: "expose-remote",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "remote-node",
				Usage:       "Remote nodename that will be dialed when local connections are made",
				Destination: &remoteNode,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "tcp",
				Usage:       "Local address which we should listen for connections",
				Destination: &localAddr,
				Required:    true,
			},
		},
		Action: func(ctx *cli.Context) error {
			return client.ListenAndServeTCP(ctx.Context, localAddr, *upstream, remoteNode, *ks)
		},
	}
}
