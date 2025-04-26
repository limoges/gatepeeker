package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/hairyhenderson/go-fsimpl"
	"github.com/hairyhenderson/go-fsimpl/blobfs"
	"github.com/hairyhenderson/go-fsimpl/filefs"
	"github.com/hairyhenderson/go-fsimpl/gitfs"
	"github.com/hairyhenderson/go-fsimpl/httpfs"
	"github.com/limoges/gatepeeker/internal/bundle"
	"github.com/urfave/cli/v3"
)

var (
	flagPolicies = &cli.StringSliceFlag{
		Name:  "policies",
		Usage: "A location to load policies from",
		Value: []string{},
	}
	flagVerbose = &cli.BoolFlag{
		Name:  "verbose",
		Usage: "Show display more log information",
		Value: false,
	}
	flagBuildOCI = &cli.BoolFlag{
		Name:  "build-oci",
		Usage: "EXPERIMENTAL: build+push an oci image",
	}
)

func logging(ctx context.Context, cmd *cli.Command) (context.Context, error) {

	verbose := cmd.Bool(flagVerbose.Name)

	logLevel := slog.LevelWarn
	if verbose {
		logLevel = slog.LevelInfo
	}
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.Attr{} // remove timestamp
			}
			if attr.Key == slog.LevelKey {
				return slog.Attr{} // remove log level
			}
			return attr
		},
	})

	slog.SetDefault(slog.New(h))

	return ctx, nil

}

func getWorkingDirURL() string {
	wd, _ := os.Getwd()
	abs, _ := filepath.Abs(wd)
	u := &url.URL{}
	u.Path = abs
	u.Scheme = "file"
	return u.String()
}

func formatURL(in string) (searchPath string, err error) {
	if in == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		searchPath = fmt.Sprintf("file://%s", wd)
	} else {
		u, err := url.Parse(in)
		if err != nil {
			return "", fmt.Errorf("failed to parse: %w", err)
		}
		if u.Scheme == "" {
			abs, err := filepath.Abs(in)
			if err != nil {
				return "", fmt.Errorf("failed to convert to absolute path: %w", err)
			}
			searchPath = fmt.Sprintf("file://%s", abs)
		} else {
			searchPath = u.String()
		}
	}
	return searchPath, nil
}

func fsFromURL(u string) (fs.FS, error) {
	m := fsimpl.NewMux()
	m.Add(filefs.FS)
	m.Add(httpfs.FS)
	m.Add(blobfs.FS)
	m.Add(gitfs.FS)
	return m.Lookup(u)
}

func loadBundle(ctx context.Context, cmd *cli.Command) (*bundle.Bundle, error) {
	urls := cmd.StringSlice(flagPolicies.Name)

	b := bundle.New()

	for _, item := range urls {
		u, err := formatURL(item)
		if err != nil {
			return nil, err
		}

		base, err := url.Parse(u)
		if err != nil {
			return nil, err
		}

		filename := filepath.Base(base.Path)
		dir := filepath.Dir(base.Path)

		base.Path = dir

		fsys, err := fsFromURL(base.String())
		if err != nil {
			return nil, err
		}

		buf, err := fs.ReadFile(fsys, filename)
		if err != nil {
			return nil, err
		}

		parsed, err := bundle.ParsePolicies(buf)
		if err != nil {
			return nil, err
		}

		b.Merge(parsed)
	}

	if len(b.GetConstraintTemplates()) == 0 {
		return nil, errors.New("no templates were found")
	}

	if len(b.GetConstraints()) == 0 {
		return nil, errors.New("no constraints were found")
	}

	return b, nil
}

type FileLoader struct {
	mux fsimpl.FSMux
}

func NewFileLoader() *FileLoader {
	l := &FileLoader{}
	mux := fsimpl.NewMux()
	mux.Add(filefs.FS)
	mux.Add(httpfs.FS)
	mux.Add(gitfs.FS)
	l.mux = mux
	return l
}

func (f *FileLoader) IsSupportedURL(s string) bool {
	_, err := f.mux.Lookup(s)
	return err == nil
}

func (f *FileLoader) ReadFile(s string) ([]byte, error) {

	base, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	// File-systems support only directories, not file as targets.
	filename := filepath.Base(base.Path)
	base.Path = strings.TrimSuffix(base.Path, filename)

	target := base.String()
	slog.Info("Opening", "target", target, "filename", filename)
	fsys, err := f.mux.Lookup(target)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup: %w", err)
	}

	buf, err := fs.ReadFile(fsys, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return buf, nil
}

var loader = NewFileLoader()

func readSource(s string) ([]byte, error) {
	if s == "" {
		return nil, errors.New("empty source")
	}

	if loader.IsSupportedURL(s) {
		return loader.ReadFile(s)
	}

	slog.Info("URL is not supported", "url", s)
	fi, err := os.Stat(s)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, errors.New("argument must point to a file, not a directory")
	}
	return os.ReadFile(s)
}

func readFromStdin() ([]byte, error) {

	info, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeNamedPipe != 0 {
		slog.Info("Reading input from stdin")
		return io.ReadAll(os.Stdin)
	}
	return nil, nil
}
