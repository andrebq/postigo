package main

import (
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
	return &cli.Command{
		Name: "expose-local",
		Action: func(ctx *cli.Context) error {
			return client.ExposeLocalPort(ctx.Context, *upstream, *nodename)
		},
	}
}
