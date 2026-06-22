package scaffold

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"mkproj"
)

var shippedStacks = []string{
	"go-cli-cobra",
	"go-api-chi",
	"python-cli-typer",
	"python-fastapi",
	"csharp-cli",
	"csharp-webapi",
}

func TestShippedStacksCarryGatePipelineAssets(t *testing.T) {
	t.Parallel()

	assets := mkproj.Assets()
	for _, stack := range shippedStacks {
		stack := stack
		t.Run(stack, func(t *testing.T) {
			t.Parallel()

			for _, rel := range []string{
				".mkproj-overlay/mise.toml",
				".mkproj-overlay/lefthook.yml",
				".mkproj-overlay/.github/workflows/ci.yml",
			} {
				assetPath := filepath.ToSlash(filepath.Join("templates/golden", stack, rel))
				if _, err := fs.Stat(assets, assetPath); err != nil {
					t.Fatalf("fs.Stat(%s) error = %v", assetPath, err)
				}
			}
		})
	}
}

func TestShippedStacksDefineGateTasksAndHookCallers(t *testing.T) {
	t.Parallel()

	assets := mkproj.Assets()
	for _, stack := range shippedStacks {
		stack := stack
		t.Run(stack, func(t *testing.T) {
			t.Parallel()

			misePath := filepath.ToSlash(filepath.Join("templates/golden", stack, ".mkproj-overlay", "mise.toml"))
			miseData, err := fs.ReadFile(assets, misePath)
			if err != nil {
				t.Fatalf("ReadFile(%s) error = %v", misePath, err)
			}
			miseText := string(miseData)
			for _, section := range []string{"[tasks.fmt]", "[tasks.lint]", "[tasks.test]", "[tasks.ci]"} {
				if !strings.Contains(miseText, section) {
					t.Fatalf("%s missing %s", misePath, section)
				}
			}

			hookPath := filepath.ToSlash(filepath.Join("templates/golden", stack, ".mkproj-overlay", "lefthook.yml"))
			hookData, err := fs.ReadFile(assets, hookPath)
			if err != nil {
				t.Fatalf("ReadFile(%s) error = %v", hookPath, err)
			}
			hookText := string(hookData)
			for _, snippet := range []string{"scan-staged", "mise run lint", "mise run fmt", "mise run test"} {
				if !strings.Contains(hookText, snippet) {
					t.Fatalf("%s missing %q", hookPath, snippet)
				}
			}
		})
	}
}

func TestSharedCIWorkflowOnlyDelegatesToMise(t *testing.T) {
	t.Parallel()

	assets := mkproj.Assets()
	for _, stack := range shippedStacks {
		stack := stack
		t.Run(stack, func(t *testing.T) {
			t.Parallel()

			ciPath := filepath.ToSlash(filepath.Join("templates/golden", stack, ".mkproj-overlay", ".github", "workflows", "ci.yml"))
			ciData, err := fs.ReadFile(assets, ciPath)
			if err != nil {
				t.Fatalf("ReadFile(%s) error = %v", ciPath, err)
			}
			ciText := string(ciData)
			for _, snippet := range []string{"actions/checkout@v4", "jdx/mise-action@v2", "mise install", "mise run ci"} {
				if !strings.Contains(ciText, snippet) {
					t.Fatalf("%s missing %q", ciPath, snippet)
				}
			}
			for _, forbidden := range []string{"go test", "pytest", "dotnet test", "ruff", "golangci-lint"} {
				if strings.Contains(ciText, forbidden) {
					t.Fatalf("%s should not inline %q", ciPath, forbidden)
				}
			}
		})
	}
}
