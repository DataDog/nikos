package cos

import (
	"context"

	"cloud.google.com/go/storage"
	"github.com/DataDog/nikos/tarball"
	"github.com/DataDog/nikos/types"
	"github.com/pkg/errors"
	"google.golang.org/api/option"
)

type Backend struct {
	buildID string
	client  *storage.Client
}

func (b *Backend) GetKernelHeaders(directory string) error {
	filename := "kernel-headers.tgz"
	bucketHandle := b.client.Bucket("cos-tools")
	objectHandle := bucketHandle.Object(b.buildID + "/" + filename)
	reader, err := objectHandle.NewReader(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to download bucket object")
	}
	defer reader.Close()

	tarball.ExtractTarball(reader, filename, directory)
	return err
}

func NewBackend(target *types.Target) (*Backend, error) {
	buildID := target.OSRelease["BUILD_ID"]
	if buildID == "" {
		return nil, errors.New("failed to detect COS version, missing BUILD_ID in /etc/os-release")
	}

	client, err := storage.NewClient(context.Background(), option.WithoutAuthentication())
	if err != nil {
		return nil, errors.Wrap(err, "failed to creating COS backend")
	}

	return &Backend{
		client:  client,
		buildID: buildID,
	}, nil
}
