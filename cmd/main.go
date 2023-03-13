package main

import (
	"a/kit/cmd/build"
	"a/kit/cmd/kube"
	"fmt"
	"log"
	"os"
	"runtime"

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

	var deployCommand = cli.Command{

		Name:   "deploy",
		Usage:  "manager deploy server",
		Before: build.InitProject,
		Subcommands: []*cli.Command{

			&build.BuildCmd,
			&build.PushCmd,
			&kube.RestartCmd,
		},
	}
	app.Commands = []*cli.Command{

		&deployCommand,
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)

	}
}
