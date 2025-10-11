package main

import (
	"github.com/xi-mad/MontageGo/cmd"
)

var version = "dev" // This will be overwritten by the linker

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
