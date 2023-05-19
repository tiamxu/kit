package build

import (
	"github.com/urfave/cli/v2"
)

var (
	BuildCmd = cli.Command{
		Name:   "build",
		Usage:  "build code and docker image",
		Flags:  Flags,
		Before: InitFlags,
		Action: RunBuild,
	}
	PushCmd = cli.Command{

		Name:   "push",
		Usage:  "docker push image registry",
		Flags:  Flags,
		Before: InitFlags,
		Action: RunPush,
	}
)

// var DeployCommand = cli.Command{

// 	Name:   "deploy",
// 	Usage:  "manager deploy server",
// 	Before: InitProject,
// 	Subcommands: []*cli.Command{
// {
// 	Name:   "build",
// 	Usage:  "build code and docker image",
// 	Flags:  Flags,
// 	Before: InitFlags,
// 	Action: RunBuild,
// },
// {
// 	Name:   "push",
// 	Usage:  "docker push image registry",
// 	Flags:  Flags,
// 	Before: InitFlags,
// 	Action: RunPush,
// },
// 		&BuildCmd,
// 		&PushCmd,
// 		&kube.RestartCmd,
// 	},
// }
