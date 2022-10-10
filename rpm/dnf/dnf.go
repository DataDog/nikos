//go:build dnf
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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/go-ini/ini"

	"github.com/DataDog/nikos/extract"
	"github.com/DataDog/nikos/types"
	"github.com/DataDog/nikos/utils"
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

	pkgDescriptions := []lookupPkgDescription{
		{pkgNevra, C.HY_EQ},
		{fmt.Sprintf("%s*%s", pkgNevra, b.target.Uname.Machine), C.HY_GLOB},
		{fmt.Sprintf("%s*noarch", pkgNevra), C.HY_GLOB},
	}

	pkg, err := b.lookupPackages(C.HY_PKG_NEVRA, pkgDescriptions)
	if err != nil {
		return err
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
	return extract.ExtractRPMPackage(pkgPath, directory, b.target.Uname.Kernel, logger)
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

type lookupPkgDescription struct {
	value      string
	comparison int
}

func (b *DnfBackend) lookupPackages(filter int, descriptions []lookupPkgDescription) (*C.DnfPackage, error) {
	var (
		pkg *C.DnfPackage
		err error
	)
	for _, desc := range descriptions {
		logger.Infof("Looking for package %s", desc.value)
		pkg, err = b.lookupPackage(C.HY_PKG_NEVRA, desc.comparison, desc.value)
		if err == nil {
			return pkg, nil
		}
	}
	return nil, err
}

func (b *DnfBackend) AddRepository(id, baseurl string, enabled bool, gpgKey, sslCACert, sslClientCert, sslClientKey string) (*Repository, error) {
	idC := C.CString(id)
	defer C.free(unsafe.Pointer(idC))

	baseurlC := C.CString(baseurl)
	defer C.free(unsafe.Pointer(baseurlC))

	gpgKeyC := C.CString(gpgKey)
	defer C.free(unsafe.Pointer(gpgKeyC))

	sslCACertC := C.CString(sslCACert)
	defer C.free(unsafe.Pointer(sslCACertC))

	sslClientCertC := C.CString(sslClientCert)
	defer C.free(unsafe.Pointer(sslClientCertC))

	sslClientKeyC := C.CString(sslClientKey)
	defer C.free(unsafe.Pointer(sslClientKeyC))

	result := C.AddRepository(b.dnfContext, idC, baseurlC, C.bool(enabled), gpgKeyC, sslCACertC, sslClientCertC, sslClientKeyC)

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

type ReplacerPair struct {
	varName string
	value   string
}

type ReplacerState struct {
	pairs []ReplacerPair
}

func NewReplacerState() *ReplacerState {
	s := &ReplacerState{}
	s.loadVarsFromDir(types.HostEtc("yum/vars"))
	s.loadVarsFromDir(types.HostEtc("dnf/vars"))
	return s
}

func (s *ReplacerState) loadVarsFromDir(dir string) {
	files, err := os.ReadDir(dir)
	if err == nil {
		for _, file := range files {
			filename := file.Name()
			if content, err := os.ReadFile(filepath.Join(dir, filename)); err == nil {
				s.pairs = append(s.pairs, ReplacerPair{
					varName: filename,
					value:   strings.TrimSpace(string(content)),
				})
			}
		}
	}
}

func hostifyRepositories(reposDir string) (string, string, error) {
	tmpDir, err := ioutil.TempDir("", "repos.d")
	if err != nil {
		return "", "", err
	}

	varsDir, err := ioutil.TempDir("", "vars")
	if err != nil {
		return "", "", err
	}

	logger.Infof("Scanning repo files in '%s'", reposDir)
	repoFiles, err := filepath.Glob(reposDir + "/*.repo")
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", "", err
	}

	replacerState := NewReplacerState()
	for _, pair := range replacerState.pairs {
		destFilename := filepath.Join(varsDir, pair.varName)
		if err := os.WriteFile(destFilename, []byte(pair.value), 0o644); err != nil {
			logger.Warnf("Failed to write var file (`%s`)", pair.varName)
		}
	}

	for _, repoFile := range repoFiles {
		destFilename := filepath.Join(tmpDir, filepath.Base(repoFile))

		logger.Infof("Reading repo file '%s'", repoFile)
		cfg, err := ini.Load(repoFile)
		if err != nil {
			logger.Warnf("Failed to read file '%s': %v", repoFile, err)

			err = utils.CopyFile(repoFile, destFilename)
			if err != nil {
				logger.Warnf("Failed to copy %s to tmp dir: %v", repoFile, err)
			}
			continue
		}

		sections := cfg.Sections()
		for _, section := range sections {
			keys := section.Keys()
			for _, key := range keys {
				value := key.String()

				if strings.HasPrefix(value, "/etc/") {
					key.SetValue(types.HostEtc(strings.TrimPrefix(value, "/etc/")))
				} else if strings.HasPrefix(value, "file:///etc/") {
					key.SetValue("file://" + types.HostEtc(strings.TrimPrefix(value, "file:///etc/")))
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

	return tmpDir, varsDir, nil
}

func NewDnfBackend(release string, reposDir string, l types.Logger, target *types.Target) (*DnfBackend, error) {
	logger = l
	C.dnf_set_default_handler()

	releaseC := C.CString(release)
	defer C.free(unsafe.Pointer(releaseC))

	tmpDir, varsDir, err := hostifyRepositories(reposDir)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	reposDirC := C.CString(tmpDir)
	defer C.free(unsafe.Pointer(reposDirC))

	varsDirC := C.CString(varsDir)
	defer C.free(unsafe.Pointer(varsDirC))

	result := C.CreateAndSetupDNFContext(releaseC, reposDirC, varsDirC)
	if result.err_msg != nil {
		defer C.free(unsafe.Pointer(result.err_msg))
		return nil, errors.New("error creating new dnf context: " + C.GoString(result.err_msg))
	}

	return &DnfBackend{
		target:     target,
		dnfContext: result.context,
	}, nil
}
