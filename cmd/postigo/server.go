package main

import (
	"fmt"
	"net"
	"path/filepath"

	"github.com/andrebq/postigo/internal/kdb"
	"github.com/andrebq/postigo/internal/server"
	"github.com/urfave/cli/v2"
)

func serverCmd() *cli.Command {
	bindAddr := "127.0.0.1"
	bindPort := uint(9000)
	var dataDir string
	return &cli.Command{
		Name: "serve",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "data-dir",
				Usage:       "Directory where the server database is located",
				Required:    true,
				EnvVars:     []string{"POSTIGO_SERVER_DATA_DIR"},
				Destination: &dataDir,
			},
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
		Before: func(ctx *cli.Context) error {
			var err error
			dataDir, err = filepath.Abs(dataDir)
			return err
		},
		Action: func(ctx *cli.Context) error {
			db, err := kdb.Open(dataDir)
			if err != nil {
				return err
			}
			ipAndPort := net.JoinHostPort(bindAddr, fmt.Sprintf("%d", bindPort))
			return server.Run(ctx.Context, db, ipAndPort)
		},
	}
}
