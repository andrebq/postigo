package main

import (
	"github.com/andrebq/postigo/internal/kdb"
	"github.com/andrebq/postigo/internal/server"
	"github.com/urfave/cli/v2"
)

func ctlCmd() *cli.Command {
	var db *kdb.DB
	dataFlag, openDB := dataDirFlag()
	return &cli.Command{
		Name:    "control",
		Aliases: []string{"ctl"},
		Flags: []cli.Flag{
			dataFlag,
		},
		Before: func(ctx *cli.Context) error {
			var err error
			db, _, err = openDB()
			return err
		},
		Subcommands: []*cli.Command{
			ctlRegisterKeyCmd(&db),
		},
	}
}

func ctlRegisterKeyCmd(db **kdb.DB) *cli.Command {
	var kid string
	var exposes cli.StringSlice
	var dials cli.StringSlice
	return &cli.Command{
		Name: "register-key",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "kid",
				Usage:       "Key ID that is being modified, usually base64 encoding of the public key component. Usually printed when a node is starting",
				Destination: &kid,
				Required:    true,
			},
			&cli.StringSliceFlag{
				Name:        "expose",
				Usage:       "List of nodenames that this key is authorized to expose",
				Destination: &exposes,
			},
			&cli.StringSliceFlag{
				Name:        "dial",
				Usage:       "List of nodenames that this key is authorized to dial, use '*' to allow all",
				Destination: &dials,
			},
		},
		Action: func(ctx *cli.Context) error {
			return server.RegisterKey(ctx.Context, *db, kid, exposes.Value(), dials.Value())
		},
	}
}
