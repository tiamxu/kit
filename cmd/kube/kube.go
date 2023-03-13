package kube

import (
	"github.com/urfave/cli/v2"
)

var (
	RestartCmd = cli.Command{
		Name:  "restart",
		Usage: "build code and docker image",
		// Flags:  Flags,
		// Before: InitFlags,
		// Action: RunBuild,
	}
	DeleteCmd = cli.Command{
		Name:  "delete",
		Usage: "build code and docker image",
		// Flags:  Flags,
		// Before: InitFlags,
		// Action: RunBuild,
	}
)
