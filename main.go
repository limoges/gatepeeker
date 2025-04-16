package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/limoges/gatepeeker/internal/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	root := cmd.BuildRoot()
	root.Version = fmt.Sprintf("s commit=%s date=%s", version, commit, date)

	if err := root.Run(context.Background(), os.Args); err != nil {
		slog.Error("terminated with error", "error", err)
		os.Exit(1)
	}
}
