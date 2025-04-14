package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/limoges/gatepeeker/internal/cmd"
)

func main() {
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error("terminated with error", "error", err)
		os.Exit(1)
	}
}
