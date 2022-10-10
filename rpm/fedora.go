package rpm

import (
	"fmt"

	"github.com/DataDog/nikos/rpm/dnf"
	"github.com/DataDog/nikos/types"
)

type RepoInfo struct {
	repoName          string
	baseURL           string
	gpgKeyHostEtcPath string
}

type FedoraBackend struct {
	extraRepoInfo *RepoInfo
	dnfBackend    *dnf.DnfBackend
	logger        types.Logger
	target        *types.Target
}

func (b *FedoraBackend) GetKernelHeaders(directory string) error {
	for _, repo := range b.dnfBackend.GetEnabledRepositories() {
		if repo.Id != "base" && repo.Id != "updates" {
			b.dnfBackend.DisableRepository(repo)
			continue
		}
	}

	// First, check for the correct kernel-headers package
	pkgNevra := "kernel-headers-" + b.target.Uname.Kernel
	fmt.Printf("Repositories %+v\n", b.dnfBackend.GetEnabledRepositories())
	err := b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
	if err == nil {
		return nil
	}

	// If that doesn't work, try again with the updates-archive repo
	updatesRepoGPGKey := "file:///" + types.HostEtc(b.extraRepoInfo.gpgKeyHostEtcPath)
	b.logger.Infof("Trying with %s repository", b.extraRepoInfo.repoName)
	if _, err := b.dnfBackend.AddRepository(b.extraRepoInfo.repoName, b.extraRepoInfo.baseURL, true, updatesRepoGPGKey, "", "", ""); err == nil {
		err = b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
		if err == nil {
			return nil
		}
	} else {
		b.logger.Warnf("Failed to add %s repository: %w", b.extraRepoInfo.repoName, err)
	}

	// As a last resort, check for the kernel-devel package
	pkgNevra = "kernel-devel-" + b.target.Uname.Kernel
	return b.dnfBackend.GetKernelHeaders(pkgNevra, directory)
}

func (b *FedoraBackend) Close() {
	b.dnfBackend.Close()
}

func newRawFedoraBackend(target *types.Target, extraRepoInfo *RepoInfo, reposDir string, logger types.Logger) (*FedoraBackend, error) {
	dnfBackend, err := dnf.NewDnfBackend(target.Distro.Release, reposDir, logger, target)
	if err != nil {
		return nil, fmt.Errorf("failed to create fedora dnf backend: %w", err)
	}

	return &FedoraBackend{
		extraRepoInfo: extraRepoInfo,
		target:        target,
		logger:        logger,
		dnfBackend:    dnfBackend,
	}, nil
}

func NewFedoraBackend(target *types.Target, reposDir string, logger types.Logger) (*FedoraBackend, error) {
	updatesArchiveRepoInfo := RepoInfo{
		repoName:          "updates-archive",
		baseURL:           "https://fedoraproject-updates-archive.fedoraproject.org/fedora/$releasever/$basearch/",
		gpgKeyHostEtcPath: "pki/rpm-gpg/RPM-GPG-KEY-fedora-$releasever-$basearch",
	}
	return newRawFedoraBackend(target, &updatesArchiveRepoInfo, reposDir, logger)
}

func NewAmazonLinux2022Backend(target *types.Target, reposDir string, logger types.Logger) (*FedoraBackend, error) {
	amazonLinuxRepoInfo := RepoInfo{
		repoName:          "amazonlinux",
		baseURL:           "https://al2022-repos-$awsregion-9761ab97.s3.dualstack.$awsregion.$awsdomain/core/mirrors/$releasever/$basearch/mirror.list",
		gpgKeyHostEtcPath: "pki/rpm-gpg/RPM-GPG-KEY-amazon-linux-2022",
	}
	return newRawFedoraBackend(target, &amazonLinuxRepoInfo, reposDir, logger)
}
