package rpm

import (
	"fmt"
	"regexp"

	"github.com/DataDog/nikos/rpm/dnfv2"
	"github.com/DataDog/nikos/rpm/dnfv2/backend"
	"github.com/DataDog/nikos/types"
)

type OracleBackend struct {
	dnfBackend *backend.Backend
	logger     types.Logger
	target     *types.Target
}

func (b *OracleBackend) GetKernelHeaders(directory string) error {
	for _, targetPackageName := range []string{"kernel-devel", "kernel-uek-devel"} {
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

func (b *OracleBackend) Close() {
}

func NewOracleBackend(target *types.Target, reposDir string, logger types.Logger) (*RedHatBackend, error) {
	b, err := dnfv2.NewBackend(target.Distro.Release, reposDir)
	if err != nil {
		return nil, err
	}

	uekRepoPattern := regexp.MustCompile(`^ol\d_UEK.*`)

	// force enable UEK repos
	for i := range b.Repositories {
		repo := &b.Repositories[i]

		if uekRepoPattern.MatchString(repo.Name) {
			repo.Enabled = true
		}
	}

	return &RedHatBackend{
		target:     target,
		logger:     logger,
		dnfBackend: b,
	}, nil
}
