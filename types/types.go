package types

import (
	"bytes"

	"github.com/cobaugh/osrelease"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
		return target, errors.Wrap(err, "failed to call uname syscall")
	}

	target.Uname.Kernel = string(uname.Release[:bytes.IndexByte(uname.Release[:], 0)])
	target.Uname.Machine = string(uname.Machine[:bytes.IndexByte(uname.Machine[:], 0)])

	target.OSRelease, err = osrelease.Read()
	if err != nil {
		log.Errorf("failed to read /etc/os-release file: %s", err)
		target.OSRelease = make(map[string]string)
	}

	return target, nil
}
