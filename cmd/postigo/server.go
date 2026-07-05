package main

import (
	"fmt"
	"net"
	"path/filepath"
	"sync"

	"github.com/andrebq/postigo/internal/kdb"
	"github.com/andrebq/postigo/internal/server"
	"github.com/urfave/cli/v2"
)

func dataDirFlag() (flag *cli.StringFlag, dbname func() (*kdb.DB, *kdb.DB, error)) {
	var output string
	var dataDB *kdb.DB
	var secretDB *kdb.DB
	var openErr error
	open := sync.OnceFunc(func() {
		output, openErr = filepath.Abs(output)
		if openErr != nil {
			return
		}
		dataDB, openErr = kdb.Open(filepath.Join(output, "data.db"))
		if openErr != nil {
			return
		}
		secretDB, openErr = kdb.Open(filepath.Join(output, "secrets.db"))
	})
	return &cli.StringFlag{
			Name:        "data-dir",
			Usage:       "Directory where the server database is located",
			Required:    true,
			EnvVars:     []string{"POSTIGO_DATA_DIR"},
			Destination: &output,
		}, func() (*kdb.DB, *kdb.DB, error) {
			open()
			return dataDB, secretDB, openErr
		}
}

func serverCmd() *cli.Command {
	bindAddr := "127.0.0.1"
	bindPort := uint(9000)
	dataFlag, openDB := dataDirFlag()
	return &cli.Command{
		Name: "serve",
		Flags: []cli.Flag{
			dataFlag,
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
			db, _, err := openDB()
			if err != nil {
				return err
			}
			ipAndPort := net.JoinHostPort(bindAddr, fmt.Sprintf("%d", bindPort))
			return server.Run(ctx.Context, db, ipAndPort)
		},
	}
}
