package kube

import (
	"os"

	"github.com/urfave/cli/v2"
)

var (
	namespace string
	name      string
	Flags     = []cli.Flag{
		&cli.StringFlag{
			Name:    "namespace",
			Aliases: []string{"n"},
			Usage:   "Set k8s namespace",
		},
		&cli.StringFlag{
			Name:  "name",
			Usage: "Set k8s deployment name",
		},
	}
)

func InitFlags(c *cli.Context) error {
	namespace = c.String("namespace")
	if namespace == "" {
		namespace = "default"
		kubeconf = os.Getenv("HOME") + "/.kube/config"

	} else {
		kubeconf = os.Getenv("HOME") + "/.kube/config-" + namespace

	}
	name = c.String("name")

	return nil
}

func int32Ptr(i int32) *int32 { return &i }
