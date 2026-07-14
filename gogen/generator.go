package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/**
var templatesFS embed.FS

type TemplateData struct {
	ProjectName string
	Module      string
	KitReplace  string
	WithDB      bool
}

type ProjectFile struct {
	SourcePath string
	TargetPath string
	Render     bool
}

type TemplateSet struct {
	Name string
	Root string
}

type Manifest struct {
	ProjectType string   `json:"project_type"`
	ProjectName string   `json:"project_name"`
	Module      string   `json:"module"`
	Files       []string `json:"files"`
}

func Generate(opts Options) error {
	if err := ValidateOptions(opts); err != nil {
		return err
	}
	set, err := templateSet(opts.Type)
	if err != nil {
		return err
	}
	files, err := projectFiles(set)
	if err != nil {
		return err
	}
	files = filterProjectFiles(files, opts)
	if opts.DryRun {
		for _, file := range files {
			fmt.Println(filepath.Join(opts.Target, filepath.FromSlash(file.TargetPath)))
		}
		fmt.Println(filepath.Join(opts.Target, ".gogen", "manifest.json"))
		return nil
	}
	if err := ensureWritableTarget(opts.Target, opts.Force); err != nil {
		return err
	}

	data := TemplateData{
		ProjectName: opts.Name,
		Module:      opts.Module,
		WithDB:      opts.WithDB,
	}
	if opts.LocalKitReplace {
		data.KitReplace = findKitReplace()
		if data.KitReplace == "" {
			return fmt.Errorf("--local-kit-replace enabled but github.com/tiamxu/kit repo was not found")
		}
	}
	for _, file := range files {
		if err := writeProjectFile(opts.Target, set, file, data, opts.Force); err != nil {
			return err
		}
	}
	if err := writeManifest(opts.Target, opts, files); err != nil {
		return err
	}

	if !opts.SkipTidy {
		if err := Run(opts.Target, "go", "mod", "tidy"); err != nil {
			return err
		}
	}
	if !opts.SkipTest {
		if err := Run(opts.Target, "go", "test", "./..."); err != nil {
			return err
		}
	}
	return nil
}

func templateSet(projectType string) (TemplateSet, error) {
	if projectType != "api" {
		return TemplateSet{}, fmt.Errorf("unsupported project type %q, only api is supported", projectType)
	}
	return TemplateSet{
		Name: "api",
		Root: "templates/api",
	}, nil
}

func projectFiles(set TemplateSet) ([]ProjectFile, error) {
	var files []ProjectFile
	err := fs.WalkDir(templatesFS, set.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		sourcePath, err := filepath.Rel(set.Root, path)
		if err != nil {
			return err
		}
		sourcePath = filepath.ToSlash(sourcePath)
		targetPath := sourcePath
		render := strings.HasSuffix(sourcePath, ".tmpl")
		if render {
			targetPath = strings.TrimSuffix(sourcePath, ".tmpl")
		}
		files = append(files, ProjectFile{
			SourcePath: sourcePath,
			TargetPath: targetPath,
			Render:     render,
		})
		return nil
	})
	return files, err
}

func projectFilesByType(projectType string) ([]ProjectFile, error) {
	set, err := templateSet(projectType)
	if err != nil {
		return nil, err
	}
	return projectFiles(set)
}

func filterProjectFiles(files []ProjectFile, opts Options) []ProjectFile {
	if opts.WithDB {
		return files
	}

	filtered := make([]ProjectFile, 0, len(files))
	for _, file := range files {
		if file.TargetPath == "repo/init.go" {
			continue
		}
		filtered = append(filtered, file)
	}
	return filtered
}

func ensureWritableTarget(target string, force bool) error {
	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(target, 0755)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("target exists and is not a directory: %s", target)
	}
	if isKitRepoRoot(target) {
		return fmt.Errorf("target must not be the kit repository root: %s", target)
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return err
	}
	if len(entries) > 0 && !force {
		return fmt.Errorf("target directory is not empty: %s", target)
	}
	return nil
}

func isKitRepoRoot(target string) bool {
	data, err := os.ReadFile(filepath.Join(target, "go.mod"))
	if err != nil {
		return false
	}
	return hasModulePath(string(data), "github.com/tiamxu/kit")
}

func writeProjectFile(target string, set TemplateSet, file ProjectFile, data TemplateData, force bool) error {
	path := filepath.Join(target, filepath.FromSlash(file.TargetPath))
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("file already exists: %s", path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	content, err := templatesFS.ReadFile(filepath.ToSlash(filepath.Join(set.Root, file.SourcePath)))
	if err != nil {
		return err
	}
	if file.Render {
		tpl, err := template.New(file.SourcePath).Parse(string(content))
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		if err := tpl.Execute(&buf, data); err != nil {
			return err
		}
		content = buf.Bytes()
	}
	return os.WriteFile(path, content, 0644)
}

func writeManifest(target string, opts Options, files []ProjectFile) error {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.TargetPath)
	}
	manifest := Manifest{
		ProjectType: opts.Type,
		ProjectName: opts.Name,
		Module:      opts.Module,
		Files:       paths,
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	path := filepath.Join(target, ".gogen", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
