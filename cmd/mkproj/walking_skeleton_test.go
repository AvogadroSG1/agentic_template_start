package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type walkingSkeletonStack struct {
	language      string
	projectType   string
	stack         string
	starterTest   string
	starterSource string
	vanillaFile   string
	vanillaSource string
	gitignoreStem string
}

var walkingSkeletonStacks = []walkingSkeletonStack{
	{
		language:      "go",
		projectType:   "cli",
		stack:         "go-cli-cobra",
		starterTest:   "cmd/root_test.go",
		starterSource: "templates/golden/go-cli-cobra/.mkproj-overlay/cmd/root_test.go.tmpl",
		vanillaFile:   "cmd/root.go",
		vanillaSource: "templates/golden/go-cli-cobra/cmd/root.go.tmpl",
		gitignoreStem: "go",
	},
	{
		language:      "go",
		projectType:   "api",
		stack:         "go-api-chi",
		starterTest:   "internal/httpapi/health_test.go",
		starterSource: "templates/golden/go-api-chi/.mkproj-overlay/internal/httpapi/health_test.go.tmpl",
		vanillaFile:   "configs/.keep",
		vanillaSource: "templates/golden/go-api-chi/configs/.keep",
		gitignoreStem: "go",
	},
	{
		language:      "python",
		projectType:   "cli",
		stack:         "python-cli-typer",
		starterTest:   "tests/test_cli.py",
		starterSource: "templates/golden/python-cli-typer/.mkproj-overlay/tests/test_cli.py",
		vanillaFile:   "src/app/main.py",
		vanillaSource: "templates/golden/python-cli-typer/src/app/main.py",
		gitignoreStem: "python",
	},
	{
		language:      "python",
		projectType:   "api",
		stack:         "python-fastapi",
		starterTest:   "tests/test_health.py",
		starterSource: "templates/golden/python-fastapi/.mkproj-overlay/tests/test_health.py",
		vanillaFile:   "app/main.py",
		vanillaSource: "templates/golden/python-fastapi/app/main.py",
		gitignoreStem: "python",
	},
	{
		language:      "csharp",
		projectType:   "cli",
		stack:         "csharp-cli",
		starterTest:   "tests/Project.Tests/ProgramTests.cs",
		starterSource: "templates/golden/csharp-cli/.mkproj-overlay/tests/Project.Tests/ProgramTests.cs",
		vanillaFile:   "Program.cs",
		vanillaSource: "templates/golden/csharp-cli/Program.cs",
		gitignoreStem: "visualstudio",
	},
	{
		language:      "csharp",
		projectType:   "api",
		stack:         "csharp-webapi",
		starterTest:   "tests/Project.Tests/HealthEndpointTests.cs",
		starterSource: "templates/golden/csharp-webapi/.mkproj-overlay/tests/Project.Tests/HealthEndpointTests.cs",
		vanillaFile:   "Program.cs",
		vanillaSource: "templates/golden/csharp-webapi/Program.cs",
		gitignoreStem: "visualstudio",
	},
}

