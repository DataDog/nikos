package cos

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/DataDog/nikos/extract"
	"github.com/DataDog/nikos/types"
	"google.golang.org/api/option"
)

type Backend struct {
	buildID string
	logger  types.Logger
	client  *storage.Client
}

func (b *Backend) GetKernelHeaders(directory string) error {
	filename := "kernel-headers.tgz"
	bucketHandle := b.client.Bucket("cos-tools")
	objectHandle := bucketHandle.Object(b.buildID + "/" + filename)
	reader, err := objectHandle.NewReader(context.Background())
	if err != nil {
		return fmt.Errorf("failed to download bucket object: %w", err)
	}
	defer reader.Close()

	extract.ExtractTarball(reader, filename, directory, b.logger)
	return err
}

func (b *Backend) Close() {}

func NewBackend(target *types.Target, logger types.Logger) (*Backend, error) {
	buildID := target.OSRelease["BUILD_ID"]
	if buildID == "" {
		return nil, errors.New("failed to detect COS version, missing BUILD_ID in /etc/os-release")
	}

	client, err := storage.NewClient(context.Background(), option.WithoutAuthentication())
	if err != nil {
		return nil, fmt.Errorf("failed to creating COS backend: %w", err)
	}

	return &Backend{
		client:  client,
		logger:  logger,
		buildID: buildID,
	}, nil
}
