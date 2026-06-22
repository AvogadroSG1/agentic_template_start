package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"mkproj/internal/project"
)

func TestWriterComposesCommonVanillaAndOverlayAssets(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	assets := fstest.MapFS{
		"templates/common/AGENTS.md.tmpl":                              {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base":                              {Data: []byte(".DS_Store\n")},
		"templates/common/claude/settings.local.json.tmpl":             {Data: []byte("{\"project\":\"{{.ProjectName}}\"}\n")},
		"templates/common/claude/hooks/secret-scan.sh":                 {Data: []byte("#!/usr/bin/env bash\n")},
		"templates/common/codex/hooks.json":                            {Data: []byte("{\"hooks\":{}}\n")},
		"templates/gitignore/Go.gitignore":                             {Data: []byte("bin/\n")},
		"templates/golden/go-cli-cobra/main.go.tmpl":                   {Data: []byte("package main\n\nconst Name = \"{{.ProjectName}}\"\n")},
		"templates/golden/go-cli-cobra/.mkproj-overlay/main.go":        {Data: []byte("package main\n\nconst Overlay = true\n")},
		"templates/golden/go-cli-cobra/.mkproj-overlay/README.md.tmpl": {Data: []byte("# {{.ProjectName}}\n")},
	}

	vars, err := project.ResolveVariables(project.Input{
		ProjectName: "Sample App",
		Language:    "go",
		ProjectType: "cli",
		Stack:       "go-cli-cobra",
		AuthorName:  "Ada Lovelace",
		AuthorEmail: "ada@example.com",
		Remote:      project.RemoteNone,
	})
	if err != nil {
		t.Fatalf("ResolveVariables() error = %v", err)
	}

	writer := Writer{Assets: assets}
	if err := writer.Write(tempDir, vars); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	assertFileContains(t, filepath.Join(tempDir, "AGENTS.md"), "Project Sample App")
	assertFileContains(t, filepath.Join(tempDir, "README.md"), "# Sample App")
	assertFileContains(t, filepath.Join(tempDir, "main.go"), "const Overlay = true")
	assertFileContains(t, filepath.Join(tempDir, ".gitignore"), ".DS_Store")
	assertFileContains(t, filepath.Join(tempDir, ".gitignore"), "# ===== go gitignore =====")
	assertFileContains(t, filepath.Join(tempDir, ".gitignore"), "bin/")

	linkTarget, err := os.Readlink(filepath.Join(tempDir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if linkTarget != "AGENTS.md" {
		t.Fatalf("CLAUDE.md link target = %q, want %q", linkTarget, "AGENTS.md")
	}

	info, err := os.Stat(filepath.Join(tempDir, ".claude", "hooks", "secret-scan.sh"))
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("secret-scan.sh mode = %#o, want %#o", info.Mode().Perm(), fs.FileMode(0o755))
	}
}

func TestEnsureEmptyDirRejectsNonEmptyDirectories(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "existing.txt"), []byte("boom"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := ensureEmptyDir(tempDir)
	if err == nil || !strings.Contains(err.Error(), "directory not empty") {
		t.Fatalf("ensureEmptyDir() error = %v, want directory not empty", err)
	}
}

func TestWriterFailsBeforeWritingPartialTemplatesOnMissingVariables(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	assets := fstest.MapFS{
		"templates/common/gitignore.base":              {Data: []byte(".DS_Store\n")},
		"templates/common/AGENTS.md.tmpl":              {Data: []byte("{{.Undefined}}\n")},
		"templates/gitignore/Go.gitignore":             {Data: []byte("bin/\n")},
		"templates/golden/go-cli-cobra/main.go":        {Data: []byte("package main\n")},
		"templates/common/claude/hooks/secret-scan.sh": {Data: []byte("#!/usr/bin/env bash\n")},
		"templates/common/codex/hooks.json":            {Data: []byte("{\"hooks\":{}}\n")},
	}

	writer := Writer{Assets: assets}
	err := writer.Write(tempDir, project.Variables{
		ProjectName: "Broken",
		Language:    "go",
		Stack:       "go-cli-cobra",
	})
	if err == nil || !strings.Contains(err.Error(), "Undefined") {
		t.Fatalf("Write() error = %v, want missing key error", err)
	}

	if _, statErr := os.Stat(filepath.Join(tempDir, "AGENTS.md")); !os.IsNotExist(statErr) {
		t.Fatalf("AGENTS.md stat error = %v, want not exists", statErr)
	}
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s does not contain %q:\n%s", path, want, data)
	}
}
