package utils

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/DataDog/nikos/rpm/dnfv2/types"
)

type HttpClientCacheEntry struct {
	repoID uintptr
	url    string
}

type HttpClientCache struct {
	sync.RWMutex
	cache map[HttpClientCacheEntry][]byte
}

func NewHttpClientCache() *HttpClientCache {
	return &HttpClientCache{
		cache: make(map[HttpClientCacheEntry][]byte),
	}
}

func (c *HttpClientCache) Get(repoID uintptr, url string) ([]byte, bool) {
	c.RLock()
	defer c.RUnlock()

	content, ok := c.cache[HttpClientCacheEntry{repoID, url}]
	return content, ok
}

func (c *HttpClientCache) Set(repoID uintptr, url string, content []byte) {
	c.Lock()
	defer c.Unlock()

	c.cache[HttpClientCacheEntry{repoID, url}] = content
}

type HttpClient struct {
	inner  *http.Client
	cache  *HttpClientCache
	repoID uintptr
}

func NewHttpClientFromInner(inner *http.Client, cache *HttpClientCache, repoID uintptr) *HttpClient {
	return &HttpClient{inner: inner, cache: cache, repoID: repoID}
}

func (hc *HttpClient) GetWithChecksum(ctx context.Context, url string, checksum *types.Checksum) ([]byte, error) {
	content, ok := hc.cache.Get(hc.repoID, url)

	if !ok {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := hc.inner.Do(req)
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

		readContent, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		content = readContent
	}

	if checksum != nil {
		if err := verifyChecksum(content, checksum); err != nil {
			return nil, err
		}
	}

	hc.cache.Set(hc.repoID, url, content)
	return content, nil
}

func (hc *HttpClient) Get(ctx context.Context, url string) ([]byte, error) {
	return hc.GetWithChecksum(ctx, url, nil)
}
