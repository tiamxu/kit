package main

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	projectNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	modulePathPattern  = regexp.MustCompile(`^[A-Za-z0-9._~/-]+$`)
)

func ValidateOptions(opts Options) error {
	if opts.Type != "api" {
		return fmt.Errorf("unsupported project type %q, only api is supported", opts.Type)
	}
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	if !projectNamePattern.MatchString(name) || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("--name must be a project name, not a path")
	}
	module := strings.TrimSpace(opts.Module)
	if module == "" {
		return fmt.Errorf("--module is required")
	}
	if strings.Contains(module, `\`) || strings.Contains(module, " ") || !strings.Contains(module, "/") || !modulePathPattern.MatchString(module) {
		return fmt.Errorf("--module must be a valid go module path")
	}
	if strings.TrimSpace(opts.Target) == "" {
		return fmt.Errorf("--target is required")
	}
	return nil
}
