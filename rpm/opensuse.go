package rpm

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/DataDog/nikos/rpm/dnf"
	"github.com/DataDog/nikos/types"
)

type OpenSUSEBackend struct {
	target     *types.Target
	logger     types.Logger
	dnfBackend *dnf.DnfBackend
}

func (b *OpenSUSEBackend) GetKernelHeaders(directory string) error {
	kernelRelease := b.target.Uname.Kernel

	pkgNevra := "kernel"
	flavourIndex := strings.LastIndex(kernelRelease, "-")
	if flavourIndex != -1 {
		pkgNevra += kernelRelease[flavourIndex:]
		kernelRelease = kernelRelease[:flavourIndex]
	}
	pkgNevra += "-devel-" + kernelRelease

	var disabledRepositories []*dnf.Repository
	ossRepoRgx := regexp.MustCompile(`openSUSE-Leap-\d+.\d+-Oss`)
	updateRepoRgx := regexp.MustCompile(`openSUSE-Leap-\d+.\d+-Update`)

	// For OpenSUSE Leap, we first try with only the repo-oss and repo-update repositories
	b.logger.Infof("Trying with Oss & Update repositories")
	for _, repo := range b.dnfBackend.GetEnabledRepositories() {
		if repo.Id != "repo-oss" && repo.Id != "repo-update" &&
			!ossRepoRgx.MatchString(repo.Id) && !updateRepoRgx.MatchString(repo.Id) {
			b.dnfBackend.DisableRepository(repo)
			disabledRepositories = append(disabledRepositories, repo)
		}
	}
	err := b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
	if err == nil {
		return nil
	}

	b.logger.Infof("Error downloading package: %s. Retrying with the full set of repositories", err)
	for _, repo := range disabledRepositories {
		b.dnfBackend.EnableRepository(repo)
	}
	return b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
}

func NewOpenSUSEBackend(target *types.Target, reposDir string, logger types.Logger) (types.Backend, error) {
	dnfBackend, err := dnf.NewDnfBackend(target.Distro.Release, reposDir, logger, target)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNF backend: %w", err)
	}

	return &OpenSUSEBackend{
		target:     target,
		logger:     logger,
		dnfBackend: dnfBackend,
	}, nil
}
