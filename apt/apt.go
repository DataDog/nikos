package apt

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/database/goleveldb"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/http"
	"github.com/arduino/go-apt-client"
	"github.com/pkg/errors"
	"github.com/xor-gate/ar"

	"github.com/DataDog/nikos/tarball"
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
			return errors.Wrap(err, "failed to decompress deb")
		}
		b.logger.Debugf("Found header: %s", header.Name)

		if strings.HasPrefix(header.Name, "data.tar") {
			return tarball.ExtractTarball(reader, header.Name, directory, b.logger)
		}
	}

	return errors.New("failed to decompress deb")
}

func (b *Backend) GetKernelHeaders(directory string) error {
	downloader := http.NewDownloader(0, 1, nil)

	// TODO(lebauce) fix GPG verifier
	// gpgVerifier := pgp.NewGpgVerifier(pgp.GPGDefaultFinder())

	collectionFactory := deb.NewCollectionFactory(b.db)

	kernelRelease := b.target.Uname.Kernel
	query := &deb.FieldQuery{
		Field:    "Name",
		Relation: deb.VersionPatternMatch,
		Value:    "linux-headers-" + kernelRelease + "*",
	}
	b.logger.Infof("Looking for %s", query.Value)

	var packageURL *url.URL

	err := b.repoCollection.ForEach(func(repo *deb.RemoteRepo) error {
		if packageURL != nil {
			return nil
		}

		b.logger.Debugf("Fetching repository: name=%s, distribution=%s, components=%v, arch=%v", repo.Name, repo.Distribution, repo.Components, repo.Architectures)
		if err := repo.Fetch(downloader, nil); err != nil {
			b.logger.Debugf("Error fetching repo: %s", err)
			return err
		}

		b.logger.Debug("Downloading package indexes")
		if err := repo.DownloadPackageIndexes(nil, downloader, nil, collectionFactory, false); err != nil {
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
				return errors.New("No package file for %s" + pkg.Name)
			}

			packageURL = repo.PackageURL(packageFiles[0].DownloadURL())
			b.logger.Infof("Package URL: %s", packageURL)
			return nil
		})

		return nil
	})

	if err != nil {
		return err
	}

	if packageURL == nil {
		return errors.New("failed to find package linux-headers-" + kernelRelease)
	}

	b.logger.Info("Downloading package")
	url := packageURL.String()
	outputFile := filepath.Join(directory, filepath.Base(url))
	if err := downloader.Download(context.Background(), url, outputFile); err != nil {
		return errors.Wrap(err, "failed to download "+url+" to "+directory)
	}
	// defer os.Remove(outputFile)

	return b.extractPackage(outputFile, directory)
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
		return nil, errors.Wrap(err, "failed to create aptly database")
	}

	backend.repoCollection = deb.NewRemoteRepoCollection(backend.db)

	repoList, err := apt.ParseAPTConfigFolder(aptConfigDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse APT folder")
	}

	for i, repo := range repoList {
		if repo.Enabled && !repo.SourceRepo {
			prefix := target.Distro.Codename
			if prefix == "" {
				prefix = target.Distro.Display
			}
			repoID := fmt.Sprintf("%s-%d", prefix, i)
			components := strings.Split(repo.Components, " ")
			remoteRepo, err := deb.NewRemoteRepo(repoID, repo.URI, repo.Distribution, components, []string{debArch}, false, false, false)
			if err != nil {
				return nil, err
			}

			if err := backend.repoCollection.Add(remoteRepo); err != nil {
				backend.Close()
				return nil, errors.Wrap(err, "failed to add collection")
			}

			backend.logger.Debugf("Added repository '%s' %s %s %v %v", repoID, repo.URI, repo.Distribution, components, debArch)
		}
	}

	return backend, nil
}
