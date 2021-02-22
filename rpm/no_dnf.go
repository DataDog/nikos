// +build !dnf,!librepo

package rpm

import (
	"errors"

	"github.com/lebauce/nikos/types"
)

func NewRedHatBackend(target *types.Target) (types.Backend, error) {
	return nil, errors.New("dnf backend not supported")
}

func NewCentOSBackend(target *types.Target) (types.Backend, error) {
	return nil, errors.New("dnf backend not supported")
}

func NewOpenSUSEBackend(target *types.Target) (types.Backend, error) {
	return nil, errors.New("dnf backend not supported")
}

func NewSLESBackend(target *types.Target) (types.Backend, error) {
	return nil, errors.New("dnf backend not supported")
}
