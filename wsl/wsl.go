package wsl

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/DataDog/nikos/extract"
	"github.com/DataDog/nikos/types"
)

type Backend struct {
	target *types.Target
	logger types.Logger
}

func (b *Backend) GetKernelHeaders(directory string) error {
	filename := b.target.Uname.Kernel + ".tar.gz"
	url := fmt.Sprintf("https://codeload.github.com/microsoft/WSL2-Linux-Kernel/tar.gz/%s", b.target.Uname.Kernel)

	tempfile, err := ioutil.TempFile("", "wsl-headers")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for kernel headers: %w", err)
	}
	defer os.Remove(tempfile.Name())

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return extract.ExtractTarball(resp.Body, filename, directory, b.logger)
}

func (b *Backend) Close() {}

func NewBackend(target *types.Target, logger types.Logger) (*Backend, error) {
	backend := &Backend{
		target: target,
		logger: logger,
	}

	return backend, nil
}
