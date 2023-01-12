package utils

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"strings"
)

func UrlJoinPath(base string, elems ...string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	u.Path = path.Join(append([]string{u.Path}, elems...)...)
	return u.String(), nil
}

func UrlHasSuffix(rawUrl string, suffix string) bool {
	parsed, err := url.Parse(rawUrl)
	if err != nil {
		return strings.HasSuffix(rawUrl, suffix)
	}

	return strings.HasSuffix(parsed.Path, suffix)
}

func HttpGet(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return client.Do(req)
}
