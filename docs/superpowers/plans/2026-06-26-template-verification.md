# Template Verification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prove that every shipped stack produces a working project by running real toolchains (scaffold → install → local-release) in an integration test gated by build tag.

**Architecture:** A single table-driven Go test file (`test/local_release_test.go`) behind an `integration` build tag. TestMain builds the mkproj binary once. Each subtest scaffolds a stack into `t.TempDir()`, runs `mise install`, then `mise run ci` with real tools. A CI workflow runs the fast gate (3 CLI stacks) on every PR and the slow gate (all 6) on a daily schedule.

**Tech Stack:** Go 1.24+ test framework, `os/exec` with `context.WithTimeout`, mise toolchains, GitHub Actions.

## Global Constraints

- Build tag: `//go:build integration` — tests MUST NOT run without explicit opt-in.
- Module boundary: test file lives in `test/` package (already exists, package `test`).
- Per-step timeout: 5 minutes (`context.WithTimeout`).
- Global test timeout: 10 minutes (fast), 20 minutes (slow).
- All scaffolds use `--remote none` to avoid network/credential dependencies.
- `t.Fatalf` on failure (not `t.Errorf`) — steps are sequential dependencies.
- Failure output MUST include: step name, command string, exit code, combined output, directory tree.

---

## File Structure

| Action | Path | Responsibility |
|--------|------|---------------|
| Create | `test/local_release_test.go` | Integration test: TestMain + table-driven TestLocalRelease |
| Modify | `Makefile` | Add `verify-fast` and `verify-slow` targets |
| Create | `.github/workflows/verify-templates.yml` | CI workflow: fast gate on PR, slow gate on schedule |

---

### Task 1: Integration Test File

**Files:**
- Create: `test/local_release_test.go`

**Interfaces:**
- Consumes: `mkproj` binary built by TestMain (package-level `var mkprojBinary string`)
- Produces: `TestLocalRelease` with subtests named by stack (`go-cli-cobra`, `python-cli-typer`, etc.)

- [ ] **Step 1: Write the test file with build tag, TestMain, and runStep helper**

```go
//go:build integration

package test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

const stepTimeout = 5 * time.Minute

var mkprojBinary string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "mkproj-verify-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating temp dir: %v\n", err)
		os.Exit(1)
	}

	binaryPath := filepath.Join(tmpDir, "mkproj")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/mkproj")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "building mkproj: %v\n%s\n", err, output)
		os.Exit(1)
	}
	mkprojBinary = binaryPath

	code := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

func runStep(t *testing.T, name string, dir string, command string, args ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), stepTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		exitCode := -1
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		tree, _ := exec.Command("find", dir, "-type", "f").Output()
		t.Fatalf("%s: exit %d (cmd: %s)\n%s\n--- directory tree ---\n%s",
			name, exitCode, cmd.String(), output, tree)
	}
}
```

- [ ] **Step 2: Add the table-driven TestLocalRelease function**

Append to the same file:

```go
func TestLocalRelease(t *testing.T) {
	type stack struct {
		name        string
		language    string
		projectType string
		stack       string
	}

	stacks := []stack{
		{"go-cli-cobra", "go", "cli", "go-cli-cobra"},
		{"python-cli-typer", "python", "cli", "python-cli-typer"},
		{"csharp-cli", "csharp", "cli", "csharp-cli"},
		{"go-api-chi", "go", "api", "go-api-chi"},
		{"python-fastapi", "python", "api", "python-fastapi"},
		{"csharp-webapi", "csharp", "api", "csharp-webapi"},
	}

	for _, s := range stacks {
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			runStep(t, "mkproj init", dir, mkprojBinary,
				"init",
				"--project-name", "Verify "+s.name,
				"--language", s.language,
				"--project-type", s.projectType,
				"--stack", s.stack,
				"--author-name", "CI Bot",
				"--author-email", "ci@example.com",
				"--remote", "none",
			)

			runStep(t, "mise install", dir, "mise", "install")
			runStep(t, "mise run ci", dir, "mise", "run", "ci")
		})
	}
}
```

- [ ] **Step 3: Verify the file compiles with the integration tag**

Run:
```bash
GOCACHE=$PWD/.cache/go-build go vet -tags=integration ./test/
```

Expected: exit 0, no output (clean vet).

- [ ] **Step 4: Commit**

```bash
git add test/local_release_test.go
git commit -m "feat(test): add integration test for template local-release verification

Table-driven test behind //go:build integration that scaffolds each stack
into t.TempDir() and runs mise install + mise run ci with real toolchains.
TestMain builds mkproj once; runStep provides structured failure output."
```

---

### Task 2: Makefile Targets

**Files:**
- Modify: `Makefile`

