package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"mkproj"
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

func TestWriterRendersPythonPackageAndLanguageScopedManifestFromEmbeddedTemplates(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	vars, err := project.ResolveVariables(project.Input{
		ProjectName: "My Cool API",
		Language:    "python",
		ProjectType: "cli",
		Stack:       "python-cli-typer",
		AuthorName:  "Ada Lovelace",
		AuthorEmail: "ada@example.com",
		Remote:      project.RemoteNone,
	})
	if err != nil {
		t.Fatalf("ResolveVariables() error = %v", err)
	}

	writer := Writer{Assets: mkproj.Assets()}
	if err := writer.Write(tempDir, vars); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	assertFileContains(t, filepath.Join(tempDir, "pyproject.toml"), `name = "my_cool_api"`)
	assertFileContains(t, filepath.Join(tempDir, "README.md"), "# My Cool API")
	assertFileContains(t, filepath.Join(tempDir, "CONTEXT.md"), "# Context")
	assertFileContains(t, filepath.Join(tempDir, "docs", "adr", "0000-template.md"), "# ADR 0000")

	manifestPath := filepath.Join(tempDir, ".claude", "skill-manifest.json")
	manifest := readFile(t, manifestPath)
	if strings.Contains(manifest, "golang/golang-cli") {
		t.Fatalf("python manifest should not include go-only skills:\n%s", manifest)
	}
	if !strings.Contains(manifest, "productivity/mise") {
		t.Fatalf("python manifest missing shared skills:\n%s", manifest)
	}
}

func TestWriterRendersCSharpNamespaceIntoProjectFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	vars, err := project.ResolveVariables(project.Input{
		ProjectName: "My Cool API",
		Language:    "csharp",
		ProjectType: "cli",
		Stack:       "csharp-cli",
		AuthorName:  "Ada Lovelace",
		AuthorEmail: "ada@example.com",
		Remote:      project.RemoteNone,
	})
	if err != nil {
		t.Fatalf("ResolveVariables() error = %v", err)
	}

	writer := Writer{Assets: mkproj.Assets()}
	if err := writer.Write(tempDir, vars); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	projectFile := filepath.Join(tempDir, "Project.csproj")
	assertFileContains(t, projectFile, "<AssemblyName>MyCoolApi</AssemblyName>")
	assertFileContains(t, projectFile, "<RootNamespace>MyCoolApi</RootNamespace>")
}

func TestWriterAllowsGitDirectoryFromPreinitializedRepo(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.git) error = %v", err)
	}

	assets := fstest.MapFS{
		"templates/common/AGENTS.md.tmpl":                  {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base":                  {Data: []byte(".DS_Store\n")},
		"templates/common/claude/skill-manifest.json.tmpl": {Data: []byte("{\"skills\":[\"productivity/mise\"]}\n")},
		"templates/common/claude/hooks/secret-scan.sh":     {Data: []byte("#!/usr/bin/env bash\n")},
		"templates/common/codex/hooks.json":                {Data: []byte("{\"hooks\":{}}\n")},
		"templates/gitignore/Go.gitignore":                 {Data: []byte("bin/\n")},
		"templates/golden/go-cli-cobra/main.go.tmpl":       {Data: []byte("package main\n")},
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
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}

	return string(data)
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()

	data := readFile(t, path)
	if !strings.Contains(data, want) {
		t.Fatalf("%s does not contain %q:\n%s", path, want, data)
	}
}

func TestTemplateSecretScanMatchesSharedScanner(t *testing.T) {
	t.Parallel()

	shared := readFile(t, filepath.Join("..", "..", ".claude", "hooks", "secret-scan.sh"))
	template := readFile(t, filepath.Join("..", "..", "templates", "common", "claude", "hooks", "secret-scan.sh"))
	if template != shared {
		t.Fatalf("template secret scanner drifted from shared scanner")
	}
}

func TestTemplateGuardMatchesSharedGuard(t *testing.T) {
	t.Parallel()

	shared := readFile(t, filepath.Join("..", "..", ".claude", "hooks", "guard"))
	template := readFile(t, filepath.Join("..", "..", "templates", "common", "claude", "hooks", "guard"))
	if template != shared {
		t.Fatalf("template guard drifted from shared guard")
	}
}
