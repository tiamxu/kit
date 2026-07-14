package main

import (
	"fmt"
	"os"
)

func main() {
	opts, err := ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := Generate(opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if opts.DryRun {
		fmt.Printf("dry-run completed: %s\n", opts.Target)
		return
	}
	fmt.Printf("project generated: %s\n", opts.Target)
}
