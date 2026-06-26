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
				"templates/golden/csharp-cli/GreetingBuilder.cs",
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

func TestCSharpCLIProgramPlacesTopLevelStatementsBeforeTypeDeclarations(t *testing.T) {
	repoRoot := repoRoot(t)
	programPath := filepath.Join(repoRoot, "templates", "golden", "csharp-cli", "Program.cs")

	contentBytes, err := os.ReadFile(programPath)
	if err != nil {
		t.Fatalf("read %s: %v", programPath, err)
	}

	content := string(contentBytes)
	topLevelIndex := strings.Index(content, "var target =")
	typeIndex := strings.Index(content, "public static class GreetingBuilder")
	if topLevelIndex == -1 {
		t.Fatalf("program missing expected snippets:\n%s", content)
	}
	if typeIndex == -1 {
		return
	}
	if topLevelIndex > typeIndex {
		t.Fatalf("top-level statements must precede type declarations:\n%s", content)
	}
}

func TestCSharpCLIProjectTemplateExcludesTestSourcesFromRootCompile(t *testing.T) {
	repoRoot := repoRoot(t)
	projectPath := filepath.Join(repoRoot, "templates", "golden", "csharp-cli", "Project.csproj.tmpl")

	contentBytes, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("read %s: %v", projectPath, err)
	}

	content := string(contentBytes)
	if !strings.Contains(content, `<Compile Remove="tests/**/*.cs" />`) {
		t.Fatalf("project template must exclude test sources from root compile:\n%s", content)
	}
}

func TestCSharpCLIStarterFilesCarryStyleCopFileHeaders(t *testing.T) {
	repoRoot := repoRoot(t)

	files := []string{
		filepath.Join(repoRoot, "templates", "golden", "csharp-cli", "Program.cs"),
		filepath.Join(repoRoot, "templates", "golden", "csharp-cli", ".mkproj-overlay", "tests", "Project.Tests", "ProgramTests.cs"),
	}

	for _, path := range files {
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}

		content := string(contentBytes)
		if !strings.Contains(content, `<copyright file=`) {
			t.Fatalf("%s missing StyleCop file header:\n%s", path, content)
		}
	}
}

func TestCSharpCLIProjectFilesSuppressStyleCopHeaderMismatchRule(t *testing.T) {
	repoRoot := repoRoot(t)

	files := []string{
		filepath.Join(repoRoot, "templates", "golden", "csharp-cli", "Project.csproj.tmpl"),
		filepath.Join(repoRoot, "templates", "golden", "csharp-cli", ".mkproj-overlay", "tests", "Project.Tests", "Project.Tests.csproj.tmpl"),
	}

	for _, path := range files {
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}

		content := string(contentBytes)
		if !strings.Contains(content, "SA1636") {
			t.Fatalf("%s missing SA1636 suppression:\n%s", path, content)
		}
	}
}

func TestCSharpCLIProjectFilesEnableXmlDocumentationAnalysis(t *testing.T) {
	repoRoot := repoRoot(t)

	files := []string{
		filepath.Join(repoRoot, "templates", "golden", "csharp-cli", "Project.csproj.tmpl"),
		filepath.Join(repoRoot, "templates", "golden", "csharp-cli", ".mkproj-overlay", "tests", "Project.Tests", "Project.Tests.csproj.tmpl"),
	}

	for _, path := range files {
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}

		content := string(contentBytes)
		if !strings.Contains(content, "<GenerateDocumentationFile>true</GenerateDocumentationFile>") {
			t.Fatalf("%s must enable XML documentation output so StyleCop XML analysis can run:\n%s", path, content)
		}
	}
}

