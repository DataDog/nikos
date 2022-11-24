package rpm

import (
	"fmt"

	"github.com/DataDog/nikos/rpm/dnfv2"
	"github.com/DataDog/nikos/rpm/dnfv2/backend"
	"github.com/DataDog/nikos/rpm/dnfv2/repo"
	"github.com/DataDog/nikos/types"
)

type FedoraBackend struct {
	dnfBackend *backend.Backend
	logger     types.Logger
	target     *types.Target
}

func (b *FedoraBackend) GetKernelHeaders(directory string) error {
	for _, targetPackageName := range []string{"kernel-devel", "kernel-headers"} {
		pkgMatcher := dnfv2.DefaultPkgMatcher(targetPackageName, b.target.Uname.Kernel)

		pkg, data, err := b.dnfBackend.FetchPackage(pkgMatcher)
		if err != nil {
			b.logger.Errorf("failed to fetch `%s` package: %v", targetPackageName, err)
			continue
		}

		return dnfv2.ExtractPackage(pkg, data, directory, b.target, b.logger)
	}

	return fmt.Errorf("failed to find a valid package")
}

func (b *FedoraBackend) Close() {
}

func NewFedoraBackend(target *types.Target, reposDir string, logger types.Logger) (*FedoraBackend, error) {
	b, err := dnfv2.NewBackend(target.Distro.Release, reposDir)
	if err != nil {
		return nil, err
	}

	const (
		updatesArchiveRepoBaseURL = "https://fedoraproject-updates-archive.fedoraproject.org/fedora/$releasever/$basearch/"
		updatesArchiveGpgKeyPath  = "file:///etc/pki/rpm-gpg/RPM-GPG-KEY-fedora-$releasever-$basearch"
	)

	// updates archive as a fallback
	b.AppendRepository(repo.Repo{
		Name:     "updates-archive",
		BaseURL:  updatesArchiveRepoBaseURL,
		Enabled:  true,
		GpgCheck: true,
		GpgKey:   updatesArchiveGpgKeyPath,
	})

	return &FedoraBackend{
		target:     target,
		logger:     logger,
		dnfBackend: b,
	}, nil
}