func TestWalkingSkeletonAllShippedStacksScaffoldOfflineAndExposeGateWorkflow(t *testing.T) {
	t.Parallel()

	buildDir, binaryPath := buildWalkingSkeletonBinary(t)
	stubDir := filepath.Join(buildDir, "bin")
	toolsDir := filepath.Join(buildDir, "host-tools")
	logPath := filepath.Join(buildDir, "commands.log")
	pathEnv := writeWalkingSkeletonStubs(t, stubDir, toolsDir)

	for _, stack := range walkingSkeletonStacks {
		stack := stack
		t.Run(stack.stack, func(t *testing.T) {
			targetDir := filepath.Join(buildDir, stack.stack)
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				t.Fatalf("MkdirAll(%s) error = %v", targetDir, err)
			}
			if err := os.WriteFile(logPath, nil, 0o644); err != nil {
				t.Fatalf("WriteFile(%s) error = %v", logPath, err)
			}
			env := append(os.Environ(), "PATH="+pathEnv, "WALKING_LOG="+logPath)

			runWalkingCommand(t, targetDir, env, binaryPath,
				"init",
				"--project-name", "Sample App",
				"--language", stack.language,
				"--project-type", stack.projectType,
				"--stack", stack.stack,
				"--author-name", "Ada Lovelace",
				"--author-email", "ada@example.com",
				"--remote", "none",
			)

			for _, rel := range []string{"mise.toml", "lefthook.yml", ".github/workflows/ci.yml", stack.vanillaFile, stack.starterTest} {
				if _, err := os.Stat(filepath.Join(targetDir, rel)); err != nil {
					t.Fatalf("Stat(%s) error = %v", rel, err)
				}
			}

			assertOutputMatchesRepoAsset(t, targetDir, stack.vanillaFile, stack.vanillaSource)
			assertOutputMatchesRepoAsset(t, targetDir, stack.starterTest, stack.starterSource)
			assertOutputMatchesRepoAsset(t, targetDir, "lefthook.yml", filepath.ToSlash(filepath.Join("templates", "golden", stack.stack, ".mkproj-overlay", "lefthook.yml")))
			assertOutputMatchesRepoAsset(t, targetDir, ".github/workflows/ci.yml", filepath.ToSlash(filepath.Join("templates", "golden", stack.stack, ".mkproj-overlay", ".github", "workflows", "ci.yml")))

			hookData, err := os.ReadFile(filepath.Join(targetDir, "lefthook.yml"))
			if err != nil {
				t.Fatalf("ReadFile(lefthook.yml) error = %v", err)
			}
			if !strings.Contains(string(hookData), "parallel: false") {
				t.Fatalf("lefthook.yml should serialize pre-commit gates:\n%s", string(hookData))
			}

			linkTarget, err := os.Readlink(filepath.Join(targetDir, "CLAUDE.md"))
			if err != nil {
				t.Fatalf("Readlink(CLAUDE.md) error = %v", err)
			}
			if linkTarget != "AGENTS.md" {
				t.Fatalf("CLAUDE.md link target = %q, want %q", linkTarget, "AGENTS.md")
			}

			assertExpectedGitignore(t, targetDir, stack.gitignoreStem)

			ciData, err := os.ReadFile(filepath.Join(targetDir, ".github/workflows/ci.yml"))
			if err != nil {
				t.Fatalf("ReadFile(ci.yml) error = %v", err)
			}
			ciText := string(ciData)
			for _, snippet := range []string{"actions/checkout@v4", "jdx/mise-action@v2", "mise install", "mise run ci"} {
				if !strings.Contains(ciText, snippet) {
					t.Fatalf("ci.yml missing %q:\n%s", snippet, ciText)
				}
			}
			for _, forbidden := range []string{"go test", "pytest", "dotnet test", "ruff", "golangci-lint"} {
				if strings.Contains(ciText, forbidden) {
					t.Fatalf("ci.yml should not inline %q:\n%s", forbidden, ciText)
				}
			}

				runWalkingCommand(t, targetDir, env, filepath.Join(stubDir, "mise"), "install")
				runWalkingCommand(t, targetDir, env, filepath.Join(stubDir, "mise"), "run", "lint")
				runWalkingCommand(t, targetDir, env, filepath.Join(stubDir, "mise"), "run", "test")
				runWalkingCommand(t, targetDir, env, filepath.Join(stubDir, "mise"), "run", "ci")
				assertLogContains(t, logPath, "mise install", "mise run lint", "mise run test", "mise run ci")
				assertLogOmitsNetwork(t, logPath)

				clearWalkingLog(t, logPath)
				runConfiguredHookPhase(t, targetDir, env, "pre-commit")
				assertLogContains(t, logPath, "./.claude/hooks/secret-scan.sh scan-staged", "mise run lint", "mise run fmt")
				assertLogOmitsNetwork(t, logPath)

				clearWalkingLog(t, logPath)
				runConfiguredHookPhase(t, targetDir, env, "pre-push")
				assertLogContains(t, logPath, "mise run test")
				assertLogOmitsNetwork(t, logPath)
			})
		}
	}

