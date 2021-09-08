package extract

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/DataDog/nikos/types"
	"github.com/xi2/xz"
)

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
	}

	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filename, err)
	}

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
			os.Symlink(filepath.Join(directory, hdr.Linkname), path)
		case tar.TypeDir:
			os.MkdirAll(path, 0755)
		case tar.TypeReg:
			output, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("failed to create output file '%s': %w", path, err)
			}

			if _, err := io.Copy(output, tarReader); err != nil {
				return fmt.Errorf("failed to uncompress file %s: %w", hdr.Name, err)

			}
			output.Close()
		default:
			logger.Warnf("Unsupported header flag '%d' for '%s'", hdr.Typeflag, hdr.Name)
		}
	}

	return nil
}