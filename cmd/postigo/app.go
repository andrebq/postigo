package main

import (
	"io"
	"log/slog"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

func App(out, err io.Writer) *cli.App {
	var debug bool
	return &cli.App{
		Name: "postigo",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "debug",
				Hidden:      true,
				Destination: &debug,
			},
		},
		Before: func(ctx *cli.Context) error {
			level := slog.LevelInfo
			if debug {
				level = slog.LevelDebug
			}
			var handler slog.Handler
			fd, _ := ctx.App.ErrWriter.(*os.File)
			if fd != nil && isatty.IsTerminal(fd.Fd()) {
				handler = slog.NewTextHandler(ctx.App.ErrWriter, &slog.HandlerOptions{
					Level: level,
				})
			}
			if handler == nil {
				handler = slog.NewJSONHandler(ctx.App.ErrWriter, &slog.HandlerOptions{
					Level: level,
				})
			}
			slog.SetDefault(slog.New(handler))
			return nil
		},
		Commands: []*cli.Command{
			serverCmd(),
			clientCmd(),
		},
		Writer:    out,
		ErrWriter: err,
	}
}