func TestWalkingSkeletonGoCLIInitRunsThroughMiseCI(t *testing.T) {
	t.Parallel()

	buildDir, binaryPath := buildWalkingSkeletonBinary(t)

	stubDir := filepath.Join(buildDir, "bin")
	toolsDir := filepath.Join(buildDir, "host-tools")
	if err := os.MkdirAll(stubDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	writeExecutable(t, filepath.Join(stubDir, "bd"), "#!/usr/bin/env bash\nset -euo pipefail\nexit 0\n")
	writeExecutable(t, filepath.Join(stubDir, "instill"), "#!/usr/bin/env bash\nset -euo pipefail\nexit 0\n")
	writeExecutable(t, filepath.Join(stubDir, "lefthook"), "#!/usr/bin/env bash\nset -euo pipefail\nexit 0\n")
	writeExecutable(t, filepath.Join(stubDir, "golangci-lint"), "#!/usr/bin/env bash\nset -euo pipefail\nexit 0\n")
	writeExecutable(t, filepath.Join(stubDir, "govulncheck"), "#!/usr/bin/env bash\nset -euo pipefail\nexit 0\n")
	logPath := filepath.Join(buildDir, "go-cli-real.log")
	if err := os.WriteFile(logPath, nil, 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", logPath, err)
	}
	networkTool := `#!/usr/bin/env bash
set -euo pipefail
if [ -n "${WALKING_LOG:-}" ]; then
  printf 'NETWORK %s %s\n' "$(basename "$0")" "$*" >> "$WALKING_LOG"
fi
exit 97
`
	for _, tool := range []string{"curl", "wget", "ssh", "scp", "rsync", "nc", "ncat", "socat"} {
		writeExecutable(t, filepath.Join(stubDir, tool), networkTool)
	}
	writeHostToolWrappers(t, toolsDir, "env", "bash", "find", "python3", "git", "go", "gofmt")
	writeExecutable(t, filepath.Join(stubDir, "mise"), `#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
  install)
    exit 0
    ;;
  run)
    task="${2:-}"
    if [ -z "$task" ]; then
      echo "missing task" >&2
      exit 64
    fi
    command_line="$(python3 - "$task" <<'PY'
import pathlib
import re
import sys

task = sys.argv[1]
text = pathlib.Path("mise.toml").read_text()
pattern = rf"(?ms)^\[tasks\.{re.escape(task)}\]\s*\nrun = \"([^\"]+)\""
match = re.search(pattern, text)
if not match:
    raise SystemExit(1)
print(match.group(1))
PY
)"
    if [ -n "${WALKING_LOG:-}" ]; then
      printf '%s\n' "$command_line" >> "$WALKING_LOG"
    fi
    bash -c "$command_line"
    ;;
  *)
    echo "unsupported mise command: $*" >&2
    exit 64
    ;;
esac
`)

	targetDir := filepath.Join(buildDir, "sample-app")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	pathEnv := stubDir + string(os.PathListSeparator) + toolsDir

	initCmd := exec.Command(binaryPath,
		"init",
		"--project-name", "Sample App",
		"--language", "go",
		"--project-type", "cli",
		"--stack", "go-cli-cobra",
		"--author-name", "Ada Lovelace",
		"--author-email", "ada@example.com",
		"--remote", "none",
	)
	initCmd.Dir = targetDir
	initCmd.Env = append(os.Environ(), "PATH="+pathEnv, "WALKING_LOG="+logPath, "GOPROXY=off", "GOSUMDB=off")
	if output, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("mkproj init error = %v\n%s", err, output)
	}

	for _, rel := range []string{"mise.toml", "lefthook.yml", ".github/workflows/ci.yml", "cmd/root_test.go"} {
		if _, err := os.Stat(filepath.Join(targetDir, rel)); err != nil {
			t.Fatalf("Stat(%s) error = %v", rel, err)
		}
	}

	installCmd := exec.Command(filepath.Join(stubDir, "mise"), "install")
	installCmd.Dir = targetDir
	installCmd.Env = append(os.Environ(), "PATH="+pathEnv, "WALKING_LOG="+logPath, "GOPROXY=off", "GOSUMDB=off")
	if output, err := installCmd.CombinedOutput(); err != nil {
		t.Fatalf("mise install error = %v\n%s", err, output)
	}

	ciCmd := exec.Command(filepath.Join(stubDir, "mise"), "run", "ci")
	ciCmd.Dir = targetDir
	ciCmd.Env = append(os.Environ(), "PATH="+pathEnv, "WALKING_LOG="+logPath, "GOPROXY=off", "GOSUMDB=off")
	if output, err := ciCmd.CombinedOutput(); err != nil {
		t.Fatalf("mise run ci error = %v\n%s", err, output)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", logPath, err)
	}
	if !strings.Contains(string(logData), "go test ./...") {
		t.Fatalf("real go-cli walking skeleton did not reach go test:\n%s", string(logData))
	}
	if strings.Contains(string(logData), "NETWORK ") {
		t.Fatalf("real go-cli walking skeleton invoked a network stub:\n%s", string(logData))
	}
}

