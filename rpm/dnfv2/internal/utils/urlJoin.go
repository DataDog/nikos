package utils

import (
	"net/url"
	"path"
)

func UrlJoinPath(base string, elems ...string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	u.Path = path.Join(append([]string{u.Path}, elems...)...)
	return u.String(), nil
}
