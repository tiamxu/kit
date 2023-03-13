package kube

import "os"

func HomeDir() string {
	return os.Getenv("HOME")
}
