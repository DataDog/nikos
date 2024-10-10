package rpm

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

	flavour := ""
	flavourIndex := strings.LastIndex(kernelRelease, "-")
	if flavourIndex != -1 {
		kernelRelease = b.target.Uname.Kernel[:flavourIndex]
		flavour = strings.TrimLeft(b.target.Uname.Kernel[flavourIndex:], "-")
	}

	pkgKernelDevel := "kernel-devel"
	kernelDevelMatcher := func(pkg *repo.PkgInfoHeader) bool {
		return pkg.Name == pkgKernelDevel && kernelRelease == fmt.Sprintf("%s-%s", pkg.Version.Ver, pkg.Version.Rel)
	}

	b.logger.Debugf("fetching `%s` package", pkgKernelDevel)
	pkg, data, err := b.dnfBackend.FetchPackage(kernelDevelMatcher)
	if err != nil {
		return fmt.Errorf("failed to fetch `%s` package: %w", pkgKernelDevel, err)
	}

	if err := dnfv2.ExtractPackage(pkg, data, directory, b.target, b.logger); err != nil {
		return fmt.Errorf("failed to extract `%s` package: %w", pkgKernelDevel, err)
	}

	kernelDevelBase := filepath.Join(directory, "usr", "src", "linux-"+kernelRelease)
	if flavour != "" {
		pkgFlavourDevel := fmt.Sprintf("kernel-%s-devel", flavour)
		kernelFlavourDevelMatcher := func(pkg *repo.PkgInfoHeader) bool {
			return pkg.Name == pkgFlavourDevel && kernelRelease == fmt.Sprintf("%s-%s", pkg.Version.Ver, pkg.Version.Rel) && pkg.Arch == b.target.Uname.Machine
		}

		b.logger.Debugf("fetching `%s` package", pkgFlavourDevel)
		pkg, data, err := b.dnfBackend.FetchPackage(kernelFlavourDevelMatcher)
		if err != nil {
			return fmt.Errorf("failed to fetch `%s` package: %w", pkgFlavourDevel, err)
		}

		if err := dnfv2.ExtractPackage(pkg, data, directory, b.target, b.logger); err != nil {
			return fmt.Errorf("failed to extract `%s` package: %w", pkgFlavourDevel, err)
		}

		kernelFlavourBase := filepath.Join(
			directory, "usr", "src", fmt.Sprintf("linux-%s-obj", kernelRelease),
			b.target.Uname.Machine, flavour,
		)

		if err := filepath.WalkDir(kernelFlavourBase, func(path string, _ fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			relPath, err := filepath.Rel(kernelFlavourBase, path)
			if err != nil {
				return err
			}
			kernelDevelPath := filepath.Join(kernelDevelBase, relPath)
			if _, err := os.Stat(kernelDevelPath); err != nil && os.IsNotExist(err) {
				if err := os.Symlink(path, kernelDevelPath); err != nil {
					return err
				}
				b.logger.Debugf("created symlink to %s at %s", path, kernelDevelPath)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("failed to walk through %s: %w", kernelFlavourBase, err)
		}
	}

	symlink := filepath.Join(directory, "usr", "src", fmt.Sprintf("linux-headers-%s", b.target.Uname.Kernel))
	if err := os.Symlink(kernelDevelBase, symlink); err != nil {
		return fmt.Errorf("failed to create symlink to %s at %s: %w", kernelDevelBase, symlink, err)
	}
	b.logger.Debugf("created symlink to %s at %s", kernelDevelBase, symlink)

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
