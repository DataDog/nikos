// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package backend

import (
	"crypto/x509"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/DataDog/nikos/rpm/dnfv2/internal/utils"
)

const (
	// certFileEnv is the environment variable which identifies where to locate
	// the SSL certificate file. If set this overrides the system default.
	certFileEnv = "SSL_CERT_FILE"

	// certDirEnv is the environment variable which identifies which directory
	// to check for SSL certificate files. If set this overrides the system default.
	// It is a colon separated list of directories.
	// See https://www.openssl.org/docs/man1.0.2/man1/c_rehash.html.
	certDirEnv = "SSL_CERT_DIR"
)

// Possible certificate files; stop after finding one.
var certFiles = []string{
	"/etc/ssl/certs/ca-certificates.crt",                // Debian/Ubuntu/Gentoo etc.
	"/etc/pki/tls/certs/ca-bundle.crt",                  // Fedora/RHEL 6
	"/etc/ssl/ca-bundle.pem",                            // OpenSUSE
	"/etc/pki/tls/cacert.pem",                           // OpenELEC
	"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem", // CentOS/RHEL 7
	"/etc/ssl/cert.pem",                                 // Alpine Linux
}

// Possible directories with certificate files; all will be read.
var certDirectories = []string{
	"/etc/ssl/certs",     // SLES10/SLES11, https://golang.org/issue/12139
	"/etc/pki/tls/certs", // Fedora/RHEL
}

func hostify(in []string) []string {
	out := make([]string, len(in), len(in)*2)
	copy(out, in)
	for _, i := range in {
		out = append(out, utils.HostEtcJoin(i))
	}
	return out
}

func GetSystemRoots() (*x509.CertPool, error) {
	hostifiedCertFiles := hostify(certFiles)
	hostifiedCertDirs := hostify(certDirectories)

	roots := x509.NewCertPool()

	files := hostifiedCertFiles
	if f := os.Getenv(certFileEnv); f != "" {
		files = []string{f}
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if !os.IsNotExist(err) {
			return nil, err
		}
		if err == nil {
			roots.AppendCertsFromPEM(data)
		}
	}

	dirs := hostifiedCertDirs
	if d := os.Getenv(certDirEnv); d != "" {
		// OpenSSL and BoringSSL both use ":" as the SSL_CERT_DIR separator.
		// See:
		//  * https://golang.org/issue/35325
		//  * https://www.openssl.org/docs/man1.0.2/man1/c_rehash.html
		dirs = strings.Split(d, ":")
	}

	for _, directory := range dirs {
		fis, err := readUniqueDirectoryEntries(directory)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			continue
		}
		for _, fi := range fis {
			data, err := os.ReadFile(directory + "/" + fi.Name())
			if err == nil {
				roots.AppendCertsFromPEM(data)
			}
		}
	}

	return roots, nil
}

// readUniqueDirectoryEntries is like os.ReadDir but omits
// symlinks that point within the directory.
func readUniqueDirectoryEntries(dir string) ([]fs.DirEntry, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	uniq := files[:0]
	for _, f := range files {
		if !isSameDirSymlink(f, dir) {
			uniq = append(uniq, f)
		}
	}
	return uniq, nil
}

// isSameDirSymlink reports whether fi in dir is a symlink with a
// target not containing a slash.
func isSameDirSymlink(f fs.DirEntry, dir string) bool {
	if f.Type()&fs.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(filepath.Join(dir, f.Name()))
	return err == nil && !strings.Contains(target, "/")
}
