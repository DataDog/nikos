package rpm

import (
	"bytes"
	"fmt"

	"github.com/DataDog/nikos/extract"
	"github.com/DataDog/nikos/types"
	"github.com/paulcacheux/did-not-finish/backend"
	"github.com/paulcacheux/did-not-finish/repo"
	dnfTypes "github.com/paulcacheux/did-not-finish/types"
)

type FedoraBackend struct {
	dnfBackend *backend.Backend
	logger     types.Logger
	target     *types.Target
}

func (b *FedoraBackend) GetKernelHeaders(directory string) error {
	for _, targetPackageName := range []string{"kernel-headers", "kernel-devel"} {
		pkgMatcher := func(pkg *dnfTypes.Package) bool {
			pkgKernel := fmt.Sprintf("%s-%s.%s", pkg.Version.Ver, pkg.Version.Rel, pkg.Arch)
			return pkg.Name == targetPackageName && b.target.Uname.Kernel == pkgKernel
		}

		pkg, data, err := b.dnfBackend.FetchPackage(pkgMatcher)
		if err != nil {
			b.logger.Errorf("failed to fetch `%s` package: %v", targetPackageName, err)
			continue
		}
		return extract.ExtractRPMPackageFromReader(bytes.NewReader(data), pkg.Name, directory, b.target.Uname.Kernel, b.logger)
	}

	return fmt.Errorf("failed to find a valid package")
}

func (b *FedoraBackend) Close() {
}

func NewFedoraBackend(target *types.Target, reposDir string, logger types.Logger) (*FedoraBackend, error) {
	builtinVars, err := backend.ComputeBuiltinVariables(target.Distro.Release)
	if err != nil {
		return nil, fmt.Errorf("failed to compute DNF builting variables: %w", err)
	}

	varsDir := []string{"/etc/dnf/vars/", "/etc/yum/vars/"}
	b, err := backend.NewBackend(reposDir, varsDir, builtinVars)
	if err != nil {
		return nil, fmt.Errorf("failed to create fedora dnf backend: %w", err)
	}

	const (
		updatesArchiveRepoBaseURL = "https://fedoraproject-updates-archive.fedoraproject.org/fedora/$releasever/$basearch/"
		updatesArchiveGpgKeyPath  = "file:///etc/pki/rpm-gpg/RPM-GPG-KEY-fedora-$releasever-$basearch"
	)

	// updates archive as a fallback
	b.AppendRepository(repo.Repo{
		SectionName: "",
		Name:        "updates-archive",
		BaseURL:     updatesArchiveRepoBaseURL,
		Enabled:     true,
		GpgCheck:    true,
		GpgKey:      updatesArchiveGpgKeyPath,
	})

	return &FedoraBackend{
		target:     target,
		logger:     logger,
		dnfBackend: b,
	}, nil
}
