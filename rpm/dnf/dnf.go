// +build dnf

package dnf

// #cgo pkg-config: gio-2.0
// #cgo pkg-config: libdnf
// #include "libdnf_wrapper.h"
// #include <libdnf/libdnf.h>
//
// typedef const gchar cgchar_t;
// extern void download_percentage_changed_cb(DnfState* dnfState, guint value, gpointer data);
// extern void dnf_set_default_handler();
// void dnf_state_set_percentage_changed_cb(DnfState* dnfState);
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/sassoftware/go-rpmutils"
	"github.com/vbauerster/mpb/v5"

	"github.com/DataDog/nikos/types"
)

var (
	bar    *mpb.Bar
	logger types.Logger
)

func wrapGError(err *C.struct__GError, format string, a ...interface{}) error {
	if err == nil {
		return nil
	}
	defer C.g_error_free(err)
	return fmt.Errorf("%s: %s", fmt.Sprintf(format, a...), C.GoString(err.message))
}

type Repository struct {
	Id         string
	libdnfRepo *C.DnfRepo
	enabled    bool
}

type DnfBackend struct {
	target     *types.Target
	dnfContext *C.struct__DnfContext
	isSuse     bool
}

//export download_percentage_changed_cb
func download_percentage_changed_cb(state *C.struct__DnfState, value C.guint, data C.gpointer) {
	bar.SetCurrent(int64(value))
}

//export go_log_handler
func go_log_handler(log_domain *C.cgchar_t, log_level C.GLogLevelFlags, message *C.cgchar_t, data C.gpointer) {
	switch log_level {
	case C.G_LOG_LEVEL_DEBUG:
		logger.Debug(C.GoString(message))
	case C.G_LOG_LEVEL_INFO:
		logger.Info(C.GoString(message))
	case C.G_LOG_LEVEL_WARNING:
		logger.Warn(C.GoString(message))
	case C.G_LOG_LEVEL_ERROR, C.G_LOG_LEVEL_CRITICAL:
		logger.Error(C.GoString(message))
	}
}

func (b *DnfBackend) GetKernelHeaders(pkgNevra, directory string) error {
	result := C.SetupDNFSack(b.dnfContext)
	if result.err_msg != nil {
		defer C.free(unsafe.Pointer(result.err_msg))
		return errors.New("failed to setup dnf sack: " + C.GoString(result.err_msg))
	}

	logger.Infof("Looking for package %s", pkgNevra)
	pkg, err := b.lookupPackage(C.HY_PKG_NEVRA, C.HY_EQ, pkgNevra)
	if err != nil {
		if pkg, err = b.lookupPackage(C.HY_PKG_NEVRA, C.HY_GLOB, pkgNevra+"*"); err != nil {
			return err
		}
	}
	logger.Infof("Found package %s", C.GoString(C.dnf_package_get_nevra(pkg)))

	outputDirectoryC := C.CString(directory)
	defer C.free(unsafe.Pointer(outputDirectoryC))

	dwnldResult := C.DownloadPackage(b.dnfContext, result.dnf_state, pkg, outputDirectoryC)
	if dwnldResult.err_msg != nil {
		defer C.free(unsafe.Pointer(dwnldResult.err_msg))
		return errors.New("failed to download package: " + C.GoString(dwnldResult.err_msg))
	}

	return b.extractPackage(filepath.Join(directory, filepath.Base(C.GoString(dwnldResult.filename))), directory)
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
