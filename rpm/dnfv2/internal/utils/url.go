package utils

import (
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
