package main

import (
	"fmt"
	"net"

	"github.com/andrebq/postigo/internal/server"
	"github.com/urfave/cli/v2"
)

func serverCmd() *cli.Command {
	bindAddr := "127.0.0.1"
	bindPort := uint(9000)
	return &cli.Command{
		Name: "serve",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "bind",
				Usage:       "IP address to bind for incoming connections",
				EnvVars:     []string{"POSTIGO_SERVER_BIND_ADDR"},
				Destination: &bindAddr,
				Value:       bindAddr,
			},
			&cli.UintFlag{
				Name:        "port",
				Usage:       "Port to bind for incoming connections",
				EnvVars:     []string{"POSTIGO_SERVER_BIND_PORT"},
				Destination: &bindPort,
				Value:       bindPort,
			},
		},
		Action: func(ctx *cli.Context) error {
			ipAndPort := net.JoinHostPort(bindAddr, fmt.Sprintf("%d", bindPort))
			return server.Run(ctx.Context, ipAndPort)
		},
	}
}
