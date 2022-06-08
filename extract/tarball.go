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
	"github.com/DataDog/zstd"
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
	case ".zst":
		zstdReader := zstd.NewReader(reader)
		defer zstdReader.Close()
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

			// if _, err := io.CopyBuffer(output, tarReader, buf); err != nil {
			// 	return fmt.Errorf("failed to uncompress file %s: %w", hdr.Name, err)

			// }

			type onlyWriter struct {
				io.Writer
			}

			if _, err := customCopy(onlyWriter{output}, tarReader, buf, logger); err != nil {
				return fmt.Errorf("failed to uncompress file %s: %w", hdr.Name, err)

			}
			output.Close()
		default:
			logger.Warnf("Unsupported header flag '%d' for '%s'", hdr.Typeflag, hdr.Name)
		}
	}

	return nil
}

func customCopy(dst io.Writer, src io.Reader, buf []byte, logger types.Logger) (written int64, err error) {
	if wt, ok := src.(io.WriterTo); ok {
		logger.Infof("tarReader is problem")
		return wt.WriteTo(dst)
	}
	if rt, ok := dst.(io.ReaderFrom); ok {
		logger.Infof("output is problem")
		return rt.ReadFrom(src)
	}
	logger.Infof("CopyBuffer is problem ????")
	return io.CopyBuffer(dst, src, buf)
}