func writeExecutable(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func buildWalkingSkeletonBinary(t *testing.T) (string, string) {
	t.Helper()

	buildDir := t.TempDir()
	binaryPath := filepath.Join(buildDir, "mkproj")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	build := exec.Command("go", "build", "-o", binaryPath, ".")
	build.Dir = "."
	build.Env = append(os.Environ(), "GOCACHE="+filepath.Join(buildDir, "go-build-cache"))
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build error = %v\n%s", err, output)
	}

	return buildDir, binaryPath
}

func writeWalkingSkeletonStubs(t *testing.T, stubDir string, toolsDir string) string {
	t.Helper()

	if err := os.MkdirAll(stubDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", stubDir, err)
	}
	writeHostToolWrappers(t, toolsDir, "env", "bash", "find", "python3", "git")

	genericTool := `#!/usr/bin/env bash
set -euo pipefail
if [ -n "${WALKING_LOG:-}" ]; then
  printf '%s %s\n' "$(basename "$0")" "$*" >> "$WALKING_LOG"
fi
`
	for _, tool := range []string{
		"bd",
		"instill",
		"lefthook",
		"go",
		"gofmt",
		"golangci-lint",
		"govulncheck",
		"ruff",
		"pytest",
		"pyright",
		"pip-audit",
		"dotnet",
	} {
		writeExecutable(t, filepath.Join(stubDir, tool), genericTool+"exit 0\n")
	}

	networkTool := `#!/usr/bin/env bash
set -euo pipefail
if [ -n "${WALKING_LOG:-}" ]; then
  printf 'NETWORK %s %s\n' "$(basename "$0")" "$*" >> "$WALKING_LOG"
fi
exit 97
`
	for _, tool := range []string{"curl", "wget", "ssh", "scp", "rsync", "nc", "ncat", "socat"} {
		writeExecutable(t, filepath.Join(stubDir, tool), networkTool)
	}

	writeExecutable(t, filepath.Join(stubDir, "mise"), `#!/usr/bin/env bash
set -euo pipefail
if [ -n "${WALKING_LOG:-}" ]; then
  printf 'mise %s\n' "$*" >> "$WALKING_LOG"
fi
case "${1:-}" in
  install)
    exit 0
    ;;
  run)
    task="${2:-}"
    if [ -z "$task" ]; then
      echo "missing task" >&2
      exit 64
    fi
    command_line="$(python3 - "$task" <<'PY'
import pathlib
import re
import sys

task = sys.argv[1]
text = pathlib.Path("mise.toml").read_text()
pattern = rf"(?ms)^\[tasks\.{re.escape(task)}\]\s*\nrun = \"([^\"]+)\""
match = re.search(pattern, text)
if not match:
    raise SystemExit(1)
print(match.group(1))
PY
)"
    bash -c "$command_line"
    ;;
  *)
    echo "unsupported mise command: $*" >&2
    exit 64
    ;;
esac
`)

	return stubDir + string(os.PathListSeparator) + toolsDir
}

func runWalkingCommand(t *testing.T, dir string, env []string, command string, args ...string) {
	t.Helper()

	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Env = env
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %s error = %v\n%s", command, strings.Join(args, " "), err, output)
	}
}

func writeHostToolWrappers(t *testing.T, toolsDir string, tools ...string) {
	t.Helper()

	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", toolsDir, err)
	}

	for _, tool := range tools {
		target, err := exec.LookPath(tool)
		if err != nil {
			t.Fatalf("LookPath(%s) error = %v", tool, err)
		}
		writeExecutable(t, filepath.Join(toolsDir, tool), fmt.Sprintf("#!/bin/sh\nexec %q \"$@\"\n", target))
	}
}

func assertOutputMatchesRepoAsset(t *testing.T, targetDir string, outputPath string, sourceRel string) {
	t.Helper()

	got, err := os.ReadFile(filepath.Join(targetDir, outputPath))
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", outputPath, err)
	}
	want, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(sourceRel)))
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", sourceRel, err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s contents differ from %s", outputPath, sourceRel)
	}
}

