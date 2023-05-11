package kube

import (
	"github.com/urfave/cli/v2"
)

// main k8s commands
var (
	GetDeploymentCmd = cli.Command{
		Name:   "get",
		Usage:  "get k8s resource deployment",
		Flags:  Flags,
		Before: InitFlags,
		Action: RunGetDeployment,
	}
	RestartCmd = cli.Command{
		Name:   "restart",
		Usage:  "restart k8s resource deployment",
		Flags:  Flags,
		Before: InitFlags,
		Action: RunRestart,
	}
	CreateDeploymentCmd = cli.Command{
		Name:   "create",
		Usage:  "create resource deployment",
		Flags:  Flags,
		Before: InitFlags,
		Action: CreateDeployment,
	}
)
