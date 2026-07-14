package main

import (
	"os"
	"path/filepath"
	"strings"
)

func findKitReplace() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir := wd
	for {
		modPath := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(modPath)
		if err == nil && hasModulePath(string(data), "github.com/tiamxu/kit") {
			return filepath.ToSlash(dir)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func hasModulePath(content string, module string) bool {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "module "+module {
			return true
		}
	}
	return false
}
