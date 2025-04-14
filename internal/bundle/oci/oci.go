package oci

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	oras "oras.land/oras-go/v2"
)

func BuildImage(ctx context.Context, fsys fs.FS, target oras.Target) error {
	// Build an archive and pack it into the image
	var tar bytes.Buffer

	if err := archive(&tar, fsys); err != nil {
		return err
	}

	archiveDigest := digest.FromBytes(tar.Bytes())

	var layer bytes.Buffer
	err := compress(&layer, bytes.NewReader(tar.Bytes()))
	if err != nil {
		return err
	}
	compressedDigest := digest.FromBytes(layer.Bytes())
	slog.Info("Built archive from filesystem", "digest", archiveDigest)
	slog.Info("Compressed archive", "digest", compressedDigest)

	desc := v1.Descriptor{}
	desc.Size = int64(len(layer.Bytes()))
	desc.MediaType = "application/vnd.oci.image.layer.v1.tar+gzip"
	desc.Digest = compressedDigest

	err = target.Push(ctx, desc, bytes.NewReader(layer.Bytes()))
	if err != nil {
		return fmt.Errorf("failed to push layer: %w", err)
	}

	artifactType := "application/vnd.oci.image.manifest.v1+json"
	opts := oras.PackManifestOptions{
		Layers: []v1.Descriptor{desc},
	}
	manifest, err := oras.PackManifest(ctx, target, oras.PackManifestVersion1_1, artifactType, opts)
	if err != nil {
		return fmt.Errorf("failed to pack manifest: %w", err)
	}

	err = target.Tag(ctx, manifest, "latest")
	if err != nil {
		return err
	}

	return nil
}

func archive(w io.Writer, fsys fs.FS) error {
	arch := tar.NewWriter(w)
	err := arch.AddFS(fsys)
	if err != nil {
		return err
	}
	return arch.Close()
}

func compress(w io.Writer, r io.Reader) error {
	zip := gzip.NewWriter(w)
	_, err := io.Copy(zip, r)
	if err != nil {
		return err
	}

	err = zip.Flush()
	if err != nil {
		return err
	}

	return zip.Close()
}