func assertExpectedGitignore(t *testing.T, targetDir string, gitignoreStem string) {
	t.Helper()

	base, err := os.ReadFile(filepath.Join("..", "..", "templates", "common", "gitignore.base"))
	if err != nil {
		t.Fatalf("ReadFile(gitignore.base) error = %v", err)
	}
	language, err := os.ReadFile(filepath.Join("..", "..", "templates", "gitignore", gitignoreTemplateName(gitignoreStem)+".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile(language gitignore) error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(targetDir, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile(.gitignore) error = %v", err)
	}

	var expected strings.Builder
	expected.Write(base)
	if len(base) > 0 && base[len(base)-1] != '\n' {
		expected.WriteByte('\n')
	}
	expected.WriteString(fmt.Sprintf("# ===== %s gitignore =====\n", gitignoreStem))
	expected.Write(language)

	if gotText := string(got); gotText != expected.String() {
		t.Fatalf(".gitignore mismatch:\n%s", gotText)
	}
}

func runConfiguredHookPhase(t *testing.T, targetDir string, env []string, phase string) {
	t.Helper()

	for _, commandLine := range hookCommandsForPhase(t, targetDir, phase) {
		runShellCommand(t, targetDir, env, commandLine)
	}
}

func gitignoreTemplateName(stem string) string {
	switch stem {
	case "go":
		return "Go"
	case "python":
		return "Python"
	case "visualstudio":
		return "VisualStudio"
	default:
		return stem
	}
}

func hookCommandsForPhase(t *testing.T, targetDir string, phase string) []string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(targetDir, "lefthook.yml"))
	if err != nil {
		t.Fatalf("ReadFile(lefthook.yml) error = %v", err)
	}

	lines := strings.Split(string(data), "\n")
	commands := make([]string, 0, 4)
	inPhase := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(line, " ") && strings.HasSuffix(trimmed, ":") {
			inPhase = strings.TrimSuffix(trimmed, ":") == phase
			continue
		}
		if !inPhase {
			continue
		}
		if strings.HasPrefix(trimmed, "run: ") {
			commands = append(commands, strings.TrimPrefix(trimmed, "run: "))
		}
	}
	if len(commands) == 0 {
		t.Fatalf("%s hook phase did not define any run commands", phase)
	}

	return commands
}

func runShellCommand(t *testing.T, dir string, env []string, commandLine string) {
	t.Helper()

	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Fatalf("LookPath(bash) error = %v", err)
	}
	for _, entry := range env {
		if !strings.HasPrefix(entry, "WALKING_LOG=") {
			continue
		}
		logPath := strings.TrimPrefix(entry, "WALKING_LOG=")
		file, openErr := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
		if openErr != nil {
			t.Fatalf("OpenFile(%s) error = %v", logPath, openErr)
		}
		if _, writeErr := fmt.Fprintf(file, "%s\n", commandLine); writeErr != nil {
			_ = file.Close()
			t.Fatalf("Write(%s) error = %v", logPath, writeErr)
		}
		if closeErr := file.Close(); closeErr != nil {
			t.Fatalf("Close(%s) error = %v", logPath, closeErr)
		}
		break
	}

	cmd := exec.Command(bashPath, "-c", commandLine)
	cmd.Dir = dir
	cmd.Env = env
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("shell command %q error = %v\n%s", commandLine, err, output)
	}
}

func clearWalkingLog(t *testing.T, logPath string) {
	t.Helper()

	if err := os.WriteFile(logPath, nil, 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", logPath, err)
	}
}

func assertLogContains(t *testing.T, logPath string, snippets ...string) {
	t.Helper()

	logText := readWalkingLog(t, logPath)
	for _, snippet := range snippets {
		if !strings.Contains(logText, snippet) {
			t.Fatalf("command log missing %q:\n%s", snippet, logText)
		}
	}
}

func assertLogOmitsNetwork(t *testing.T, logPath string) {
	t.Helper()

	logText := readWalkingLog(t, logPath)
	if strings.Contains(logText, "NETWORK ") {
		t.Fatalf("walking skeleton invoked a network stub:\n%s", logText)
	}
}

func readWalkingLog(t *testing.T, logPath string) string {
	t.Helper()

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", logPath, err)
	}

	return string(logData)
}

func TestWalkingSkeletonFixtureDefinesGoGateTasks(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "templates", "golden", "go-cli-cobra", ".mkproj-overlay", "mise.toml"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	for _, snippet := range []string{"[tasks.fmt]", "[tasks.lint]", "[tasks.test]", "[tasks.ci]", `"github:golangci/golangci-lint" = "latest"`, `"go:golang.org/x/vuln/cmd/govulncheck" = "latest"`, `"ubi:evilmartians/lefthook" = "latest"`, "govulncheck ./..."} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("go-cli-cobra mise.toml missing %q", snippet)
		}
	}
}
