// +build !dnf,!librepo

package dnf

import (
	"errors"

	"github.com/DataDog/nikos/types"
)

type Repository struct {
	Id string
}

type DnfBackend struct {
}

var errUnsupported = errors.New("dnf backend not supported")

func NewDnfBackend(_, _ string, logger types.Logger) (*DnfBackend, error) {
	return nil, errUnsupported
}

func (b *DnfBackend) GetEnabledRepositories() (repos []*Repository) { return nil }
func (b *DnfBackend) EnableRepository(repo *Repository) error       { return errUnsupported }
func (b *DnfBackend) DisableRepository(_ *Repository) error         { return errUnsupported }
func (b *DnfBackend) GetKernelHeaders(_, _ string) error            { return errUnsupported }
func (b *DnfBackend) AddRepository(_, _ string, _ bool, _ string) (*Repository, error) {
	return nil, errUnsupported
}
func (b *DnfBackend) Close() {}
