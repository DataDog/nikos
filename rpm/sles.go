package rpm

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/DataDog/nikos/rpm/dnf"
	"github.com/DataDog/nikos/types"
)

type SLESBackend struct {
	target     *types.Target
	logger     types.Logger
	dnfBackend *dnf.DnfBackend
}

func (b *SLESBackend) GetKernelHeaders(directory string) error {
	kernelRelease := b.target.Uname.Kernel

	flavour := "-generic"
	flavourIndex := strings.LastIndex(kernelRelease, "-")
	if flavourIndex != -1 {
		flavour = kernelRelease[flavourIndex:]
		kernelRelease = kernelRelease[:flavourIndex]
	}
	pkgNevra := "kernel" + flavour + "-devel-" + kernelRelease

	// On a registered SUSE Entreprise Linux, we should be able to find
	// the kernel headers without doing anything
	b.logger.Info("Trying with the configured set of repositories")
	if err := b.dnfBackend.GetKernelHeaders(pkgNevra, directory); err == nil {
		return nil
	}

	// On not registered systems, we use the repositories from
	// https://download.opensuse.org/repositories/Kernel:
	if version := b.target.OSRelease["VERSION"]; version != "" {
		addKernelRepository := func(version string) {
			version = "SLE" + version
			repoID := "Kernel_" + version
			baseurl := fmt.Sprintf("https://download.opensuse.org/repositories/Kernel:/%s/standard/", version)
			gpgKey := fmt.Sprintf("https://download.opensuse.org/repositories/Kernel:/%s/standard/repodata/repomd.xml.key")

			b.logger.Infof("Using with %s repository", repoID)
			b.dnfBackend.AddRepository(repoID, baseurl, true, gpgKey)
		}

		addKernelRepository(version)
		addKernelRepository(version + "-UPDATES")
		if flavour != "-generic" {
			addKernelRepository(version + strings.ToUpper(flavour))
		}
	}

	// On SLES 15.2 without a subscription, the kernel headers can be found on the 'jump' repository
	if versionID := b.target.OSRelease["VERSION_ID"]; versionID != "" {
		repoID := "Jump-" + versionID
		baseurl := fmt.Sprintf("https://download.opensuse.org/distribution/jump/%s/repo/oss/", versionID)

		b.logger.Infof("Using with %s repository", repoID)
		b.dnfBackend.AddRepository(repoID, baseurl, true, "")
	}

	return b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
}

func NewSLESBackend(target *types.Target, reposDir string, logger types.Logger) (types.Backend, error) {
	dnfBackend, err := dnf.NewDnfBackend(target.Distro.Release, reposDir, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create DNF backend")
	}

	return &SLESBackend{
		target:     target,
		logger:     logger,
		dnfBackend: dnfBackend,
	}, nil
}
