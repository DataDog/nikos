package rpm

import (
	"fmt"

	"github.com/DataDog/nikos/rpm/dnf"
	"github.com/DataDog/nikos/types"
)

type AmazonLinux2Backend struct {
	dnfBackend *dnf.DnfBackend
	logger     types.Logger
	target     *types.Target
}

func (b *AmazonLinux2Backend) Close() {
	b.dnfBackend.Close()
}

func (b *AmazonLinux2Backend) GetKernelHeaders(directory string) error {
	pkgNevra := "kernel-devel-" + b.target.Uname.Kernel
	return b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
}

func NewAmazonLinux2Backend(target *types.Target, reposDir string, logger types.Logger) (*AmazonLinux2Backend, error) {
	dnfBackend, err := dnf.NewDnfBackend(target.Distro.Release, reposDir, logger, target)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNF backend: %w", err)
	}

	return &AmazonLinux2Backend{
		target:     target,
		logger:     logger,
		dnfBackend: dnfBackend,
	}, nil
}
