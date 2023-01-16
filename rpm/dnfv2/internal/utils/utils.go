package utils

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/DataDog/nikos/rpm/dnfv2/types"
)

func GetAndUnmarshalXML[T any](ctx context.Context, httpClient *HttpClient, url string, checksum *types.Checksum) (*T, error) {
	content, err := httpClient.GetWithChecksum(ctx, url, checksum)
	if err != nil {
		return nil, err
	}

	var res T
	if err := xml.Unmarshal(content, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func verifyChecksum(content []byte, checksum *types.Checksum) error {
	var contentSum []byte

	switch checksum.Type {
	case "sha256":
		tmp := sha256.Sum256(content)
		contentSum = tmp[:]
	case "sha1":
		tmp := sha1.Sum(content)
		contentSum = tmp[:]
	default:
		return fmt.Errorf("unsupported sha type: %s", checksum.Type)
	}

	if checksum.Hash != fmt.Sprintf("%x", contentSum) {
		return errors.New("failed checksum")
	}

	return nil
}
