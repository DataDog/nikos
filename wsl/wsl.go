package wsl

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"

	"github.com/DataDog/nikos/tarball"
	"github.com/DataDog/nikos/types"
)

type Backend struct {
	target *types.Target
}

func (b *Backend) GetKernelHeaders(directory string) error {
	filename := b.target.Uname.Kernel + ".tar.gz"
	url := fmt.Sprintf("https://codeload.github.com/microsoft/WSL2-Linux-Kernel/tar.gz/%s", b.target.Uname.Kernel)

	tempfile, err := ioutil.TempFile("", "wsl-headers")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary file for kernel headers")
	}
	defer os.Remove(tempfile.Name())

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return tarball.ExtractTarball(resp.Body, filename, directory)
}

func NewBackend(target *types.Target) (*Backend, error) {
	backend := &Backend{
		target: target,
	}

	return backend, nil
}
