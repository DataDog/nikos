package rpm

import (
	"github.com/pkg/errors"

	"github.com/DataDog/nikos/rpm/dnf"
	"github.com/DataDog/nikos/types"
)

type RedHatBackend struct {
	dnfBackend *dnf.DnfBackend
	target     *types.Target
}

func (b *RedHatBackend) GetKernelHeaders(directory string) error {
	pkgNevra := "kernel-headers-" + b.target.Uname.Kernel
	return b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
}

func (b *RedHatBackend) Close() {
	b.dnfBackend.Close()
}

func NewRedHatBackend(target *types.Target, reposDir string, logger types.Logger) (*RedHatBackend, error) {
	dnfBackend, err := dnf.NewDnfBackend(target.Distro.Release, reposDir, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create DNF backend")
	}

	return &RedHatBackend{
		target:     target,
		dnfBackend: dnfBackend,
	}, nil
}
