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
	"github.com/vbauerster/mpb/v5/decor"

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

func (b *DnfBackend) lookupPackage(filter, comparison int, value string) (*C.DnfPackage, error) {
	sack := C.dnf_context_get_sack(b.dnfContext)
	query := C.hy_query_create(sack)
	defer C.hy_query_free(query)

	valueC := C.CString(value)
	defer C.free(unsafe.Pointer(valueC))

	C.hy_query_filter(query, C.int(filter), C.int(comparison), valueC)
	plist := C.hy_query_run(query)

	if plist.len == 0 {
		return nil, errors.New("failed to find package " + value)
	}

	return getPackage(plist), nil
}

func (b *DnfBackend) AddRepository(id, baseurl string, enabled bool, gpgKey string) (*Repository, error) {
	libdnfRepo := C.dnf_repo_new(b.dnfContext)

	C.dnf_repo_set_kind(libdnfRepo, C.DNF_REPO_KIND_REMOTE)

	keyFile := C.g_key_file_new()

	idC := C.CString(id)
	defer C.free(unsafe.Pointer(idC))

	baseurlKeyC := C.CString("baseurl")
	defer C.free(unsafe.Pointer(baseurlKeyC))

	baseurlC := C.CString(baseurl)
	defer C.free(unsafe.Pointer(baseurlC))
	C.g_key_file_set_string(keyFile, idC, baseurlKeyC, baseurlC)

	if gpgKey != "" {
		gpgkeyC := C.CString("gpgkey")
		defer C.free(unsafe.Pointer(gpgkeyC))

		gpgkeyPathC := C.CString(gpgKey)
		defer C.free(unsafe.Pointer(gpgkeyPathC))

		C.dnf_repo_set_gpgcheck(libdnfRepo, C.gboolean(1))
		C.g_key_file_set_string(keyFile, idC, gpgkeyC, gpgkeyC)
	}

	C.dnf_repo_set_keyfile(libdnfRepo, keyFile)

	C.dnf_repo_set_enabled(libdnfRepo, C.DNF_REPO_ENABLED_PACKAGES)

	C.dnf_repo_set_id(libdnfRepo, idC)

	filenameC := C.CString("/tmp/" + id + ".repo")
	defer C.free(unsafe.Pointer(filenameC))
	C.dnf_repo_set_filename(libdnfRepo, filenameC)

	var gerr *C.struct__GError
	if C.dnf_repo_setup(libdnfRepo, &gerr) == 0 {
		return nil, wrapGError(gerr, "failed to setup repository '%d'", id)
	}

	C.g_ptr_array_add(C.dnf_context_get_repos(b.dnfContext), C.gpointer(libdnfRepo))

	return &Repository{
		Id:         id,
		libdnfRepo: libdnfRepo,
		enabled:    enabled,
	}, nil
}

func (b *DnfBackend) GetKernelHeaders(pkgNevra, directory string) error {
	var gerr *C.struct__GError
	dnfState := C.dnf_state_new()
	C.dnf_context_setup_sack(b.dnfContext, dnfState, &gerr)
	if gerr != nil {
		return wrapGError(gerr, "failed to setup dnf sack")
	}

	logger.Infof("Looking for package %s", pkgNevra)

	pkg, err := b.lookupPackage(C.HY_PKG_NEVRA, C.HY_EQ, pkgNevra)
	if err != nil {
		if pkg, err = b.lookupPackage(C.HY_PKG_NEVRA, C.HY_GLOB, pkgNevra+"*"); err != nil {
			return err
		}
	}
	logger.Infof("Found package %s", C.GoString(C.dnf_package_get_nevra(pkg)))

	transaction := C.dnf_context_get_transaction(b.dnfContext)
	C.dnf_transaction_ensure_repo(transaction, pkg, &gerr)
	if gerr != nil {
		defer C.g_error_free(gerr)
		return fmt.Errorf("failed to set package repository: %s", C.GoString(gerr.message))
	}

	if C.dnf_package_installed(pkg) != 0 {
		return fmt.Errorf("package already installed")
	}

	outputDirectoryC := C.CString(directory)
	defer C.free(unsafe.Pointer(outputDirectoryC))

	C.dnf_state_set_percentage_changed_cb(dnfState)

	logger.Info("Downloading package")

	p := mpb.New()
	bar = p.AddBar(int64(100), mpb.AppendDecorators(decor.Percentage()))

	C.dnf_package_download(pkg, outputDirectoryC, dnfState, &gerr)
	if gerr != nil {
		return wrapGError(gerr, "failed to download package")
	}

	filename := C.GoString(C.dnf_package_get_filename(pkg))
	return b.extractPackage(filepath.Join(directory, filepath.Base(filename)), directory)
}

