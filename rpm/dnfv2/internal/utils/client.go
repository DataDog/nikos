package utils

import (
	"context"
	"net/http"
)

type HttpClient struct {
	inner *http.Client
}

func NewHttpClientFromInner(inner *http.Client) *HttpClient {
	return &HttpClient{inner: inner}
}

func (hc *HttpClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return hc.inner.Do(req)
}