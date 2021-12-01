package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/acobaugh/osrelease"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/DataDog/nikos/apt"
	"github.com/DataDog/nikos/cos"
	"github.com/DataDog/nikos/rpm"
	"github.com/DataDog/nikos/types"
	"github.com/DataDog/nikos/wsl"
)

var (
	osReleaseFile string
	target        types.Target
	outputDir     string
	verbose       bool
	aptConfigDir  string
	rpmReposDir   string
)

var RootCmd = &cobra.Command{
	Use:          "nikos [sub]",
	SilenceUsage: true,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		if osReleaseFile != "" {
			var err error
			if target.OSRelease, err = osrelease.ReadFile(osReleaseFile); err != nil {
				log.Fatalf("failed to read %s", osReleaseFile)
			}
		}

		if verbose {
			log.SetLevel(log.DebugLevel)
			log.Debugf("Set log level to debug")
		}
	},
}

var DownloadCmd = &cobra.Command{
	Use: "download package",
	Run: func(c *cobra.Command, args []string) {
		log.Infof("OS family: %s\n", target.Distro.Family)
		log.Infof("Distribution: %s\n", target.Distro.Display)
		log.Infof("Release: %s\n", target.Distro.Release)
		log.Infof("Kernel: %s\n", target.Uname.Kernel)
		log.Infof("Machine: %s\n", target.Uname.Machine)
		log.Debugf("OSRelease: %s\n", target.OSRelease)

		var (
			backend types.Backend
			err     error
		)

		logger := log.New()
		if verbose {
			logger.SetLevel(log.DebugLevel)
		}
		switch target.Distro.Family {
		case "fedora", "rhel":
			switch target.Distro.Display {
			case "fedora":
				backend, err = rpm.NewFedoraBackend(&target, rpmReposDir, logger)
			case "rhel", "redhat":
				backend, err = rpm.NewRedHatBackend(&target, rpmReposDir, logger)
			case "centos":
				backend, err = rpm.NewCentOSBackend(&target, rpmReposDir, logger)
			default:
				err = fmt.Errorf("unsupported RedHat based distribution '%s'", target.Distro.Display)
			}
		case "suse":
			switch target.Distro.Display {
			case "suse", "sles", "sled", "caasp":
				backend, err = rpm.NewSLESBackend(&target, rpmReposDir, logger)
			case "opensuse", "opensuse-leap", "opensuse-tumbleweed", "opensuse-tumbleweed-kubic":
				backend, err = rpm.NewOpenSUSEBackend(&target, rpmReposDir, logger)
			default:
				err = fmt.Errorf("unsupported Debian based distribution '%s'", target.Distro.Display)
			}
		case "debian":
			backend, err = apt.NewBackend(&target, aptConfigDir, logger)
		case "cos":
			backend, err = cos.NewBackend(&target, logger)
		case "wsl":
			backend, err = wsl.NewBackend(&target, logger)
		default:
			err = fmt.Errorf("unsupported distribution '%s'", target.Distro.Display)
		}
		if err != nil {
			log.Fatal(err)
		}

		if err = os.MkdirAll(outputDir, 0755); err != nil {
			log.Fatal(err)
		}

		if err = backend.GetKernelHeaders(outputDir); err != nil {
			log.Fatalf("failed to download kernel headers: %s", err)
		}
	},
}

func SetupCommands() error {
	var err error
	target, err = types.NewTarget()
	if err != nil && strings.HasPrefix(err.Error(), "failed to read default os-release file") {
		log.Warnf("%s: please use the -os-release flag to provide the path to a valid os-release file", err)
		target.OSRelease = make(map[string]string)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve target information: %s", err)
	}

	RootCmd.PersistentFlags().StringVarP(&osReleaseFile, "os-release", "", "", "path to os-release file")
	RootCmd.PersistentFlags().StringVarP(&target.Distro.Family, "family", "f", target.Distro.Family, "OS family")
	RootCmd.PersistentFlags().StringVarP(&target.Distro.Display, "platform", "p", target.Distro.Display, "OS platform")
	RootCmd.PersistentFlags().StringVarP(&target.Distro.Release, "release", "r", target.Distro.Release, "distribution release")
	RootCmd.PersistentFlags().StringVarP(&target.Uname.Kernel, "kernel", "k", target.Uname.Kernel, "kernel version")
	RootCmd.PersistentFlags().StringVarP(&target.Uname.Machine, "arch", "a", target.Uname.Machine, "architecture")
	RootCmd.PersistentFlags().StringVarP(&outputDir, "output", "o", "/tmp", "output directory")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose mode")

	switch target.Distro.Family {
	case "debian":
		RootCmd.PersistentFlags().StringVarP(&aptConfigDir, "apt-config-dir", "", types.HostEtc("apt"), "APT configuration dir")
	case "fedora", "rhel":
		RootCmd.PersistentFlags().StringVarP(&rpmReposDir, "yum-repos-dir", "", types.HostEtc("yum.repos.d"), "YUM configuration dir")
	case "suse":
		RootCmd.PersistentFlags().StringVarP(&rpmReposDir, "yum-repos-dir", "", types.HostEtc("zypp", "repos.d"), "YUM configuration dir")
	default:
	}

	RootCmd.AddCommand(DownloadCmd)
	return nil
}
