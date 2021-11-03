package rpm

import (
	"fmt"

	"github.com/DataDog/nikos/rpm/dnf"
	"github.com/DataDog/nikos/types"
)

type RedHatBackend struct {
	dnfBackend *dnf.DnfBackend
	logger     types.Logger
	target     *types.Target
}

func (b *RedHatBackend) GetKernelHeaders(directory string) error {
	// First, check for the correct kernel-headers package
	pkgNevra := "kernel-headers-" + b.target.Uname.Kernel
	err := b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
	if err == nil {
		return nil
	}

	// As a last resort, check for the kernel-devel package
	pkgNevra = "kernel-devel-" + b.target.Uname.Kernel
	return b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
}

func (b *RedHatBackend) Close() {
	b.dnfBackend.Close()
}

func NewRedHatBackend(target *types.Target, reposDir string, logger types.Logger) (*RedHatBackend, error) {
	dnfBackend, err := dnf.NewDnfBackend(target.Distro.Release, reposDir, logger, target)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNF backend: %w", err)
	}

	return &RedHatBackend{
		target:     target,
		logger:     logger,
		dnfBackend: dnfBackend,
	}, nil
}
