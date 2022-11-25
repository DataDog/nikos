//go:build !linux

package backend

import (
	"crypto/x509"
	"errors"
)

func GetSystemRoots() (*x509.CertPool, error) {
	return nil, errors.New("unimplemented on this platform")
}
