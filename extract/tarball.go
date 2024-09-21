package extract

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/DataDog/nikos/types"
	"github.com/klauspost/compress/zstd"
	"github.com/xi2/xz"
)

type onlyWriter struct {
	io.Writer
}

func ExtractTarball(reader io.Reader, filename, directory string, logger types.Logger) error {
	var compressedTarReader io.Reader
	var err error
	switch filepath.Ext(filename) {
	case ".xz":
		compressedTarReader, err = xz.NewReader(reader, 0)
	case ".gz", ".tgz":
		compressedTarReader, err = gzip.NewReader(reader)
	case ".bz2":
		compressedTarReader = bzip2.NewReader(reader)
	case ".zst":
		zstdReader, zstdErr := zstd.NewReader(reader)
		defer zstdReader.Close()
		err = zstdErr
		compressedTarReader = zstdReader
	default:
		return fmt.Errorf("failed to extract %s", filename)
	}

	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filename, err)
	}

	buf := make([]byte, 50)
	tarReader := tar.NewReader(compressedTarReader)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("failed to read entry from tarball: %w", err)
		}

		path := filepath.Join(directory, hdr.Name)
		// logger.Debugf("Extracting %s to %s", hdr.Name, path)

		switch hdr.Typeflag {
		case tar.TypeSymlink:
			// If the symlink is to an absolute path, prefix it with the download directory
			if strings.HasPrefix(hdr.Linkname, "/") {
				os.Symlink(filepath.Join(directory, hdr.Linkname), path)
				continue
			}
			// If the symlink is to a relative path, leave it be
			os.Symlink(hdr.Linkname, path)
		case tar.TypeDir:
			os.MkdirAll(path, 0755)
		case tar.TypeReg:
			output, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("failed to create output file '%s': %w", path, err)
			}

			// By default, an os.File implements the io.ReaderFrom interface.
			// As a result, CopyBuffer will attempt to use the output.ReadFrom method to perform
			// the requested copy, which ends up calling the unbuffered io.Copy function & performing
			// a large number of allocations.
			// In order to force CopyBuffer to actually utilize the given buffer, we have to ensure
			// output does not implement the io.ReaderFrom interface.
			if _, err := io.CopyBuffer(onlyWriter{output}, tarReader, buf); err != nil {
				return fmt.Errorf("failed to uncompress file %s: %w", hdr.Name, err)

			}
			output.Close()
		default:
			logger.Warnf("Unsupported header flag '%d' for '%s'", hdr.Typeflag, hdr.Name)
		}
	}

	return nil
}
