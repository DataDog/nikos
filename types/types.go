package types

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/DataDog/gopsutil/host"
	"github.com/acobaugh/osrelease"
	"golang.org/x/sys/unix"
)

type Backend interface {
	GetKernelHeaders(directory string) error
	Close()
}

type Utsname struct {
	Kernel  string
	Machine string
}

type Distro struct {
	Display string
	Release string
	Family  string
}
type Target struct {
	Distro    Distro
	OSRelease map[string]string
	Uname     Utsname
}

func NewTarget() (Target, error) {
	platform, family, version, err := host.PlatformInformation()
	if err != nil {
		return Target{}, err
	}

	platform = strings.Trim(platform, "\"")

	target := Target{
		Distro: Distro{
			Display: platform,
			Release: version,
			Family:  family,
		},
	}

	var uname unix.Utsname
	if err := unix.Uname(&uname); err != nil {
		return target, fmt.Errorf("failed to call uname syscall: %w", err)
	}

	target.Uname.Kernel = string(uname.Release[:bytes.IndexByte(uname.Release[:], 0)])
	target.Uname.Machine = string(uname.Machine[:bytes.IndexByte(uname.Machine[:], 0)])
	target.OSRelease = getOSRelease()

	if isWSL(target.Uname.Kernel) {
		target.Distro.Display, target.Distro.Family = "wsl", "wsl"
	} else if id := target.OSRelease["ID"]; target.Distro.Display == "" && id != "" {
		target.Distro.Display, target.Distro.Family = id, id
	}

	return target, nil
}

func isWSL(kernel string) bool {
	if strings.Contains(kernel, "Microsoft") {
		return true
	}
	if _, err := os.Stat("/run/WSL"); err == nil {
		return true
	}
	if f, err := ioutil.ReadFile("/proc/version"); err == nil && strings.Contains(string(f), "Microsoft") {
		return true
	}
	return false
}

func getOSRelease() map[string]string {
	osReleasePaths := []string{
		osrelease.EtcOsRelease,
		osrelease.UsrLibOsRelease,
	}

	if hostEtc := os.Getenv("HOST_ETC"); hostEtc != "" {
		osReleasePaths = append([]string{
			filepath.Join(hostEtc, "os-release"),
		}, osReleasePaths...)
	}

	var (
		release map[string]string
		err     error
	)
	for _, osReleasePath := range osReleasePaths {
		release, err = osrelease.ReadFile(osReleasePath)
		if err == nil {
			return release
		}
	}
	return make(map[string]string)
}

type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

//GetEnv retrieves the environment variable key. If it does not exist it returns the default.
func GetEnv(key string, dfault string, combineWith ...string) string {
	value := os.Getenv(key)
	if value == "" {
		value = dfault
	}

	switch len(combineWith) {
	case 0:
		return value
	case 1:
		return filepath.Join(value, combineWith[0])
	default:
		all := make([]string, len(combineWith)+1)
		all[0] = value
		copy(all[1:], combineWith)
		return filepath.Join(all...)
	}
}

func HostEtc(combineWith ...string) string {
	return GetEnv("HOST_ETC", "/etc", combineWith...)
}
