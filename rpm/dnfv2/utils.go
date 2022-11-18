package dnfv2

import (
	"fmt"
	"os"
	"path"

	"github.com/DataDog/nikos/extract"
	"github.com/DataDog/nikos/types"
	"github.com/paulcacheux/did-not-finish/backend"
	"github.com/paulcacheux/did-not-finish/repo"
	dnfTypes "github.com/paulcacheux/did-not-finish/types"
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

func computePkgKernel(pkg *dnfTypes.Package) string {
	return fmt.Sprintf("%s-%s.%s", pkg.Version.Ver, pkg.Version.Rel, pkg.Arch)
}

func DefaultPkgMatcher(pkgName string, target *types.Target) repo.PkgMatchFunc {
	return func(pkg *dnfTypes.Package) bool {
		return pkg.Name == pkgName && target.Uname.Kernel == computePkgKernel(pkg)
	}
}

func ExtractPackage(pkg *dnfTypes.Package, data []byte, directory string, target *types.Target, logger types.Logger) error {
	pkgFileName := fmt.Sprintf("%s-%s.rpm", pkg.Name, computePkgKernel(pkg))
	pkgFileName = path.Join(directory, pkgFileName)
	if err := os.WriteFile(pkgFileName, data, 0o644); err != nil {
		return err
	}

	return extract.ExtractRPMPackage(pkgFileName, directory, target.Uname.Kernel, logger)
}
