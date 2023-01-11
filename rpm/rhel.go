package rpm

import (
	"fmt"

	"github.com/DataDog/nikos/rpm/dnfv2"
	"github.com/DataDog/nikos/rpm/dnfv2/backend"
	"github.com/DataDog/nikos/types"
)

type RedHatBackend struct {
	dnfBackend *backend.Backend
	logger     types.Logger
	target     *types.Target
}

func (b *RedHatBackend) GetKernelHeaders(directory string) error {
	pkgNevra := "kernel-devel"
	pkgMatcher := dnfv2.DefaultPkgMatcher(pkgNevra, b.target.Uname.Kernel)

	pkg, data, err := b.dnfBackend.FetchPackage(pkgMatcher)
	if err != nil {
		return fmt.Errorf("failed to fetch `%s` package: %w", pkgNevra, err)
	}

	return dnfv2.ExtractPackage(pkg, data, directory, b.target, b.logger)
}

func (b *RedHatBackend) Close() {
}

func NewRedHatBackend(target *types.Target, reposDir string, logger types.Logger) (*RedHatBackend, error) {
	b, err := dnfv2.NewBackend(target.Distro.Release, reposDir)
	if err != nil {
		return nil, err
	}

	return &RedHatBackend{
		target:     target,
		logger:     logger,
		dnfBackend: b,
	}, nil
}
