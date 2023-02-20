package dnfv2

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/DataDog/nikos/extract"
	"github.com/DataDog/nikos/rpm/dnfv2/backend"
	"github.com/DataDog/nikos/rpm/dnfv2/repo"
	"github.com/DataDog/nikos/types"
)

func NewBackend(release string, reposDir string) (*backend.Backend, error) {
	builtinVars, err := backend.ComputeBuiltinVariables(release)
	if err != nil {
		return nil, fmt.Errorf("failed to compute DNF builting variables: %w", err)
	}

	varsDir := []string{"/etc/dnf/vars/", "/etc/yum/vars/"}
	b, err := backend.NewBackend(reposDir, varsDir, builtinVars)
	if err != nil {
		return nil, fmt.Errorf("failed to create fedora dnf backend: %w", err)
	}

	return b, nil
}

func computePkgKernel(pkg *repo.PkgInfo) string {
	return fmt.Sprintf("%s-%s.%s", pkg.Version.Ver, pkg.Version.Rel, pkg.Arch)
}

func DefaultPkgMatcher(pkgName, kernelVersion string) repo.PkgMatchFunc {
	return func(pkg *repo.PkgInfo) bool {
		if strings.Contains(pkg.Name, "kernel") {
			fmt.Println(pkg, kernelVersion, computePkgKernel(pkg))
		}
		return pkg.Name == pkgName && kernelVersion == computePkgKernel(pkg)
	}
}

func ExtractPackage(pkg *repo.PkgInfo, data []byte, directory string, target *types.Target, logger types.Logger) error {
	pkgFileName := fmt.Sprintf("%s-%s.rpm", pkg.Name, computePkgKernel(pkg))
	pkgFileName = path.Join(directory, pkgFileName)
	if err := os.WriteFile(pkgFileName, data, 0o644); err != nil {
		return err
	}

	return extract.ExtractRPMPackage(pkgFileName, directory, target.Uname.Kernel, logger)
}
