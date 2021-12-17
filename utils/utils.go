package utils

import (
	"io"
	"os"
)

func CopyFile(srcPath, destPath string) error {
	dest, err := os.OpenFile(destPath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer dest.Close()

	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	_, err = io.Copy(dest, src)
	return err
}
