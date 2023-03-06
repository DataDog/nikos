package utils

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/DataDog/nikos/rpm/dnfv2/types"
)

type FetchedData struct {
	data    []byte
	gzipped bool
}

func (d *FetchedData) Reader() (io.ReadCloser, error) {
	r := bytes.NewReader(d.data)
	if d.gzipped {
		return gzip.NewReader(r)
	}
	return io.NopCloser(r), nil
}

func (d *FetchedData) Data() ([]byte, error) {
	reader, err := d.Reader()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

type HttpClient struct {
	inner  *http.Client
	repoID uintptr
}

func NewHttpClientFromInner(inner *http.Client, repoID uintptr) *HttpClient {
	return &HttpClient{inner: inner, repoID: repoID}
}

func (hc *HttpClient) GetWithChecksum(ctx context.Context, url string, checksum *types.Checksum) (FetchedData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return FetchedData{}, err
	}

	resp, err := hc.inner.Do(req)
	if err != nil {
		return FetchedData{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return FetchedData{}, fmt.Errorf("bad status for `%s`: %s", url, resp.Status)
	}

	gzipped := UrlHasSuffix(url, ".gz") || resp.Header.Get("Content-Encoding") == "gzip"
	readContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return FetchedData{}, err
	}
	content := FetchedData{data: readContent, gzipped: gzipped}

	if checksum != nil {
		contentReader, err := content.Reader()
		if err != nil {
			return FetchedData{}, err
		}
		defer contentReader.Close()

		if err := verifyChecksum(contentReader, checksum); err != nil {
			return FetchedData{}, err
		}
	}

	return content, nil
}

func (hc *HttpClient) Get(ctx context.Context, url string) (FetchedData, error) {
	return hc.GetWithChecksum(ctx, url, nil)
}
