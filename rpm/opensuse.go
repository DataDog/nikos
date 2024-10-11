package rpm

import (
	"fmt"
	"strings"

	"github.com/DataDog/nikos/rpm/dnfv2"
	"github.com/DataDog/nikos/rpm/dnfv2/backend"
	"github.com/DataDog/nikos/rpm/dnfv2/repo"
	"github.com/DataDog/nikos/types"
)

type OpenSUSEBackend struct {
	target     *types.Target
	logger     types.Logger
	dnfBackend *backend.Backend
}

func (b *OpenSUSEBackend) GetKernelHeaders(directory string) error {
	kernelRelease := b.target.Uname.Kernel

	pkgNevra := "kernel"
	flavourIndex := strings.LastIndex(kernelRelease, "-")
	if flavourIndex != -1 {
		pkgNevra += kernelRelease[flavourIndex:]
		kernelRelease = kernelRelease[:flavourIndex]
	}
	pkgNevra += "-devel"

	packagesToInstall := []string{pkgNevra}
	if pkgNevra != "kernel-devel" {
		packagesToInstall = append(packagesToInstall, "kernel-devel")
	}

	installedPackages := 0
	for _, targetPackageName := range packagesToInstall {
		pkgMatcher := func(pkg *repo.PkgInfoHeader) bool {
			return pkg.Name == targetPackageName &&
				kernelRelease == fmt.Sprintf("%s-%s", pkg.Version.Ver, pkg.Version.Rel) &&
				(pkg.Arch == b.target.Uname.Machine || pkg.Arch == "noarch")
		}

		pkg, data, err := b.dnfBackend.FetchPackage(pkgMatcher)
		if err != nil {
			b.logger.Errorf("failed to fetch `%s` package: %v", targetPackageName, err)
			continue
		}

		if err := dnfv2.ExtractPackage(pkg, data, directory, b.target, b.logger); err != nil {
			b.logger.Errorf("failed to extract `%s` package: %v", targetPackageName, err)
			continue
		}

		installedPackages++
	}

	if installedPackages == 0 {
		return fmt.Errorf("failed to find a valid package")
	}

	return nil
}

func (b *OpenSUSEBackend) Close() {
}

func NewOpenSUSEBackend(target *types.Target, reposDir string, logger types.Logger) (types.Backend, error) {
	b, err := dnfv2.NewBackend(target.Distro.Release, reposDir)
	if err != nil {
		return nil, err
	}

	return &OpenSUSEBackend{
		target:     target,
		logger:     logger,
		dnfBackend: b,
	}, nil
}
