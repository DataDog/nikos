package rpm

import (
	"bytes"
	"fmt"

	"github.com/DataDog/nikos/extract"
	"github.com/DataDog/nikos/types"
	"github.com/paulcacheux/did-not-finish/backend"
	dnfTypes "github.com/paulcacheux/did-not-finish/types"
)

type AmazonLinux2022Backend struct {
	dnfBackend *backend.Backend
	target     *types.Target
	logger     types.Logger
}

func NewAmazonLinux2022Backend(target *types.Target, reposDir string, logger types.Logger) (*AmazonLinux2022Backend, error) {
	builtinVars, err := backend.ComputeBuiltinVariables()
	if err != nil {
		return nil, err
	}

	varsDir := []string{"/etc/dnf/vars/", "/etc/yum/vars/"}
	backend, err := backend.NewBackend(reposDir, varsDir, builtinVars)
	if err != nil {
		return nil, err
	}

	return &AmazonLinux2022Backend{
		dnfBackend: backend,
		target:     target,
		logger:     logger,
	}, nil
}

func (b *AmazonLinux2022Backend) GetKernelHeaders(directory string) error {
	pkgMatcher := func(pkg *dnfTypes.Package) bool {
		return pkg.Name == "kernel-header" && b.target.Uname.Machine == pkg.Version.Ver+pkg.Version.Rel
	}

	targetPackageName := "kernel-headers"

	for _, repository := range b.dnfBackend.Repositories {
		if !repository.Enabled {
			continue
		}

		pkg, data, err := repository.FetchPackage(pkgMatcher)
		if err != nil {
			return err
		}
		if pkg == nil {
			continue
		}

		return extract.ExtractRPMPackageFromReader(bytes.NewReader(data), pkg.Name, directory, b.target.Uname.Kernel, b.logger)
	}

	return fmt.Errorf("failed to find package %s", targetPackageName)
}

func (b *AmazonLinux2022Backend) Close() {
	// do nothing
}
