package rpm

import (
	"fmt"
	"strings"

	"github.com/DataDog/nikos/rpm/dnfv2"
	"github.com/DataDog/nikos/rpm/dnfv2/backend"
	"github.com/DataDog/nikos/rpm/dnfv2/repo"
	"github.com/DataDog/nikos/types"
)

type SLESBackend struct {
	target        *types.Target
	flavour       string
	kernelRelease string
	logger        types.Logger
	dnfBackend    *backend.Backend
}

func (b *SLESBackend) GetKernelHeaders(directory string) error {
	pkgNevra := "kernel" + b.flavour + "-devel"
	packagesToInstall := []string{pkgNevra, "kernel-devel"}

	installedPackages := 0
	for _, targetPackageName := range packagesToInstall {
		pkgMatcher := func(pkg *repo.PkgInfoHeader) bool {
			return pkg.Name == targetPackageName &&
				b.kernelRelease == fmt.Sprintf("%s-%s", pkg.Version.Ver, pkg.Version.Rel) &&
				(pkg.Arch == b.target.Uname.Machine || pkg.Arch == "noarch")
		}
		pkg, data, err := b.dnfBackend.FetchPackage(pkgMatcher)
		if err != nil {
			return fmt.Errorf("failed to fetch `%s` package: %w", pkgNevra, err)
		}

		if err := dnfv2.ExtractPackage(pkg, data, directory, b.target, b.logger); err != nil {
			return fmt.Errorf("failed to extract `%s` package: %w", pkgNevra, err)
		}

		installedPackages++
	}

	if installedPackages == 0 {
		return fmt.Errorf("failed to find a valid package")
	}

	return nil
}

func (b *SLESBackend) Close() {
}

func NewSLESBackend(target *types.Target, reposDir string, logger types.Logger) (types.Backend, error) {
	b, err := dnfv2.NewBackend(target.Distro.Release, reposDir)
	if err != nil {
		return nil, err
	}

	kernelRelease := target.Uname.Kernel
	flavour := "-generic"
	flavourIndex := strings.LastIndex(kernelRelease, "-")
	if flavourIndex != -1 {
		flavour = kernelRelease[flavourIndex:]
		kernelRelease = kernelRelease[:flavourIndex]
	}

	// On not registered systems, we use the repositories from
	// https://download.opensuse.org/repositories/Kernel:
	if version := target.OSRelease["VERSION"]; version != "" {
		addKernelRepository := func(version string) {
			versionSplit := strings.SplitN(version, "-", 2)
			version = "SLE" + version
			repoID := "Kernel_" + version
			subPath := "standard"
			if versionSplit[0] == "15" {
				subPath = "pool"
			}
			baseurl := fmt.Sprintf("https://download.opensuse.org/repositories/Kernel:/%s/%s/", version, subPath)
			gpgKey := fmt.Sprintf("https://download.opensuse.org/repositories/Kernel:/%s/%s/repodata/repomd.xml.key", version, subPath)

			b.AppendRepository(repo.Repo{
				Name:     repoID,
				BaseURL:  baseurl,
				Enabled:  true,
				GpgCheck: true,
				GpgKeys:  []string{gpgKey},
			})
		}

		addKernelRepository(version)
		addKernelRepository(version + "-UPDATES")
		if flavour != "-generic" {
			addKernelRepository(version + strings.ToUpper(flavour))
		}
	}

	// On SLES 15.2 without a subscription, the kernel headers can be found on the 'jump' repository
	if versionID := target.OSRelease["VERSION_ID"]; versionID != "" {
		repoID := "Jump-" + versionID
		baseurl := fmt.Sprintf("https://download.opensuse.org/distribution/jump/%s/repo/oss/", versionID)

		b.AppendRepository(repo.Repo{
			Name:    repoID,
			BaseURL: baseurl,
			Enabled: true,
		})
	}

	return &SLESBackend{
		target:        target,
		flavour:       flavour,
		kernelRelease: kernelRelease,
		logger:        logger,
		dnfBackend:    b,
	}, nil
}
