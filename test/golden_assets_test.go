package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourcesYAMLDefinesTheV1GoldenRecipes(t *testing.T) {
	repoRoot := repoRoot(t)
	sourcesPath := filepath.Join(repoRoot, "sources.yaml")

	contentBytes, err := os.ReadFile(sourcesPath)
	if err != nil {
		t.Fatalf("read %s: %v", sourcesPath, err)
	}

	content := string(contentBytes)
	requiredSnippets := []string{
		"gitignore_repo:",
		"go-cli-cobra:\n  kind: scaffolder",
		"go-api-chi:\n  kind: recipe",
		"python-cli-typer:\n  kind: scaffolder",
		"python-fastapi:\n  kind: scaffolder",
		"csharp-cli:\n  kind: scaffolder",
		"csharp-webapi:\n  kind: scaffolder",
		"cobra-cli init --pkg-name {{.ModulePath}}",
		"github.com/golang-standards/project-layout",
		"uv init",
		"dotnet new console",
		"dotnet new webapi",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("sources.yaml missing required snippet %q", snippet)
		}
	}
}

func TestGoldenCatalogPackagesVanillaAndOverlayAssetsForEveryV1Stack(t *testing.T) {
	repoRoot := repoRoot(t)

	tests := []struct {
		name  string
		files []string
	}{
		{
			name: "go-cli-cobra",
			files: []string{
				"templates/golden/go-cli-cobra/go.mod.tmpl",
				"templates/golden/go-cli-cobra/main.go.tmpl",
				"templates/golden/go-cli-cobra/cmd/root.go.tmpl",
				"templates/golden/go-cli-cobra/cmd/serve.go.tmpl",
				"templates/golden/go-cli-cobra/cmd/config.go.tmpl",
				"templates/golden/go-cli-cobra/.mkproj-overlay/cmd/root_test.go.tmpl",
			},
		},
		{
			name: "go-api-chi",
			files: []string{
				"templates/golden/go-api-chi/go.mod.tmpl",
				"templates/golden/go-api-chi/api/.keep",
				"templates/golden/go-api-chi/configs/.keep",
				"templates/golden/go-api-chi/internal/platform/.keep",
				"templates/golden/go-api-chi/pkg/.keep",
				"templates/golden/go-api-chi/.mkproj-overlay/cmd/api/main.go.tmpl",
				"templates/golden/go-api-chi/.mkproj-overlay/internal/httpapi/router.go.tmpl",
				"templates/golden/go-api-chi/.mkproj-overlay/internal/httpapi/health.go.tmpl",
				"templates/golden/go-api-chi/.mkproj-overlay/internal/httpapi/health_test.go.tmpl",
			},
		},
		{
			name: "python-cli-typer",
			files: []string{
				"templates/golden/python-cli-typer/pyproject.toml.tmpl",
				"templates/golden/python-cli-typer/src/app/__init__.py",
				"templates/golden/python-cli-typer/src/app/main.py",
				"templates/golden/python-cli-typer/.mkproj-overlay/tests/test_cli.py",
			},
		},
		{
			name: "python-fastapi",
			files: []string{
				"templates/golden/python-fastapi/pyproject.toml.tmpl",
				"templates/golden/python-fastapi/app/__init__.py",
				"templates/golden/python-fastapi/app/main.py",
				"templates/golden/python-fastapi/.mkproj-overlay/tests/test_health.py",
			},
		},
		{
			name: "csharp-cli",
			files: []string{
				"templates/golden/csharp-cli/Project.csproj.tmpl",
				"templates/golden/csharp-cli/Program.cs",
				"templates/golden/csharp-cli/.mkproj-overlay/tests/Project.Tests/Project.Tests.csproj.tmpl",
				"templates/golden/csharp-cli/.mkproj-overlay/tests/Project.Tests/ProgramTests.cs",
			},
		},
		{
			name: "csharp-webapi",
			files: []string{
				"templates/golden/csharp-webapi/Project.csproj.tmpl",
				"templates/golden/csharp-webapi/Program.cs",
				"templates/golden/csharp-webapi/.mkproj-overlay/tests/Project.Tests/Project.Tests.csproj.tmpl",
				"templates/golden/csharp-webapi/.mkproj-overlay/tests/Project.Tests/HealthEndpointTests.cs",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, relPath := range tt.files {
				fullPath := filepath.Join(repoRoot, relPath)

				info, err := os.Stat(fullPath)
				if err != nil {
					t.Fatalf("stat %s: %v", fullPath, err)
				}
				if info.IsDir() {
					t.Fatalf("%s is a directory, want file", fullPath)
				}
			}
		})
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	return filepath.Clean(filepath.Join(wd, ".."))
}
