package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/tiamxu/kit/build"

	"github.com/tiamxu/kit/cmd/gitlab"
	"github.com/tiamxu/kit/jenkins"
	"github.com/tiamxu/kit/kube"

	"github.com/urfave/cli/v2"
)

var version string

func main() {
	version = "0.0.1"
	app := cli.NewApp()
	app.Name = ""
	app.Version = fmt.Sprintf("%s %s/%s", version, runtime.GOOS, runtime.GOARCH)
	app.Usage = "a new cmd tools"
	app.Authors = []*cli.Author{
		{
			Name:  "timaxu",
			Email: "1218366090@qq.com",
		},
	}

	var (
		DeployCommand = cli.Command{

			Name:   "deploy",
			Usage:  "manager deploy server",
			Before: build.InitProject,
			Subcommands: []*cli.Command{

				&build.BuildCmd,
				&build.PushCmd,
				&kube.RestartCmd,
				&kube.GetDeploymentCmd,
				&kube.CreateDeploymentCmd,
			},
		}
		JenkinsCommand = cli.Command{
			Name:  "jks",
			Usage: "manage jenkins cmd",
			Subcommands: []*cli.Command{
				&jenkins.CreateJobCmd,
			},
		}
		GitlabCommand = cli.Command{
			Name:  "git",
			Usage: "manage gitlab cmd",
			Subcommands: []*cli.Command{
				&gitlab.GetProjectCmd,
			},
		}
	)
	app.Commands = []*cli.Command{

		&DeployCommand,
		&JenkinsCommand,
		&GitlabCommand,
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)

	}
}
