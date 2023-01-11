package utils

import (
	"os"
	"path/filepath"
	"strings"
)

func rawHostJoin(envName, defaultValue string, parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	first := parts[0]
	hostPath := os.Getenv(envName)
	if hostPath == "" || !strings.HasPrefix(first, defaultValue) {
		return filepath.Join(parts...)
	}

	first = strings.TrimPrefix(first, defaultValue)
	newParts := make([]string, len(parts)+1)
	newParts[0] = hostPath
	newParts[1] = first
	if len(parts) > 1 {
		copy(newParts[2:], parts[1:])
	}
	return filepath.Join(newParts...)
}

func HostEtcJoin(parts ...string) string {
	return rawHostJoin("HOST_ETC", "/etc", parts...)
}

func HostVarJoin(parts ...string) string {
	return rawHostJoin("HOST_VAR", "/var", parts...)
}