**Interfaces:**
- Consumes: `test/local_release_test.go` (Task 1)
- Produces: `make verify-fast` (runs 3 CLI stacks), `make verify-slow` (runs all 6)

- [ ] **Step 1: Add verify-fast and verify-slow targets to the Makefile**

Add after the existing `clean` target:

```makefile
verify-fast: build ## Run fast-gate template verification (CLI stacks only)
	GOCACHE=$(CURDIR)/.cache/go-build go test -tags=integration -count=1 \
	  -timeout=10m \
	  -run "TestLocalRelease/(go-cli-cobra|python-cli-typer|csharp-cli)" ./test/

verify-slow: build ## Run slow-gate template verification (all stacks)
	GOCACHE=$(CURDIR)/.cache/go-build go test -tags=integration -count=1 \
	  -timeout=20m -run "TestLocalRelease" ./test/
```

- [ ] **Step 2: Update the .PHONY declaration**

Change:
```makefile
.PHONY: help build test install uninstall clean
```
to:
```makefile
.PHONY: help build test install uninstall clean verify-fast verify-slow
```

- [ ] **Step 3: Verify Make parses correctly**

Run:
```bash
make help
```

Expected: output includes `verify-fast` and `verify-slow` with their help text.

- [ ] **Step 4: Commit**

```bash
git add Makefile
git commit -m "feat(make): add verify-fast and verify-slow targets

verify-fast runs the 3 CLI stacks (fast gate, blocks PR).
verify-slow runs all 6 stacks (slow gate, daily schedule)."
```

---

### Task 3: GitHub Actions Workflow

**Files:**
- Create: `.github/workflows/verify-templates.yml`

**Interfaces:**
- Consumes: `make verify-fast` and `make verify-slow` (Task 2)
- Produces: CI jobs `fast-gate` (on push/PR) and `slow-gate` (on schedule/dispatch)

- [ ] **Step 1: Create the .github/workflows directory if needed**

```bash
mkdir -p .github/workflows
```

- [ ] **Step 2: Write the workflow file**

```yaml
name: Verify Templates

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  schedule:
    - cron: '17 4 * * *'
  workflow_dispatch:

jobs:
  fast-gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: jdx/mise-action@v2
      - run: make verify-fast

  slow-gate:
    if: github.event_name == 'schedule' || github.event_name == 'workflow_dispatch'
    runs-on: ubuntu-latest
    strategy:
      fail-fast: false
      matrix:
        stack: [go-cli-cobra, python-cli-typer, csharp-cli, go-api-chi, python-fastapi, csharp-webapi]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: jdx/mise-action@v2
      - run: |
          GOCACHE=$PWD/.cache/go-build go test -tags=integration -count=1 \
            -timeout=10m -run "TestLocalRelease/${{ matrix.stack }}" ./test/
```

- [ ] **Step 3: Validate YAML syntax**

Run:
```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/verify-templates.yml'))"
```

Expected: exit 0, no output.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/verify-templates.yml
git commit -m "ci: add verify-templates workflow

Fast gate (3 CLI stacks) runs on push/PR and blocks merge.
Slow gate (all 6 stacks) runs daily at 4:17 UTC and on manual dispatch.
Each slow-gate stack runs as a separate matrix job for isolation."
```

---

### Task 4: Local Smoke Test (go-cli-cobra only)

**Files:**
- None created/modified — this task validates that Task 1-2 work end-to-end.

**Interfaces:**
- Consumes: Everything from Tasks 1-3.
- Produces: Confidence that at least one stack passes locally.

- [ ] **Step 1: Run the go-cli-cobra subtest locally**

```bash
GOCACHE=$PWD/.cache/go-build go test -tags=integration -count=1 \
  -timeout=10m -v -run "TestLocalRelease/go-cli-cobra" ./test/
```

Expected: `--- PASS: TestLocalRelease/go-cli-cobra` in output.

- [ ] **Step 2: If it fails, diagnose using the structured output**

The failure message includes: step name, exit code, command string, full output, and directory tree. Use these to identify the root cause.

Common failure modes:
- `mkproj init: exit 1` → generator bug (check if `--remote none` flag is recognized)
- `mise install: exit 1` → toolchain resolution (check `mise.toml` in the scaffolded project)
- `mise run ci: exit 1/2` → template behavioral bug (check which sub-task failed in output)

- [ ] **Step 3: Once passing, run the full fast gate**

```bash
make verify-fast
```

Expected: All 3 CLI stacks pass. If any fail, fix the template and re-run.

- [ ] **Step 4: Commit any template fixes discovered during validation**

```bash
git add -A
git commit -m "fix(templates): resolve issues found during verification smoke test"
```

(Only if fixes were needed. Skip this step if all stacks passed on first run.)
