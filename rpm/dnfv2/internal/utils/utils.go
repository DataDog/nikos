package utils

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"fmt"
	"hash"
	"io"

	"github.com/DataDog/nikos/rpm/dnfv2/types"
)

func GetAndUnmarshalXML[T any](ctx context.Context, httpClient *HttpClient, url string, checksum *types.Checksum) (*T, error) {
	content, err := httpClient.GetWithChecksum(ctx, url, checksum)
	if err != nil {
		return nil, err
	}

	contentData, err := content.Data()
	if err != nil {
		return nil, err
	}

	var res T
	if err := xml.Unmarshal(contentData, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func verifyChecksum(reader io.Reader, checksum *types.Checksum) error {
	var contentSum []byte

	var hasher hash.Hash
	switch checksum.Type {
	case "sha256":
		hasher = sha256.New()
	case "sha1":
		hasher = sha1.New()
	default:
		return fmt.Errorf("unsupported sha type: %s", checksum.Type)
	}

	if _, err := io.Copy(hasher, reader); err != nil {
		return err
	}

	contentSum = hasher.Sum(nil)
	if checksum.Hash != fmt.Sprintf("%x", contentSum) {
		return errors.New("failed checksum")
	}

	return nil
}
