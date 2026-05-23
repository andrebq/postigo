package main

import (
	"time"

	"github.com/andrebq/postigo/internal/client"
	"github.com/urfave/cli/v2"
)

func clientCmd() *cli.Command {
	upstream := "ws://localhost:9000/ws/expose"
	nodename := ""
	return &cli.Command{
		Name: "client",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "hub",
				Usage:       "Address of server where the websocket tunnel is running, should include the /ws/expose suffix when using default paths",
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
		Subcommands: []*cli.Command{
			exposeLocalCmd(&upstream, &nodename),
		},
	}
}

func exposeLocalCmd(upstream *string, nodename *string) *cli.Command {
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
			return client.ExposeLocalPort(ctx.Context, *upstream, *nodename, client.TCPDialer(localAddr, connTimeout))
		},
	}
}