func (b *DnfBackend) Close() {
	if b.dnfContext != nil {
		C.g_object_unref(C.gpointer(b.dnfContext))
	}
}

func (b *DnfBackend) GetRepositories() (repos []*Repository) {
	repositories := C.dnf_context_get_repos(b.dnfContext)
	for i := 0; i < int(repositories.len); i++ {
		repository := getRepository(repositories, i)
		repos = append(repos, &Repository{
			Id:         C.GoString(C.dnf_repo_get_id(repository)),
			libdnfRepo: repository,
			enabled:    C.dnf_repo_get_enabled(repository) != 0,
		})
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

func (b *DnfBackend) DisableRepository(repo *Repository) error {
	var gerr *C.struct__GError
	C.dnf_context_repo_disable(b.dnfContext, C.dnf_repo_get_id(repo.libdnfRepo), &gerr)
	return wrapGError(gerr, "failed to setup dnf context")
}

func (b *DnfBackend) EnableRepository(repo *Repository) error {
	var gerr *C.struct__GError
	C.dnf_context_repo_enable(b.dnfContext, C.dnf_repo_get_id(repo.libdnfRepo), &gerr)
	return wrapGError(gerr, "failed to enable repository '%s'", repo.Id)
}

func NewDnfBackend(release string, reposDir string, l types.Logger) (*DnfBackend, error) {
	backend := &DnfBackend{
		dnfContext: C.dnf_context_new(),
	}
	logger = l

	C.dnf_set_default_handler()

	tmpDir := "/tmp"
	cacheDir := "/tmp/nikos-cache"
	solvDir := "/tmp/nikos-solv"

	tmpDirC := C.CString(tmpDir)
	defer C.free(unsafe.Pointer(tmpDirC))

	lock := C.dnf_lock_new()
	C.dnf_lock_set_lock_dir(lock, C.CString(tmpDir))

	solvDirC := C.CString(solvDir)
	defer C.free(unsafe.Pointer(solvDirC))
	C.dnf_context_set_solv_dir(backend.dnfContext, solvDirC)

	cacheDirC := C.CString(cacheDir)
	defer C.free(unsafe.Pointer(cacheDirC))
	C.dnf_context_set_cache_dir(backend.dnfContext, cacheDirC)

	if reposDir != "" {
		reposDirC := C.CString(reposDir)
		C.dnf_context_set_repo_dir(backend.dnfContext, reposDirC)
		C.free(unsafe.Pointer(reposDirC))
	}

	if solvDirC := C.dnf_context_get_solv_dir(backend.dnfContext); solvDirC != nil {
		logger.Infof("Solv directory: %s\n", C.GoString(solvDirC))
	}

	releaseVerC := C.CString(release)
	defer C.free(unsafe.Pointer(releaseVerC))
	C.dnf_context_set_release_ver(backend.dnfContext, releaseVerC)

	var gerr *C.struct__GError
	C.dnf_context_setup(backend.dnfContext, nil, &gerr)
	if gerr != nil {
		return nil, wrapGError(gerr, "failed to setup dnf context")
	}

	C.dnf_context_set_write_history(backend.dnfContext, 0)

	return backend, nil
}
