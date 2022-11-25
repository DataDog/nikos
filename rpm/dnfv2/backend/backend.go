package backend

import (
	"crypto/tls"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/DataDog/nikos/rpm/dnfv2/internal/utils"
	"github.com/DataDog/nikos/rpm/dnfv2/repo"
	"github.com/hashicorp/go-multierror"
)

type Backend struct {
	client *http.Client

	Repositories []repo.Repo
	varsReplacer *strings.Replacer
}

func hostRootsHttpClient() *http.Client {
	roots, err := GetSystemRoots()
	if err != nil {
		return &http.Client{}
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: roots,
		},
	}
	return &http.Client{Transport: tr}
}

func NewBackend(reposDir string, varsDir []string, builtinVariables map[string]string) (*Backend, error) {
	client := hostRootsHttpClient()

	varMaps := []map[string]string{builtinVariables}
	for _, varDir := range varsDir {
		if varDir == "" {
			continue
		}

		vars, err := readVars(varDir)
		if err != nil {
			continue
		}

		if len(vars) != 0 {
			varMaps = append(varMaps, vars)
		}
	}

	varsReplacer := buildVarsReplacer(varMaps...)

	repos, err := repo.ReadFromDir(reposDir)
	if err != nil {
		return nil, err
	}

	replacedRepos := make([]repo.Repo, 0, len(repos))
	for _, r := range repos {
		replacedRepos = append(replacedRepos, replaceInRepo(varsReplacer, r))
	}

	return &Backend{
		client:       client,
		Repositories: replacedRepos,
		varsReplacer: varsReplacer,
	}, nil
}

func replaceInRepo(varsReplacer *strings.Replacer, r repo.Repo) repo.Repo {
	r.Name = varsReplacer.Replace(r.Name)
	r.BaseURL = varsReplacer.Replace(r.BaseURL)
	r.MirrorList = varsReplacer.Replace(r.MirrorList)
	r.MetaLink = varsReplacer.Replace(r.MetaLink)
	r.GpgKey = varsReplacer.Replace(r.GpgKey)
	return r
}

func (b *Backend) AppendRepository(r repo.Repo) {
	b.Repositories = append(b.Repositories, replaceInRepo(b.varsReplacer, r))
}

func (b *Backend) FetchPackage(matcher repo.PkgMatchFunc) (*repo.PkgInfo, []byte, error) {
	var mErr error

	for _, repository := range b.Repositories {
		if !repository.Enabled {
			continue
		}

		p, content, err := repository.FetchPackage(b.client, matcher)
		if err != nil {
			mErr = multierror.Append(mErr, err)
			continue
		}
		return p, content, nil
	}

	if mErr == nil {
		return nil, nil, errors.New("no repository available")
	}
	return nil, nil, mErr
}

func readVars(varsDir string) (map[string]string, error) {
	varsFile, err := os.ReadDir(utils.HostEtcJoin(varsDir))
	if err != nil {
		return nil, err
	}

	vars := make(map[string]string)
	for _, f := range varsFile {
		if f.IsDir() {
			continue
		}

		varName := f.Name()
		value, err := os.ReadFile(utils.HostEtcJoin(varsDir, varName))
		if err != nil {
			return nil, err
		}

		vars[varName] = strings.TrimSpace(string(value))
	}
	return vars, nil
}

func buildVarsReplacer(varMaps ...map[string]string) *strings.Replacer {
	count := 0
	for _, varMap := range varMaps {
		count += len(varMap)
	}

	pairs := make([]string, 0, count*2)
	for _, varMap := range varMaps {
		for name, value := range varMap {
			pairs = append(pairs, "$"+name, value)
		}
	}

	return strings.NewReplacer(pairs...)
}
