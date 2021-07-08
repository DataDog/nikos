// +build dnf

package dnf

// #cgo pkg-config: gio-2.0
// #cgo pkg-config: libdnf
//
// #cgo LDFLAGS: -Wl,--wrap=__secure_getenv -Wl,--wrap=glob64 -Wl,--wrap=glob
// #include "glib_wrapper.h"
// #include "libdnf_wrapper.h"
//
// typedef const gchar cgchar_t;
// extern void go_log_handler(cgchar_t *log_domain, GLogLevelFlags log_level, cgchar_t *message, gpointer data);
// void dnf_set_default_handler() {
//      g_log_set_default_handler(go_log_handler, NULL);
// }
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/sassoftware/go-rpmutils"

	"github.com/DataDog/nikos/types"
)

var logger types.Logger

type Repository struct {
	Id         string
	libdnfRepo *C.DnfRepo
	enabled    bool
}

type DnfBackend struct {
	target     *types.Target
	dnfContext *C.struct__DnfContext
}

func (b *DnfBackend) GetKernelHeaders(pkgNevra, directory string) error {
	cErr := C.SetupDNFSack(b.dnfContext)
	if cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
		return errors.New("failed to setup dnf sack: " + C.GoString(cErr))
	}

	logger.Infof("Looking for package %s", pkgNevra)
	pkg, err := b.lookupPackage(C.HY_PKG_NEVRA, C.HY_EQ, pkgNevra)
	if err != nil {
		if pkg, err = b.lookupPackage(C.HY_PKG_NEVRA, C.HY_GLOB, pkgNevra+"*"); err != nil {
			return err
		}
	}
	defer C.g_object_unref(C.gpointer(pkg))
	logger.Infof("Found package %s", C.GoString(C.dnf_package_get_nevra(pkg)))

	outputDirectoryC := C.CString(directory)
	defer C.free(unsafe.Pointer(outputDirectoryC))

	result := C.DownloadPackage(b.dnfContext, pkg, outputDirectoryC)
	if result.err_msg != nil {
		defer C.free(unsafe.Pointer(result.err_msg))
		return errors.New("failed to download package: " + C.GoString(result.err_msg))
	}

	pkgPath := filepath.Join(directory, filepath.Base(C.GoString(result.filename)))
	return b.extractPackage(pkgPath, directory)
}

func (b *DnfBackend) lookupPackage(filter, comparison int, value string) (*C.DnfPackage, error) {
	valueC := C.CString(value)
	defer C.free(unsafe.Pointer(valueC))

	result := C.LookupPackage(b.dnfContext, C.int(filter), C.int(comparison), valueC)

	if result.err_msg != nil {
		defer C.free(unsafe.Pointer(result.err_msg))
		return nil, errors.New("error looking up package " + value + ": " + C.GoString(result.err_msg))
	}
	return result.pkg, nil
}

func (b *DnfBackend) extractPackage(pkg, directory string) error {
	pkgFile, err := os.Open(pkg)
	if err != nil {
		return errors.Wrapf(err, "failed to open download package %s", pkg)
	}

	rpm, err := rpmutils.ReadRpm(pkgFile)
	if err != nil {
		return errors.Wrapf(err, "failed to parse RPM package %s", pkg)
	}

	if err := rpm.ExpandPayload(directory); err != nil {
		return errors.Wrapf(err, "failed to extract RPM package %s", pkg)
	}

	return nil
}

func (b *DnfBackend) AddRepository(id, baseurl string, enabled bool, gpgKey string) (*Repository, error) {
	idC := C.CString(id)
	defer C.free(unsafe.Pointer(idC))

	baseurlC := C.CString(baseurl)
	defer C.free(unsafe.Pointer(baseurlC))

	gpgKeyC := C.CString(gpgKey)
	defer C.free(unsafe.Pointer(gpgKeyC))

	result := C.AddRepository(b.dnfContext, idC, baseurlC, C.bool(enabled), gpgKeyC)

	if result.err_msg != nil {
		defer C.free(unsafe.Pointer(result.err_msg))
		return nil, errors.New("failed to setup repository " + id + ": " + C.GoString(result.err_msg))
	}
	return &Repository{
		Id:         id,
		libdnfRepo: result.libdnf_repo,
		enabled:    enabled,
	}, nil
}

func (b *DnfBackend) EnableRepository(repo *Repository) error {
	err := C.EnableRepository(b.dnfContext, repo.libdnfRepo)
	if err != nil {
		defer C.free(unsafe.Pointer(err))
		return fmt.Errorf("failed to enable repository '%s': %s", repo.Id, err)
	}
	return nil
}

func (b *DnfBackend) DisableRepository(repo *Repository) error {
	err := C.DisableRepository(b.dnfContext, repo.libdnfRepo)
	if err != nil {
		defer C.free(unsafe.Pointer(err))
		return fmt.Errorf("failed to disable repository '%s': %s", repo.Id, err)
	}
	return nil
}

func (b *DnfBackend) Close() {
	if b.dnfContext != nil {
		C.g_object_unref(C.gpointer(b.dnfContext))
	}
}

func (b *DnfBackend) GetRepositories() (repos []*Repository) {
	numRepos := C.GetNumRepositories(b.dnfContext)
	if numRepos == 0 {
		return
	}

	dnfRepos := make([]*C.DnfRepo, numRepos)
	if C.GetRepositories(b.dnfContext, &dnfRepos[0], C.int(len(dnfRepos))) {
		for _, dnfRepo := range dnfRepos {
			// Note: the libdnf functions below are safe to call here (they shouldn't throw exceptions)
			repos = append(repos, &Repository{
				Id:         C.GoString(C.dnf_repo_get_id(dnfRepo)),
				libdnfRepo: dnfRepo,
				enabled:    C.dnf_repo_get_enabled(dnfRepo) != 0,
			})
		}
	}
	return
}

func (b *DnfBackend) GetEnabledRepositories() (repos []*Repository) {
	for _, repository := range b.GetRepositories() {
		if repository.enabled {
			repos = append(repos, repository)
		}
	}
	return
}

func NewDnfBackend(release string, reposDir string, l types.Logger) (*DnfBackend, error) {
	logger = l
	C.dnf_set_default_handler()

	releaseC := C.CString(release)
	defer C.free(unsafe.Pointer(releaseC))

	reposDirC := C.CString(reposDir)
	defer C.free(unsafe.Pointer(reposDirC))

	result := C.CreateAndSetupDNFContext(releaseC, reposDirC)
	if result.err_msg != nil {
		defer C.free(unsafe.Pointer(result.err_msg))
		return nil, errors.New("error creating new dnf context: " + C.GoString(result.err_msg))
	}

	return &DnfBackend{
		dnfContext: result.context,
	}, nil
}
