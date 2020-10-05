package cmd

import (
	"os"

	"github.com/cobaugh/osrelease"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/lebauce/igor/types"
)

var (
	osReleaseFile string
	Target        types.Target
	OutputDir     string
	Verbose       bool
)

var RootCmd = &cobra.Command{
	Use:          "igor [sub]",
	SilenceUsage: true,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		if osReleaseFile != "" {
			var err error
			if Target.OSRelease, err = osrelease.ReadFile(osReleaseFile); err != nil {
				log.Fatalf("failed to read %s", osReleaseFile)
			}
		}

		if Verbose {
			log.SetLevel(log.DebugLevel)
			log.Debugf("Set log level to debug")
		}
	},
}

var DownloadCmd = &cobra.Command{
	Use: "download package",
}

func init() {
	var err error
	Target, err = types.NewTarget()
	if err != nil {
		log.Fatalf("failed to retrieve target information: %s", err)
	}

	if _, err := os.Stat("/run/WSL"); err == nil {
		Target.Distro.Display = "wsl"
	} else if id := Target.OSRelease["ID"]; Target.Distro.Display == "" && id != "" {
		Target.Distro.Display = id
	}

	RootCmd.PersistentFlags().StringVarP(&osReleaseFile, "os-release", "", "", "path to os-release file")
	RootCmd.PersistentFlags().StringVarP(&Target.Distro.Display, "distribution", "d", Target.Distro.Display, "distribution name")
	RootCmd.PersistentFlags().StringVarP(&Target.Distro.Release, "release", "r", Target.Distro.Release, "distribution release")
	RootCmd.PersistentFlags().StringVarP(&Target.Uname.Kernel, "kernel", "k", Target.Uname.Kernel, "kernel version")
	RootCmd.PersistentFlags().StringVarP(&Target.Uname.Machine, "arch", "a", Target.Uname.Machine, "architecture")
	RootCmd.PersistentFlags().StringVarP(&OutputDir, "output", "o", "/tmp", "output directory")
	RootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose mode")
	RootCmd.AddCommand(DownloadCmd)
	RootCmd.Execute()
}
