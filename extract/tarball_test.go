package extract

import (
	"io"
	"net/http"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
)

func downloadFile(url string) (*os.File, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	out, err := os.CreateTemp("", "src-tarball")
	if err != nil {
		return nil, err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return nil, err
	}
	return out, nil
}

var srcTarball *os.File

func init() {
	tarballUrl := "http://archive.ubuntu.com/ubuntu/pool/main/l/linux-oem-5.6/linux-oem-5.6_5.6.0.orig.tar.gz"
	srcFile, err := downloadFile(tarballUrl)
	if err != nil {
		panic(err)
	}
	srcTarball = srcFile
}

func BenchmarkExtractTarball(b *testing.B) {
	logger := log.New()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		tempDir, err := os.MkdirTemp("", "extract_tarball_test")
		if err != nil {
			b.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		ExtractTarball(srcTarball, "linux-oem-5.6_5.6.0.orig.tar.gz", tempDir, logger)
	}
}
