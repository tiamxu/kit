package gitlab

import (
	"github.com/urfave/cli/v2"
)

var (
	GetProjectCmd = cli.Command{
		Name:   "get",
		Usage:  "get gitlab project",
		Flags:  Flags,
		Before: InitFlags,
		Action: RunGetProject,
	}
	GetProjectsCmd = cli.Command{}
)
