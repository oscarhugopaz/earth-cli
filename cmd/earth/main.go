package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/oscarhugopaz/earth-cli/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	os.Exit(cli.Run(ctx, os.Args[1:], cli.Environment{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}))
}
