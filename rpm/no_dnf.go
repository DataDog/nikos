// +build !dnf,!librepo

package rpm

import (
	"errors"

	"github.com/DataDog/nikos/types"
)

func NewBackend(_ *types.Target, _ types.Logger) (types.Backend, error) {
	return nil, errors.New("dnf backend not supported")
}
