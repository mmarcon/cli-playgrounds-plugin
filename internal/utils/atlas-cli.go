package utils

import (
	"os"

	"github.com/shirou/gopsutil/process"
)

func AtlasCliExe() string {
	callerPid := os.Getppid()

	p, err := process.NewProcess(int32(callerPid))

	if err != nil {
		return "atlas"
	}

	atlasCliExe, err := p.Exe()

	if err != nil {
		return "atlas"
	}

	// if the executable is one of the shells, e.g. bash or zsh, we will assume
	// that we are testing the plugin and return the default value
	// we can also assume that all the shell executables end with "sh"
	if atlasCliExe[len(atlasCliExe)-2:] == "sh" {
		return "atlas"
	}

	return atlasCliExe
}
