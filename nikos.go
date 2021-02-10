package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/lebauce/nikos/apt"
	"github.com/lebauce/nikos/cmd"
	"github.com/lebauce/nikos/cos"
	"github.com/lebauce/nikos/rpm"
	"github.com/lebauce/nikos/types"
	"github.com/lebauce/nikos/wsl"
)

func main() {
	cmd.DownloadCmd.Run = func(c *cobra.Command, args []string) {
		target := cmd.Target
		log.Infof("Distribution: %s\n", target.Distro.Display)
		log.Infof("Release: %s\n", target.Distro.Release)
		log.Infof("Kernel: %s\n", target.Uname.Kernel)
		log.Debugf("OSRelease: %s\n", target.OSRelease)

		var (
			backend types.Backend
			err     error
		)

		switch target.Distro.Display {
		case "Fedora", "RHEL":
			backend, err = rpm.NewRedHatBackend(&target)
		case "CentOS":
			backend, err = rpm.NewCentOSBackend(&target)
		case "openSUSE":
			backend, err = rpm.NewOpenSUSEBackend(&target)
		case "SLE":
			backend, err = rpm.NewSLESBackend(&target)
		case "Debian", "Ubuntu":
			backend, err = apt.NewBackend(&target)
		case "cos":
			backend, err = cos.NewBackend(&target)
		case "wsl":
			backend, err = wsl.NewBackend(&target)
		default:
			log.Fatalf("Unsupported distribution '%s'", target.Distro.Display)
		}

		if err != nil {
			log.Fatal(err)
		}

		os.MkdirAll(cmd.OutputDir, 0755)

		if err = backend.GetKernelHeaders(cmd.OutputDir); err != nil {
			log.Fatalf("failed to download kernel headers: %s", err)
		}
	}

	cmd.RootCmd.Execute()
}
