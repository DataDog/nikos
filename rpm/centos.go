package rpm

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	"github.com/DataDog/nikos/rpm/dnfv2"
	"github.com/DataDog/nikos/rpm/dnfv2/backend"
	"github.com/DataDog/nikos/rpm/dnfv2/repo"
	"github.com/DataDog/nikos/types"
)

type CentOSBackend struct {
	dnfBackend *backend.Backend
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
	pkgNevra := "kernel-devel"
	pkgMatcher := dnfv2.DefaultPkgMatcher(pkgNevra, b.target.Uname.Kernel)

	pkg, data, err := b.dnfBackend.FetchPackage(pkgMatcher)
	if err != nil {
		return fmt.Errorf("failed to fetch `%s` package: %w", pkgNevra, err)
	}

	return dnfv2.ExtractPackage(pkg, data, directory, b.target, b.logger)
}

func (b *CentOSBackend) Close() {
}

func NewCentOSBackend(target *types.Target, reposDir string, logger types.Logger) (*CentOSBackend, error) {
	release, err := getRedhatRelease()
	if err != nil {
		return nil, fmt.Errorf("failed to detect CentOS release: %w", err)
	}

	version, _ := strconv.Atoi(strings.SplitN(release, ".", 2)[0])
	versionStr := fmt.Sprintf("%d", version)

	b, err := dnfv2.NewBackend(versionStr, reposDir, logger)
	if err != nil {
		return nil, err
	}

	if version >= 8 {
		gpgKey := "file:///etc/pki/rpm-gpg/RPM-GPG-KEY-centosofficial"
		baseURL := fmt.Sprintf("http://vault.centos.org/%s/BaseOS/$basearch/os/", release)
		b.AppendRepository(repo.Repo{
			Name:     fmt.Sprintf("C%s-base", release),
			BaseURL:  baseURL,
			Enabled:  true,
			GpgCheck: true,
			GpgKey:   gpgKey,
		})
	} else {
		gpgKey := fmt.Sprintf("file:///etc/pki/rpm-gpg/RPM-GPG-KEY-CentOS-%d", version)
		baseURL := fmt.Sprintf("http://vault.centos.org/%s/os/$basearch/", release)
		updatesURL := fmt.Sprintf("http://vault.centos.org/%s/updates/$basearch/", release)
		b.AppendRepository(repo.Repo{
			Name:     fmt.Sprintf("C%s-base", release),
			BaseURL:  baseURL,
			Enabled:  true,
			GpgCheck: true,
			GpgKey:   gpgKey,
		})
		b.AppendRepository(repo.Repo{
			Name:     fmt.Sprintf("C%s-updates", release),
			BaseURL:  updatesURL,
			Enabled:  true,
			GpgCheck: true,
			GpgKey:   gpgKey,
		})
	}

	return &CentOSBackend{
		target:     target,
		logger:     logger,
		dnfBackend: b,
	}, nil
}
