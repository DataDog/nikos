package rpm

import (
	"fmt"

	"github.com/DataDog/nikos/rpm/dnf"
	"github.com/DataDog/nikos/types"
)

const (
	updatesRepoBaseURL = "https://fedoraproject-updates-archive.fedoraproject.org/fedora/$releasever/$basearch/"
)

type FedoraBackend struct {
	dnfBackend *dnf.DnfBackend
	logger     types.Logger
	target     *types.Target
}

func (b *FedoraBackend) GetKernelHeaders(directory string) error {
	for _, repo := range b.dnfBackend.GetEnabledRepositories() {
		if repo.Id != "base" && repo.Id != "updates" {
			b.dnfBackend.DisableRepository(repo)
			continue
		}
	}

	// First, check for the correct kernel-headers package
	pkgNevra := "kernel-headers-" + b.target.Uname.Kernel
	fmt.Printf("Repositories %+v\n", b.dnfBackend.GetEnabledRepositories())
	err := b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
	if err == nil {
		return nil
	}

	// If that doesn't work, try again with the updates-archive repo
	updatesRepoGPGKey := "file:///" + types.HostEtc("pki/rpm-gpg/RPM-GPG-KEY-fedora-$releasever-$basearch")
	b.logger.Infof("Trying with updates-archive repository")
	if _, err := b.dnfBackend.AddRepository("updates-archive", updatesRepoBaseURL, true, updatesRepoGPGKey, "", "", ""); err == nil {
		err = b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
		if err == nil {
			return nil
		}
	} else {
		b.logger.Warnf("Failed to add updates-archive repository: %w", err)
	}

	// As a last resort, check for the kernel-devel package
	pkgNevra = "kernel-devel-" + b.target.Uname.Kernel
	return b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
}

func (b *FedoraBackend) Close() {
	b.dnfBackend.Close()
}

func NewFedoraBackend(target *types.Target, reposDir string, logger types.Logger) (*FedoraBackend, error) {
	dnfBackend, err := dnf.NewDnfBackend(target.Distro.Release, reposDir, logger, target)
	if err != nil {
		return nil, fmt.Errorf("failed to create fedora dnf backend: %w", err)
	}

	return &FedoraBackend{
		target:     target,
		logger:     logger,
		dnfBackend: dnfBackend,
	}, nil
}