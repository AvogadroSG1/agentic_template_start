package project

import "testing"

func TestResolveVariablesDerivesCanonicalNames(t *testing.T) {
	vars, err := ResolveVariables(Input{
		ProjectName: "My Cool API",
		Language:    "Go",
		ProjectType: "API",
		Stack:       "go-api-chi",
		AuthorName:  "Ada Lovelace",
		AuthorEmail: "ada@example.com",
		GitHubUser:  "octocat",
		Remote:      RemoteGH,
	})
	if err != nil {
		t.Fatalf("ResolveVariables() error = %v", err)
	}

	if vars.BdPrefix != "mycoolapi" {
		t.Fatalf("BdPrefix = %q, want %q", vars.BdPrefix, "mycoolapi")
	}
	if vars.ModulePath != "github.com/octocat/my-cool-api" {
		t.Fatalf("ModulePath = %q, want %q", vars.ModulePath, "github.com/octocat/my-cool-api")
	}
	if vars.PythonPackage != "my_cool_api" {
		t.Fatalf("PythonPackage = %q, want %q", vars.PythonPackage, "my_cool_api")
	}
	if vars.CSharpNamespace != "MyCoolApi" {
		t.Fatalf("CSharpNamespace = %q, want %q", vars.CSharpNamespace, "MyCoolApi")
	}
	if vars.RepoSlug != "my-cool-api" {
		t.Fatalf("RepoSlug = %q, want %q", vars.RepoSlug, "my-cool-api")
	}
}

func TestResolveVariablesUsesPlaceholderModulePathWithoutGitHubRemote(t *testing.T) {
	vars, err := ResolveVariables(Input{
		ProjectName: "CLI Helper",
		Language:    "python",
		ProjectType: "cli",
		Stack:       "python-cli-typer",
		AuthorName:  "Ada Lovelace",
		AuthorEmail: "ada@example.com",
		Remote:      RemoteNone,
	})
	if err != nil {
		t.Fatalf("ResolveVariables() error = %v", err)
	}

	if vars.ModulePath != "github.com/your-org/cli-helper" {
		t.Fatalf("ModulePath = %q, want placeholder path", vars.ModulePath)
	}
}

func TestResolveVariablesRequiresThePromptSeedValues(t *testing.T) {
	_, err := ResolveVariables(Input{
		ProjectName: "   ",
		Language:    "go",
		ProjectType: "cli",
		Stack:       "go-cli-cobra",
		AuthorName:  "Ada Lovelace",
		AuthorEmail: "ada@example.com",
	})
	if err == nil {
		t.Fatal("ResolveVariables() error = nil, want error")
	}
}
