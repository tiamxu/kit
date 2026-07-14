package main

import (
	"flag"
	"fmt"
)

type Options struct {
	Type            string
	Name            string
	Module          string
	Target          string
	Force           bool
	DryRun          bool
	SkipTidy        bool
	SkipTest        bool
	LocalKitReplace bool
	WithDB          bool
}

func ParseArgs(args []string) (Options, error) {
	if len(args) < 2 {
		return Options{}, fmt.Errorf("usage: gogen new api --name <name> --module <module> --target <target> [--with-db]")
	}
	if args[0] != "new" {
		return Options{}, fmt.Errorf("unsupported command %q", args[0])
	}

	opts := Options{Type: args[1]}
	fs := flag.NewFlagSet("new "+opts.Type, flag.ContinueOnError)
	fs.StringVar(&opts.Name, "name", "", "project name")
	fs.StringVar(&opts.Module, "module", "", "go module path")
	fs.StringVar(&opts.Target, "target", "", "target directory")
	fs.BoolVar(&opts.Force, "force", false, "overwrite existing files")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "print files without writing")
	fs.BoolVar(&opts.SkipTidy, "skip-tidy", false, "skip go mod tidy")
	fs.BoolVar(&opts.SkipTest, "skip-test", false, "skip go test ./...")
	fs.BoolVar(&opts.LocalKitReplace, "local-kit-replace", false, "replace github.com/tiamxu/kit with local checkout")
	fs.BoolVar(&opts.WithDB, "with-db", false, "generate database config and repo initialization")
	if err := fs.Parse(args[2:]); err != nil {
		return Options{}, err
	}

	if err := ValidateOptions(opts); err != nil {
		return Options{}, err
	}
	return opts, nil
}
