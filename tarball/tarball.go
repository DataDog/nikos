package tarball

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"github.com/xi2/xz"
)

func ExtractTarball(reader io.Reader, filename, directory string) error {
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
		return errors.Wrapf(err, "failed to read %s", filename)
	}

	tarReader := tar.NewReader(compressedTarReader)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return errors.Wrap(err, "failed to read entry from tarball")
		}

		path := filepath.Join(directory, hdr.Name)
		log.Debugf("Extracting %s to %s", hdr.Name, path)

		switch hdr.Typeflag {
		case tar.TypeSymlink:
			os.Symlink(hdr.Linkname, path)
		case tar.TypeDir:
			os.MkdirAll(path, 0755)
		case tar.TypeReg:
			output, err := os.Create(path)
			if err != nil {
				return errors.Wrapf(err, "failed to create output file '%s'", path)
			}

			if _, err := io.Copy(output, tarReader); err != nil {
				return errors.Wrapf(err, "failed to uncompress file", hdr.Name)

			}
			output.Close()
		default:
			log.Warnf("Unsupported header flag '%d' for '%s'", hdr.Typeflag, hdr.Name)
		}
	}

	return nil
}
