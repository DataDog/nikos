package rpm

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/DataDog/nikos/rpm/dnf"
	"github.com/DataDog/nikos/types"
	"github.com/go-ini/ini"
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
		tryAddKernelRepository := func(version, folder string) {
			version = "SLE" + version
			repoURL := fmt.Sprintf("https://download.opensuse.org/repositories/Kernel:/%s/%s/Kernel:%s.repo", version, folder, version)
			if repo, err := getPotentialRepoFile(repoURL); err == nil {
				b.logger.Infof("Using with %s repository", repo.repoID)
				b.dnfBackend.AddRepository(repo.repoID, repo.baseURL, true, repo.gpgKey, "", "", "")
			}
		}

		for _, folder := range []string{"standard", "pool"} {
			tryAddKernelRepository(version, folder)
			tryAddKernelRepository(version+"-UPDATES", folder)
			if flavour != "-generic" {
				tryAddKernelRepository(version+strings.ToUpper(flavour), folder)
			}
		}
	}

	// On SLES 15.2 without a subscription, the kernel headers can be found on the 'jump' repository
	if versionID := b.target.OSRelease["VERSION_ID"]; versionID != "" {
		repoID := "Jump-" + versionID
		baseurl := fmt.Sprintf("https://download.opensuse.org/distribution/jump/%s/repo/oss/", versionID)

		gpgkeyurl := baseurl + "repodata/repomd.xml.key"
		if _, err := http.Get(gpgkeyurl); err == nil {
			b.logger.Infof("Using with %s repository", repoID)
			b.dnfBackend.AddRepository(repoID, baseurl, true, "", "", "", "")
		}
	}

	return b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
}

func (b *SLESBackend) Close() {
	b.dnfBackend.Close()
}

func NewSLESBackend(target *types.Target, reposDir string, logger types.Logger) (types.Backend, error) {
	dnfBackend, err := dnf.NewDnfBackend(target.Distro.Release, reposDir, logger, target)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNF backend: %w", err)
	}

	return &SLESBackend{
		target:     target,
		logger:     logger,
		dnfBackend: dnfBackend,
	}, nil
}

type repoFileContent struct {
	repoID  string
	baseURL string
	gpgKey  string
}

func getPotentialRepoFile(url string) (*repoFileContent, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	repoCfg, err := ini.Load(body)
	if err != nil {
		return nil, err
	}

	for _, section := range repoCfg.Sections() {
		if section.Key("type").String() == "rpm-md" && section.HasKey("baseurl") && section.HasKey("gpgkey") {
			return &repoFileContent{
				repoID:  section.Name(),
				baseURL: section.Key("baseurl").String(),
				gpgKey:  section.Key("gpgkey").String(),
			}, nil
		}
	}
	return nil, errors.New("no valid repo found at this URL")
}
