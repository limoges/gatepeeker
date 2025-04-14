package cmd

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"fmt"

	"github.com/limoges/gatepeeker/internal/bundle"
	"github.com/limoges/gatepeeker/internal/validating"
	"github.com/urfave/cli/v3"
)

func ValidateCmd() *cli.Command {
	cmd := &cli.Command{}
	cmd.Name = "validate"
	cmd.Usage = "Validate manifests against a policy bundle"
	cmd.Action = validate
	cmd.Flags = []cli.Flag{
		flagPolicies,
		flagVerbose,
	}
	return cmd
}

func validate(ctx context.Context, cmd *cli.Command) error {
	logging(ctx, cmd)

	b := bundle.New()

	urlstrs := cmd.StringSlice(flagPolicies.Name)
	for _, urlstr := range urlstrs {

		buf, err := readSource(urlstr)
		if err != nil {
			return fmt.Errorf("failed to read arg source: %w", err)
		}

		argBundle, err := bundle.ParsePolicies(buf)
		if err != nil {
			return fmt.Errorf("failed to build bundle from yaml: %w", err)
		}
		b.Merge(argBundle)
	}

	client, err := validating.NewClientWithBundle(ctx, b)
	if err != nil {
		return err
	}

	var (
		failures int
		output   = os.Stdout

		inputs [][]byte
	)

	// Read resources to validate from stdin
	stdin, err := readFromStdin()
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	if len(stdin) > 0 {
		inputs = append(inputs, stdin)
	}

	for _, arg := range cmd.Args().Slice() {
		buf, err := readSource(arg)
		if err != nil {
			slog.Error("failed to read source", "source", arg, "error", err)
			continue
		}
		inputs = append(inputs, buf)
	}

	if len(inputs) == 0 {
		return errors.New("no files were validated")
	}

	for _, input := range inputs {
		report, err := client.Validate(ctx, input)
		if err != nil {
			slog.Error("failed to validate", "error", err)
			continue
		}
		failures += report.FailureCount()
		report.WriteTo(output)
	}

	if failures > 0 {
		slog.Error("validation failed", "failed", failures)
		os.Exit(2)
	}
	return nil
}