func TestPythonCLITyperStarterTestInvokesHelloSubcommand(t *testing.T) {
	repoRoot := repoRoot(t)
	mainPath := filepath.Join(repoRoot, "templates", "golden", "python-cli-typer", "src", "app", "main.py")
	testPath := filepath.Join(repoRoot, "templates", "golden", "python-cli-typer", ".mkproj-overlay", "tests", "test_cli.py")

	mainBytes, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read %s: %v", mainPath, err)
	}

	testBytes, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("read %s: %v", testPath, err)
	}

	mainContent := string(mainBytes)
	testContent := string(testBytes)
	if strings.Contains(mainContent, `def hello(name: str = "world")`) &&
		!strings.Contains(testContent, `runner.invoke(app, ["--name", "Peter"])`) {
		t.Fatalf("python-cli-typer starter test must pass the defaulted Typer parameter as an option:\n%s", testContent)
	}
}

func TestCSharpWebAPIProjectTemplateExcludesTestSourcesFromRootCompile(t *testing.T) {
	repoRoot := repoRoot(t)
	projectPath := filepath.Join(repoRoot, "templates", "golden", "csharp-webapi", "Project.csproj.tmpl")

	contentBytes, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("read %s: %v", projectPath, err)
	}

	content := string(contentBytes)
	if !strings.Contains(content, `<Compile Remove="tests/**/*.cs" />`) {
		t.Fatalf("webapi project template must exclude test sources from root compile:\n%s", content)
	}
	if !strings.Contains(content, `<Content Remove="tests/**" />`) {
		t.Fatalf("webapi project template must exclude test artifacts from web content discovery:\n%s", content)
	}
}

func TestCSharpWebAPITestProjectReferencesTestHost(t *testing.T) {
	repoRoot := repoRoot(t)
	projectPath := filepath.Join(repoRoot, "templates", "golden", "csharp-webapi", ".mkproj-overlay", "tests", "Project.Tests", "Project.Tests.csproj.tmpl")

	contentBytes, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("read %s: %v", projectPath, err)
	}

	content := string(contentBytes)
	if !strings.Contains(content, `PackageReference Include="Microsoft.AspNetCore.TestHost"`) {
		t.Fatalf("webapi test project must reference Microsoft.AspNetCore.TestHost:\n%s", content)
	}
}

func TestCSharpWebAPIStarterFilesCarryStyleCopFileHeaders(t *testing.T) {
	repoRoot := repoRoot(t)

	files := []string{
		filepath.Join(repoRoot, "templates", "golden", "csharp-webapi", "Program.cs"),
		filepath.Join(repoRoot, "templates", "golden", "csharp-webapi", ".mkproj-overlay", "tests", "Project.Tests", "HealthEndpointTests.cs"),
	}

	for _, path := range files {
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}

		content := string(contentBytes)
		if !strings.Contains(content, `<copyright file=`) {
			t.Fatalf("%s missing StyleCop file header:\n%s", path, content)
		}
	}
}

func TestCSharpWebAPIProjectFilesEnableXmlDocumentationAnalysis(t *testing.T) {
	repoRoot := repoRoot(t)

	files := []string{
		filepath.Join(repoRoot, "templates", "golden", "csharp-webapi", "Project.csproj.tmpl"),
		filepath.Join(repoRoot, "templates", "golden", "csharp-webapi", ".mkproj-overlay", "tests", "Project.Tests", "Project.Tests.csproj.tmpl"),
	}

	for _, path := range files {
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}

		content := string(contentBytes)
		if !strings.Contains(content, "<GenerateDocumentationFile>true</GenerateDocumentationFile>") {
			t.Fatalf("%s must enable XML documentation output so StyleCop XML analysis can run:\n%s", path, content)
		}
	}
}

func TestCSharpWebAPIProjectFilesSuppressStyleCopHeaderMismatchRule(t *testing.T) {
	repoRoot := repoRoot(t)

	files := []string{
		filepath.Join(repoRoot, "templates", "golden", "csharp-webapi", "Project.csproj.tmpl"),
		filepath.Join(repoRoot, "templates", "golden", "csharp-webapi", ".mkproj-overlay", "tests", "Project.Tests", "Project.Tests.csproj.tmpl"),
	}

	for _, path := range files {
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}

		content := string(contentBytes)
		if !strings.Contains(content, "SA1636") {
			t.Fatalf("%s missing SA1636 suppression:\n%s", path, content)
		}
	}
}

