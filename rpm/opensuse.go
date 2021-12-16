package rpm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DataDog/nikos/rpm/dnf"
	"github.com/DataDog/nikos/types"
	"github.com/DataDog/nikos/utils"
	"github.com/go-ini/ini"
)

type OpenSUSEBackend struct {
	target     *types.Target
	logger     types.Logger
	dnfBackend *dnf.DnfBackend
}

func (b *OpenSUSEBackend) GetKernelHeaders(directory string) error {
	kernelRelease := b.target.Uname.Kernel

	pkgNevra := "kernel"
	flavourIndex := strings.LastIndex(kernelRelease, "-")
	if flavourIndex != -1 {
		pkgNevra += kernelRelease[flavourIndex:]
		kernelRelease = kernelRelease[:flavourIndex]
	}
	pkgNevra += "-devel-" + kernelRelease

	var disabledRepositories []*dnf.Repository
	ossRepoRgx := regexp.MustCompile(`openSUSE-Leap-\d+.\d+-Oss`)
	updateRepoRgx := regexp.MustCompile(`openSUSE-Leap-\d+.\d+-Update`)

	// For OpenSUSE Leap, we first try with only the repo-oss and repo-update repositories
	b.logger.Infof("Trying with Oss & Update repositories")
	for _, repo := range b.dnfBackend.GetEnabledRepositories() {
		if repo.Id != "repo-oss" && repo.Id != "repo-update" &&
			!ossRepoRgx.MatchString(repo.Id) && !updateRepoRgx.MatchString(repo.Id) {
			b.dnfBackend.DisableRepository(repo)
			disabledRepositories = append(disabledRepositories, repo)
		}
	}
	err := b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
	if err == nil {
		b.logger.Infof("successfully downloaded %s", pkgNevra)

		// kernel-[flavour]-devel packages require kernel-devel
		return b.dnfBackend.GetKernelHeaders("kernel-devel-"+kernelRelease, directory)
	}

	b.logger.Infof("Error downloading package: %s. Retrying with the full set of repositories", err)
	for _, repo := range disabledRepositories {
		b.dnfBackend.EnableRepository(repo)
	}
	err = b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
	if err == nil {
		b.logger.Infof("successfully downloaded %s", pkgNevra)
		return b.dnfBackend.GetKernelHeaders("kernel-devel-"+kernelRelease, directory)
	}
	return err
}

// On newer versions of openSUSE, the repos are type yast2 instead of type yum.
// For now, librepo only supports yum, so we need to convert the yast2 repo baseurls
// to 'yum' format. Without this, attempting to download the repomd.xml file
// results in a 404 error.
func yumifyRepositories(reposDir string, logger types.Logger) (string, error) {
	tmpDir, err := ioutil.TempDir("", "yum.repos.d")
	if err != nil {
		return "", err
	}

	repoFiles, err := filepath.Glob(reposDir + "/*.repo")
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	for _, repoFile := range repoFiles {
		destFilename := filepath.Join(tmpDir, filepath.Base(repoFile))

		logger.Debugf("Reading repo file '%s'", repoFile)
		cfg, err := ini.Load(repoFile)
		if err != nil {
			logger.Warnf("Failed to read file '%s': %v", repoFile, err)

			err = utils.CopyFile(repoFile, destFilename)
			if err != nil {
				logger.Warnf("Failed to copy %s to tmp dir: %v", repoFile, err)
			}
			continue
		}

		if isYast2(cfg) {
			sections := cfg.Sections()
			for _, section := range sections {
				if section.HasKey("baseurl") {
					baseurl := section.Key("baseurl").String()
					section.Key("baseurl").SetValue(baseurl + "suse/")
				}
			}
		}

		if err := cfg.SaveTo(destFilename); err != nil {
			logger.Warnf("Failed to write file '%s': %v", destFilename, err)

			err = utils.CopyFile(repoFile, destFilename)
			if err != nil {
				logger.Warnf("Failed to copy %s to tmp dir: %v", repoFile, err)
			}
		}
	}

	return tmpDir, nil
}

func isYast2(repoFile *ini.File) bool {
	sections := repoFile.Sections()
	for _, section := range sections {
		if section.HasKey("type") {
			return section.Key("type").String() == "yast2"
		}
	}
	return false
}

func (b *OpenSUSEBackend) Close() {
	b.dnfBackend.Close()
}

func NewOpenSUSEBackend(target *types.Target, reposDir string, logger types.Logger) (types.Backend, error) {
	tmpReposDir, err := yumifyRepositories(reposDir, logger)
	if err != nil {
		logger.Warnf("Fail to convert yast2 repos to yum repos: %v", err)
	} else {
		reposDir = tmpReposDir
		//defer os.RemoveAll(tmpReposDir)
	}

	dnfBackend, err := dnf.NewDnfBackend(target.Distro.Release, reposDir, logger, target)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNF backend: %w", err)
	}

	return &OpenSUSEBackend{
		target:     target,
		logger:     logger,
		dnfBackend: dnfBackend,
	}, nil
}
