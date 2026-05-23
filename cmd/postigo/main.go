package main

import (
	"context"
	"log"
	"os"
	"os/signal"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	app := App(os.Stdout, os.Stderr)
	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Fatalf("Fatal error: %v", err)
	}
}
