package cmd

import (
	"github.com/urfave/cli/v3"
)

func BuildRoot() *cli.Command {
	cmd := &cli.Command{}
	cmd.Name = "gatepeeker"
	cmd.Usage = "Validate configurations against Gatekeeper policies"
	cmd.Commands = []*cli.Command{
		ValidateCmd(),
		BuildCmd(),
	}
	return cmd
}
