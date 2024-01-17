package apt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/DataDog/aptly/aptly"
	"github.com/DataDog/aptly/database"
	"github.com/DataDog/aptly/database/goleveldb"
	"github.com/DataDog/aptly/deb"
	"github.com/DataDog/aptly/http"
	"github.com/DataDog/aptly/pgp"
	"github.com/xor-gate/ar"

	"github.com/DataDog/nikos/extract"
	"github.com/DataDog/nikos/types"
)

type Backend struct {
	target         *types.Target
	logger         types.Logger
	repoCollection *deb.RemoteRepoCollection
	db             database.Storage
	tmpDir         string
}

func (b *Backend) Close() {
	b.db.Close()
	os.RemoveAll(b.tmpDir)
}

func (b *Backend) extractPackage(pkg, directory string) error {
	f, err := os.Open(pkg)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := ar.NewReader(f)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("failed to decompress deb: %w", err)
		}
		b.logger.Debugf("Found header: %s", header.Name)

		if strings.HasPrefix(header.Name, "data.tar") {
			return extract.ExtractTarball(reader, header.Name, directory, b.logger)
		}
	}

	return errors.New("failed to decompress deb")
}

func (b *Backend) downloadPackage(downloader aptly.Downloader, verifier pgp.Verifier, factory *deb.CollectionFactory, query *deb.FieldQuery, directory string) (*deb.PackageDependencies, error) {
	var packageURL *url.URL
	var packageDeps *deb.PackageDependencies

	stanza := make(deb.Stanza, 32)

	err := b.repoCollection.ForEach(func(repo *deb.RemoteRepo) error {
		if packageURL != nil {
			return nil
		}

		b.logger.Debugf("Fetching repository: name=%s, distribution=%s, components=%v, arch=%v", repo.Name, repo.Distribution, repo.Components, repo.Architectures)
		repo.SkipComponentCheck = true

		stanza.Clear()
		if err := repo.FetchBuffered(stanza, downloader, verifier); err != nil {
			b.logger.Debugf("Error fetching repo: %s", err)
			return err
		}

		b.logger.Debug("Downloading package indexes")
		if err := repo.DownloadPackageIndexes(nil, downloader, nil, factory, false); err != nil {
			b.logger.Debugf("Failed to download package indexes: %s", err)
			return err
		}

		_, _, err := repo.ApplyFilter(-1, query, nil)
		if err != nil {
			b.logger.Debugf("Failed to apply filter: %s", err)
			return err
		}

		/*
			// For some reason, this overrides the `downloadPath` field of package so we don't
			// have the full remote path of the package. As a workaround, aptly was patched to
			// expose the repository package list using RemoteRepo.PackageList()

			if err := repo.FinalizeDownload(collectionFactory, progress); err != nil {
				return errors.Wrap(err, "failed to finalize download")
			}

			refList := repo.RefList()
			packageList, err := deb.NewPackageListFromRefList(refList, collectionFactory.PackageCollection(), progress)
			if err != nil {
				return err
			}
		*/

		packageList := repo.PackageList()
		packageList.ForEach(func(pkg *deb.Package) error {
			b.logger.Infof("Found package %s with version %s", pkg.Name, pkg.Version)

			packageFiles := pkg.Files()
			if len(packageFiles) == 0 {
				return errors.New("No package file for " + pkg.Name)
			}

			packageURL = repo.PackageURL(packageFiles[0].DownloadURL())
			packageDeps = pkg.Deps()
			b.logger.Infof("Package URL: %s", packageURL)
			return nil
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	if packageURL == nil {
		return nil, errors.New("failed to find package " + query.Value)
	}

	b.logger.Info("Downloading package")
	url := packageURL.String()
	outputFile := filepath.Join(directory, filepath.Base(url))
	if err := downloader.Download(context.Background(), url, outputFile); err != nil {
		return nil, fmt.Errorf("failed to download %s to %s: %w", url, directory, err)
	}
	// defer os.Remove(outputFile)

	return packageDeps, b.extractPackage(outputFile, directory)
}

func (b *Backend) createGpgVerifier() (*pgp.GoVerifier, error) {
	gpgVerifier := &pgp.GoVerifier{}

	for _, searchPattern := range []string{types.HostEtc("apt", "trusted.gpg"), types.HostEtc("apt", "trusted.gpg.d", "*.gpg"), "/usr/share/keyrings/*.gpg"} {
		keyrings, err := filepath.Glob(searchPattern)
		if err != nil {
			return nil, fmt.Errorf("failed to find valid apt keyrings: %w", err)
		}
		for _, keyring := range keyrings {
			b.logger.Infof("Adding keyring from: %s", keyring)
			gpgVerifier.AddKeyring(keyring)
		}
	}

	if err := gpgVerifier.InitKeyring(); err != nil {
		return nil, err
	}
	return gpgVerifier, nil
}

func (b *Backend) GetKernelHeaders(directory string) error {
	downloader := http.NewDownloader(0, 1, nil)

	gpgVerifier, err := b.createGpgVerifier()
	if err != nil {
		return err
	}

	collectionFactory := deb.NewCollectionFactory(b.db)

	kernelRelease := b.target.Uname.Kernel
	query := &deb.FieldQuery{
		Field:    "Name",
		Relation: deb.VersionPatternMatch,
		Value:    fmt.Sprintf("linux-headers-%s", kernelRelease),
	}
	b.logger.Infof("Looking for %s", query.Value)

	dependencies, err := b.downloadPackage(downloader, gpgVerifier, collectionFactory, query, directory)
	if err != nil {
		return err
	}

	// Sometimes, the header package depends on other header packages
	// If this is the case, download the dependency in addition
	if dependencies != nil {
		for _, dep := range dependencies.Depends {
			if strings.Contains(dep, "linux") && strings.Contains(dep, "headers") {

				depName := strings.Split(dep, " ")[0]
				b.logger.Infof("Looking for dependency %s", dep)
				query = &deb.FieldQuery{
					Field:    "Name",
					Relation: deb.VersionPatternMatch,
					Value:    depName,
				}

				_, err = b.downloadPackage(downloader, gpgVerifier, collectionFactory, query, directory)
				if err != nil {
					b.logger.Warnf("Failed to download dependent package %s", depName)
				}
			}
		}
	}

	return nil
}

func NewBackend(target *types.Target, aptConfigDir string, logger types.Logger) (*Backend, error) {
	var debArch string
	switch target.Uname.Machine {
	case "x86_64":
		debArch = "amd64"
	case "i386", "i686":
		debArch = "i386"
	case "aarch64":
		debArch = "arm64"
	case "s390":
		debArch = "s390"
	case "s390x":
		debArch = "s390x"
	case "ppc64le":
		debArch = "ppc64el"
	case "mips64el":
		debArch = "mips64el"
	default:
		return nil, fmt.Errorf("unsupported architecture '%s'", target.Uname.Machine)
	}

	tmpDir, err := ioutil.TempDir("", "aptly-db")
	if err != nil {
		return nil, err
	}

	backend := &Backend{
		target: target,
		logger: logger,
		tmpDir: tmpDir,
	}

	if backend.db, err = goleveldb.NewOpenDB(tmpDir); err != nil {
		backend.Close()
		return nil, fmt.Errorf("failed to create aptly database: %w", err)
	}

	backend.repoCollection = deb.NewRemoteRepoCollection(backend.db)

	repoList, err := parseAPTConfigFolder(aptConfigDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse APT folder: %w", err)
	}

	for i, repo := range repoList {
		if !repo.Enabled || repo.SourceRepo {
			continue
		}

		prefix := target.Distro.Display
		repoID := fmt.Sprintf("%s-%d", prefix, i)

		var components []string
		if repo.Components != "" {
			components = strings.Split(repo.Components, " ")
		}

		remoteRepo, err := deb.NewRemoteRepo(repoID, repo.URI, repo.Distribution, components, []string{debArch}, false, false, false)
		if err != nil {
			return nil, err
		}

		if err := backend.repoCollection.Add(remoteRepo); err != nil {
			backend.Close()
			return nil, fmt.Errorf("failed to add collection: %w", err)
		}

		backend.logger.Debugf("Added repository '%s' %s %s %v %v", repoID, repo.URI, repo.Distribution, components, debArch)
	}

	return backend, nil
}
