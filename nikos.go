package main

import (
	"github.com/DataDog/nikos/cmd"
)

func main() {
	cmd.SetupCommands()
	cmd.RootCmd.Execute()
}
