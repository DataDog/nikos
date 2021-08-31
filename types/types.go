package types

import (
	"bytes"
	"fmt"

	"github.com/cobaugh/osrelease"
	"github.com/wille/osutil"
	"golang.org/x/sys/unix"
)

type Backend interface {
	GetKernelHeaders(directory string) error
}

type Utsname struct {
	Kernel  string
	Machine string
}

type Target struct {
	Distro    osutil.Distro
	OSRelease map[string]string
	Uname     Utsname
}

func NewTarget() (Target, error) {
	target := Target{
		Distro: osutil.GetDist(),
	}

	var err error
	var uname unix.Utsname
	if err := unix.Uname(&uname); err != nil {
		return target, fmt.Errorf("failed to call uname syscall: %w", err)
	}

	target.Uname.Kernel = string(uname.Release[:bytes.IndexByte(uname.Release[:], 0)])
	target.Uname.Machine = string(uname.Machine[:bytes.IndexByte(uname.Machine[:], 0)])

	if target.OSRelease, err = osrelease.Read(); err != nil {
		return target, fmt.Errorf("failed to read default os-release file: %s", err)
	}
	return target, nil
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
