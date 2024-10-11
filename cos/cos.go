package cos

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/DataDog/nikos/extract"
	"github.com/DataDog/nikos/types"
)

type Backend struct {
	buildID string
	logger  types.Logger
}

const (
	kernelHeadersFilename = "kernel-headers.tgz"
	bucketName            = "cos-tools"
)

func (b *Backend) GetKernelHeaders(directory string) error {
	objectName := url.QueryEscape(fmt.Sprintf("%s/%s", b.buildID, kernelHeadersFilename))
	url := fmt.Sprintf("https://storage.googleapis.com/download/storage/v1/b/%s/o/%s?alt=media", bucketName, objectName)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to start download kernel headers from COS bucket: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download kernel headers from COS bucket: %s", resp.Status)
	}

	if err := extract.ExtractTarball(resp.Body, kernelHeadersFilename, directory, b.logger); err != nil {
		return fmt.Errorf("failed to extract kernel headers: %w", err)
	}

	return nil
}

func (b *Backend) Close() {}

func NewBackend(target *types.Target, logger types.Logger) (*Backend, error) {
	buildID := target.OSRelease["BUILD_ID"]
	if buildID == "" {
		return nil, errors.New("failed to detect COS version, missing BUILD_ID in /etc/os-release")
	}

	return &Backend{
		logger:  logger,
		buildID: buildID,
	}, nil
}
