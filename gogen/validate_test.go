package main

import "testing"

func TestValidateOptionsRequiresNameModuleAndTarget(t *testing.T) {
	tests := []struct {
		name string
		opts Options
	}{
		{name: "missing name", opts: Options{Module: "github.com/tiamxu/demo", Target: t.TempDir()}},
		{name: "missing module", opts: Options{Name: "demo", Target: t.TempDir()}},
		{name: "missing target", opts: Options{Name: "demo", Module: "github.com/tiamxu/demo"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateOptions(tt.opts); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidateOptionsRejectsUnsupportedProjectType(t *testing.T) {
	opts := Options{
		Type:   "web",
		Name:   "demo",
		Module: "github.com/tiamxu/demo",
		Target: t.TempDir(),
	}

	if err := ValidateOptions(opts); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseArgsAcceptsLocalKitReplace(t *testing.T) {
	opts, err := ParseArgs([]string{
		"new",
		"api",
		"--name",
		"demo-api",
		"--module",
		"github.com/tiamxu/demo-api",
		"--target",
		t.TempDir(),
		"--local-kit-replace",
		"--with-db",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.LocalKitReplace {
		t.Fatal("expected local kit replace to be enabled")
	}
	if !opts.WithDB {
		t.Fatal("expected database generation to be enabled")
	}
}
