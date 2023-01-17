package utils

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/DataDog/nikos/rpm/dnfv2/types"
)

type HttpClientCacheEntry struct {
	repoID uintptr
	url    string
}

type HttpClientCache struct {
	sync.RWMutex
	cache map[HttpClientCacheEntry]FetchedData
}

func NewHttpClientCache() *HttpClientCache {
	return &HttpClientCache{
		cache: make(map[HttpClientCacheEntry]FetchedData),
	}
}

func (c *HttpClientCache) Get(repoID uintptr, url string) (FetchedData, bool) {
	c.RLock()
	defer c.RUnlock()

	content, ok := c.cache[HttpClientCacheEntry{repoID, url}]
	return content, ok
}

func (c *HttpClientCache) Set(repoID uintptr, url string, content FetchedData) {
	c.Lock()
	defer c.Unlock()

	c.cache[HttpClientCacheEntry{repoID, url}] = content
}

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
	cache  *HttpClientCache
	repoID uintptr
}

func NewHttpClientFromInner(inner *http.Client, cache *HttpClientCache, repoID uintptr) *HttpClient {
	return &HttpClient{inner: inner, cache: cache, repoID: repoID}
}

func (hc *HttpClient) GetWithChecksum(ctx context.Context, url string, checksum *types.Checksum) (FetchedData, error) {
	fmt.Printf("get: %s\n", url)
	content, ok := hc.cache.Get(hc.repoID, url)

	if !ok {
		fmt.Println("not cached")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return FetchedData{}, err
		}

		start := time.Now()
		resp, err := hc.inner.Do(req)
		if err != nil {
			return FetchedData{}, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return FetchedData{}, fmt.Errorf("bad status for `%s`: %s", url, resp.Status)
		}

		gzipped := UrlHasSuffix(url, ".gz")
		readContent, err := io.ReadAll(resp.Body)
		if err != nil {
			return FetchedData{}, err
		}
		fmt.Printf("in: %v\n", time.Since(start))
		content = FetchedData{data: readContent, gzipped: gzipped}
	}

	if checksum != nil {
		contentReader, err := content.Reader()
		defer contentReader.Close()
		if err != nil {
			return FetchedData{}, err
		}

		if err := verifyChecksum(contentReader, checksum); err != nil {
			return FetchedData{}, err
		}
	}

	hc.cache.Set(hc.repoID, url, content)
	fmt.Printf("size: %d\n", len(content.data))
	return content, nil
}

func (hc *HttpClient) Get(ctx context.Context, url string) (FetchedData, error) {
	return hc.GetWithChecksum(ctx, url, nil)
}