func TestCSharpWebAPITestProjectEnablesImplicitUsings(t *testing.T) {
	repoRoot := repoRoot(t)
	projectPath := filepath.Join(repoRoot, "templates", "golden", "csharp-webapi", ".mkproj-overlay", "tests", "Project.Tests", "Project.Tests.csproj.tmpl")

	contentBytes, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("read %s: %v", projectPath, err)
	}

	content := string(contentBytes)
	if !strings.Contains(content, "<ImplicitUsings>enable</ImplicitUsings>") {
		t.Fatalf("webapi test project must enable implicit usings for starter async tests:\n%s", content)
	}
}

func TestCSharpWebAPIHealthEndpointTestUsesStyleCopFriendlyAsyncPattern(t *testing.T) {
	repoRoot := repoRoot(t)
	testPath := filepath.Join(repoRoot, "templates", "golden", "csharp-webapi", ".mkproj-overlay", "tests", "Project.Tests", "HealthEndpointTests.cs")

	contentBytes, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("read %s: %v", testPath, err)
	}

	content := string(contentBytes)
	if !strings.Contains(content, "/// <returns>A task that completes when the assertion finishes.</returns>") {
		t.Fatalf("health endpoint test must document the async return value for StyleCop:\n%s", content)
	}
	if !strings.Contains(content, "Program.ConfigureApp(app);") {
		t.Fatalf("health endpoint test must exercise the shared app configuration entrypoint:\n%s", content)
	}
}

func TestCSharpWebAPIProgramTemplateExposesTestableConstructionHooks(t *testing.T) {
	repoRoot := repoRoot(t)
	programPath := filepath.Join(repoRoot, "templates", "golden", "csharp-webapi", "Program.cs")

	contentBytes, err := os.ReadFile(programPath)
	if err != nil {
		t.Fatalf("read %s: %v", programPath, err)
	}

	content := string(contentBytes)
	for _, snippet := range []string{
		"var builder = Program.CreateBuilder(args);",
		"var app = builder.Build();",
		"Program.ConfigureApp(app);",
		"public static WebApplicationBuilder CreateBuilder(string[] args)",
		"public static void ConfigureApp(WebApplication app)",
	} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("program template missing testable webapi construction hook %q:\n%s", snippet, content)
		}
	}
}

func TestCSharpWebAPIHealthEndpointTestUsesTestServerClient(t *testing.T) {
	repoRoot := repoRoot(t)
	testPath := filepath.Join(repoRoot, "templates", "golden", "csharp-webapi", ".mkproj-overlay", "tests", "Project.Tests", "HealthEndpointTests.cs")

	contentBytes, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("read %s: %v", testPath, err)
	}

	content := string(contentBytes)
	for _, snippet := range []string{
		"using Microsoft.AspNetCore.TestHost;",
		"var builder = Program.CreateBuilder(Array.Empty<string>());",
		"builder.WebHost.UseTestServer();",
		"Program.ConfigureApp(app);",
		"using var client = app.GetTestClient();",
	} {
		if !strings.Contains(content, snippet) {
			t.Fatalf("health endpoint test missing test-server wiring %q:\n%s", snippet, content)
		}
	}
}

func TestGoAPIChiPinsPatchedGoToolchainForVulnerabilityGate(t *testing.T) {
	repoRoot := repoRoot(t)

	files := []string{
		filepath.Join(repoRoot, "templates", "golden", "go-api-chi", "go.mod.tmpl"),
		filepath.Join(repoRoot, "templates", "golden", "go-api-chi", ".mkproj-overlay", "mise.toml"),
	}

	for _, path := range files {
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}

		content := string(contentBytes)
		if !strings.Contains(content, "1.26.4") {
			t.Fatalf("%s must pin a Go patch release that satisfies the vulnerability gate:\n%s", path, content)
		}
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
