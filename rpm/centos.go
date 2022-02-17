package rpm

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	"github.com/DataDog/nikos/rpm/dnf"
	"github.com/DataDog/nikos/types"
)

type CentOSBackend struct {
	version    int
	release    string
	dnfBackend *dnf.DnfBackend
	target     *types.Target
	logger     types.Logger
}

func getRedhatRelease() (string, error) {
	redhatReleasePath := types.HostEtc("redhat-release")
	redhatRelease, err := ioutil.ReadFile(redhatReleasePath)
	if err != nil {
		return "", fmt.Errorf("failed to read /etc/redhat-release: %w", err)
	}

	re := regexp.MustCompile(`.* release ([0-9\.]*)`)
	submatches := re.FindStringSubmatch(string(redhatRelease))
	if len(submatches) == 2 {
		return submatches[1], nil
	}

	return "", fmt.Errorf("failed to parse release from %s", redhatReleasePath)
}

func (b *CentOSBackend) GetKernelHeaders(directory string) error {
	pkgNevra := "kernel-devel-" + b.target.Uname.Kernel

	// First try with the 'base' and 'updates' repositories.
	// This should work if the user is using the latest minor version
	b.logger.Info("Trying with 'base' and 'updates' repositories")

	for _, repo := range b.dnfBackend.GetEnabledRepositories() {
		if repo.Id != "base" && repo.Id != "updates" {
			b.dnfBackend.DisableRepository(repo)
		}
	}

	if b.dnfBackend.GetKernelHeaders(pkgNevra, directory) == nil {
		return nil
	}

	// Otherwise, we try with Vault
	b.logger.Infof("Trying with Vault repositories for %s", b.release)

	var baseURL string
	gpgKey := "file:///" + types.HostEtc("pki/rpm-gpg/RPM-GPG-KEY-")
	if b.version >= 8 {
		gpgKey += "centosofficial" // gpg key name convention changed in centos8
		baseURL = fmt.Sprintf("http://vault.centos.org/%s/BaseOS/$basearch/os/", b.release)
	} else {
		gpgKey += "CentOS-" + strconv.Itoa(b.version)
		baseURL = fmt.Sprintf("http://vault.centos.org/%s/os/$basearch/", b.release)

		updatesURL := fmt.Sprintf("http://vault.centos.org/%s/updates/$basearch/", b.release)
		if _, err := b.dnfBackend.AddRepository("C"+b.release+"-updates", updatesURL, true, gpgKey, "", "", ""); err != nil {
			return fmt.Errorf("failed to add Vault updates repository: %w", err)
		}
	}

	if _, err := b.dnfBackend.AddRepository("C"+b.release+"-base", baseURL, true, gpgKey, "", "", ""); err != nil {
		return fmt.Errorf("failed to add Vault base repository: %w", err)
	}

	return b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
}

func (b *CentOSBackend) Close() {
	b.dnfBackend.Close()
}

func NewCentOSBackend(target *types.Target, reposDir string, logger types.Logger) (*CentOSBackend, error) {
	release, err := getRedhatRelease()
	if err != nil {
		return nil, fmt.Errorf("failed to detect CentOS release: %w", err)
	}

	version, _ := strconv.Atoi(strings.SplitN(release, ".", 2)[0])
	dnfBackend, err := dnf.NewDnfBackend(fmt.Sprintf("%d", version), reposDir, logger, target)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNF backend: %w", err)
	}

	return &CentOSBackend{
		version:    version,
		target:     target,
		logger:     logger,
		release:    release,
		dnfBackend: dnfBackend,
	}, nil
}
