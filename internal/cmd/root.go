package cmd

import (
	"context"

	"github.com/urfave/cli/v3"
)

func Run(ctx context.Context, args []string) error {
	return RootCmd().Run(ctx, args)
}

func RootCmd() *cli.Command {
	cmd := &cli.Command{}
	cmd.Name = "gatepeeker"
	cmd.Usage = "Validate configurations against Gatekeeper policies"
	cmd.Commands = []*cli.Command{
		ValidateCmd(),
		BuildCmd(),
	}
	return cmd
}
