package bundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"path/filepath"

	"github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/helper/iofs"
	"github.com/go-git/go-billy/v6/memfs"
	"github.com/go-git/go-billy/v6/util"
)

// WriteFS writes a Bundle to the provide file-system.
func WriteFS(b *Bundle, fsys billy.Filesystem, baseDir string) error {
	path := "policies.yaml"
	if baseDir != "" {
		err := fsys.MkdirAll(baseDir, 0755)
		if err != nil {
			return err
		}
		path = filepath.Join(baseDir, path)
	}

	buf, err := WriteYAML(b)
	if err != nil {
		return err
	}
	return util.WriteFile(fsys, path, buf, 0644)
}

func WriteCompressedArchive(b *Bundle, w io.Writer) error {
	var (
		buf  bytes.Buffer
		fsys = memfs.New()
	)
	err := WriteFS(b, fsys, "")
	if err != nil {
		return err
	}
	archive := tar.NewWriter(&buf)
	err = archive.AddFS(iofs.New(fsys))
	if err != nil {
		return err
	}
	err = archive.Close()
	if err != nil {
		return err
	}
	zip := gzip.NewWriter(w)
	_, err = io.Copy(zip, &buf)
	if err != nil {
		return err
	}
	err = zip.Flush()
	if err != nil {
		return err
	}
	return zip.Close()
}

func WriteYAML(b *Bundle) ([]byte, error) {
	var objects [][]byte
	for _, obj := range b.templates {
		objects = append(objects, obj.getRaw())
	}
	for _, obj := range b.constraints {
		objects = append(objects, obj.getRaw())
	}

	var buf bytes.Buffer
	for _, obj := range objects {
		// we don't write newline because raw may contains it
		if _, err := buf.Write([]byte("---")); err != nil {
			return nil, err
		}
		if _, err := buf.Write(obj); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}
