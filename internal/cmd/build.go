package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/go-git/go-billy/v6/helper/iofs"
	"github.com/go-git/go-billy/v6/memfs"
	"github.com/limoges/gatepeeker/internal/bundle"
	"github.com/limoges/gatepeeker/internal/bundle/oci"
	"github.com/urfave/cli/v3"
	orasoci "oras.land/oras-go/v2/content/oci"
)

func BuildCmd() *cli.Command {

	cmd := &cli.Command{}
	cmd.Name = "build"
	cmd.Usage = "Bundle policies into an artifact"
	cmd.Description = `
You can output policies found in a multi-document yaml file to stdout:
$ cat manifests.yaml | gatepeeker bundle

Redirecting to a file
$ cat manifests.yaml | gatepeeker bundle > policies.yaml
`
	cmd.Action = build
	cmd.Flags = []cli.Flag{
		flagPolicies,
		flagVerbose,
	}
	if os.Getenv("GATEPEEKER_EXPERIMENTAL") != "" {
		cmd.Flags = append(cmd.Flags, flagBuildOCI)
	}
	return cmd
}

func build(ctx context.Context, cmd *cli.Command) error {
	logging(ctx, cmd)

	if cmd.Args().Len() > 0 {
		slog.Warn(fmt.Sprintf("cli args are not used but %v were found", cmd.Args().Len()))
	}

	b := bundle.New()

	stdin, err := readFromStdin()
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	if len(stdin) > 0 {
		stdinBundle, err := bundle.ParsePolicies(stdin)
		if err != nil {
			return fmt.Errorf("failed to build bundle from yaml: %w", err)
		}
		b.Merge(stdinBundle)
	}

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

	buf, err := bundle.WriteYAML(b)
	if err != nil {
		return fmt.Errorf("failed to write policies: %w", err)
	}

	if _, err := io.Copy(os.Stdout, bytes.NewReader(buf)); err != nil {
		return fmt.Errorf("failed to copy policies: %w", err)
	}

	if cmd.Bool(flagBuildOCI.Name) {
		return buildOCI(ctx, b, "oci-output")
	}

	return nil
}

func buildOCI(ctx context.Context, b *bundle.Bundle, outputDir string) error {
	fsys := memfs.New()
	err := bundle.WriteFS(b, fsys, "")
	if err != nil {
		return fmt.Errorf("failed to write fs: %w", err)
	}
	storage, err := orasoci.New(outputDir)
	if err != nil {
		return fmt.Errorf("failed to create oras storage: %w", err)
	}
	return oci.BuildImage(ctx, iofs.New(fsys), storage)
}
