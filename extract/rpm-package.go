package extract

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DataDog/nikos/types"
	"github.com/sassoftware/go-rpmutils"
)

func ExtractRPMPackage(pkg, directory, kernelUname string, l types.Logger) error {
	pkgFile, err := os.Open(pkg)
	if err != nil {
		return fmt.Errorf("failed to open download package %s: %w", pkg, err)
	}

	rpm, err := rpmutils.ReadRpm(pkgFile)
	if err != nil {
		return fmt.Errorf("failed to parse RPM package %s: %w", pkg, err)
	}

	if err := rpm.ExpandPayload(directory); err != nil {
		return fmt.Errorf("failed to extract RPM package %s: %w", pkg, err)
	}

	fixKernelModulesSymlinks(directory, kernelUname, l)

	return nil
}

// fixKernelModulesSymlinks prefixes the kernel modules symlinks with the path to the output directory
// This is necessary because we have installed the rpm package in a non-default directory, but the symlinks
// point to an installation in the root directory
func fixKernelModulesSymlinks(directory, kernelUname string, l types.Logger) {
	kernelModulesSymlinks := []string{
		filepath.Join(directory, fmt.Sprintf("/lib/modules/%s/build", kernelUname)),
		filepath.Join(directory, fmt.Sprintf("/lib/modules/%s/source", kernelUname)),
	}

	for _, symlink := range kernelModulesSymlinks {
		if fileInfo, err := os.Lstat(symlink); err != nil || !isSymlink(fileInfo) {
			continue
		}

		if destinationPath, err := os.Readlink(symlink); err == nil {
			if strings.HasPrefix(destinationPath, directory) {
				continue // symlink is already correct
			}

			if err := os.Remove(symlink); err != nil {
				l.Warnf("failed to unlink symlink at %s: %v", symlink, err)
				continue
			}

			newDestinationPath := filepath.Join(directory, destinationPath)
			err := os.Symlink(newDestinationPath, symlink)
			if err != nil {
				l.Warnf("failed to create symlink from %s to %s: %v", symlink, newDestinationPath, err)
				continue
			}
			l.Debugf("created symlink from %s to %s", symlink, newDestinationPath)
		}
	}

}

func isSymlink(fi os.FileInfo) bool {
	return fi.Mode()&os.ModeSymlink != 0
}
