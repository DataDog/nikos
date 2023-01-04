package utils

import (
	"compress/gzip"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/DataDog/nikos/rpm/dnfv2/types"
)

func GetAndUnmarshalXML[T any](httpClient *http.Client, url string, checksum *types.Checksum) (*T, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status for `%s`: %s", url, resp.Status)
	}

	var reader io.Reader = resp.Body
	if UrlHasSuffix(url, ".gz") {
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	if checksum != nil {
		if err := verifyChecksum(content, checksum); err != nil {
			return nil, err
		}
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
