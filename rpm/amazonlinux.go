package rpm

import (
	"github.com/DataDog/nikos/types"

	"github.com/paulcacheux/did-not-finish/backend"
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
