package jenkins

import (
	"github.com/urfave/cli/v2"
)

var (
	CreateJobCmd = cli.Command{
		Name:   "create",
		Usage:  "create jenkins jobs",
		Flags:  Flags,
		Before: InitFlags,
		Action: RunCreateJob,
	}
)
