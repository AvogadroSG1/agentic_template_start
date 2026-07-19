# Python Package Directory Naming Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace hardcoded `app/` directories in Python templates with `{{.PythonPackage}}/` so the scaffolded source directory matches the project's derived package name.

**Architecture:** Add a `renderPath` function to the scaffold writer that resolves Go template expressions in embedded FS paths before writing. Then rename `app/` directories in templates to `{{.PythonPackage}}/`, update file content references, add validation, and fix the `update` command's reverse-mapping logic.

**Tech Stack:** Go 1.24, `text/template`, `embed.FS`, `testing/fstest`

## Global Constraints

- All existing tests MUST continue passing after each task (run `GOCACHE=$PWD/.cache/go-build go test ./... -count=1`)
- Template expressions in paths use Go `text/template` syntax: `{{.PythonPackage}}`
- The `strings.Contains(path, "{{")` fast-path ensures zero overhead for non-Python stacks
- Rendered paths containing `..` MUST be rejected (path traversal guard)
- The `embed.FS` uses forward slashes regardless of OS
- `mapOutputPath` (dot-prefix rewriting) MUST run AFTER `renderPath` (template expansion)

---

### Task 1: Add `renderPath` to scaffold writer

**Files:**
- Modify: `internal/scaffold/writer.go:138` (insert `renderPath` call before `mapOutputPath`)
- Modify: `internal/scaffold/writer.go` (add `renderPath` function after `mapOutputPath` at line 179)
- Test: `internal/scaffold/writer_test.go` (add unit test)

**Interfaces:**
- Consumes: `project.Variables` struct (unchanged)
- Produces: `renderPath(path string, vars project.Variables) (string, error)` — renders `{{.Var}}` expressions in a path string, returns error on parse/render failure or path traversal

- [ ] **Step 1: Write the failing test**

Add to `internal/scaffold/writer_test.go`:

```go
func TestRenderPathExpandsTemplateExpressions(t *testing.T) {
	t.Parallel()

	vars := project.Variables{PythonPackage: "my_cool_api"}

	got, err := renderPath("{{.PythonPackage}}/__init__.py", vars)
	if err != nil {
		t.Fatalf("renderPath() error = %v", err)
	}
	if got != "my_cool_api/__init__.py" {
		t.Fatalf("renderPath() = %q, want %q", got, "my_cool_api/__init__.py")
	}
}

func TestRenderPathPassesThroughPlainPaths(t *testing.T) {
	t.Parallel()

	vars := project.Variables{PythonPackage: "my_cool_api"}

	got, err := renderPath("claude/settings.json", vars)
	if err != nil {
		t.Fatalf("renderPath() error = %v", err)
	}
	if got != "claude/settings.json" {
		t.Fatalf("renderPath() = %q, want %q", got, "claude/settings.json")
	}
}

func TestRenderPathRejectsTraversalAttempts(t *testing.T) {
	t.Parallel()

	vars := project.Variables{PythonPackage: ".."}

	_, err := renderPath("{{.PythonPackage}}/main.py", vars)
	if err == nil {
		t.Fatal("renderPath() expected error for traversal, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/scaffold/ -run TestRenderPath -v -count=1`
Expected: FAIL with "undefined: renderPath"

- [ ] **Step 3: Implement `renderPath`**

Add to `internal/scaffold/writer.go` after the `mapOutputPath` function (after line 179):

```go
func renderPath(path string, vars project.Variables) (string, error) {
	if !strings.Contains(path, "{{") {
		return path, nil
	}
	tmpl, err := template.New("path:" + path).Option("missingkey=error").Parse(path)
	if err != nil {
		return "", fmt.Errorf("template path %q: parse: %w", path, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("template path %q: render: %w", path, err)
	}
	rendered := buf.String()
	if strings.Contains(rendered, "..") {
		return "", fmt.Errorf("template path %q: rendered to unsafe value %q", path, rendered)
	}
	return rendered, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/scaffold/ -run TestRenderPath -v -count=1`
Expected: PASS (3 tests)

- [ ] **Step 5: Wire `renderPath` into `copyTree`**

In `internal/scaffold/writer.go`, replace line 138:

```go
// Before (line 138):
targetPath := filepath.Join(targetDir, destPrefix, mapOutputPath(path))

// After:
renderedPath, err := renderPath(path, vars)
if err != nil {
    return err
}
targetPath := filepath.Join(targetDir, destPrefix, mapOutputPath(renderedPath))
```

- [ ] **Step 6: Run full test suite**

Run: `GOCACHE=$PWD/.cache/go-build go test ./... -count=1`
Expected: ALL PASS (no paths contain `{{` yet, so behavior is unchanged)

- [ ] **Step 7: Commit**

```bash
git add internal/scaffold/writer.go internal/scaffold/writer_test.go
git commit -m "feat(scaffold): add renderPath for template expressions in directory paths

Enables {{.Variable}} expansion in embedded FS paths before writing.
No behavior change yet — wired but dormant until template dirs are renamed.

Co-Authored-By: Peter O'Connor <poconnor@stackoverflow.com>
Co-Authored-By: Claude Code <noreply@anthropic.com> - databricks-claude-opus-4-6"
```

---

### Task 2: Add `ValidatePythonPackage` with tests

**Files:**
- Modify: `internal/project/project.go` (add validation function and regex)
- Test: `internal/project/project_test.go` (add validation test cases)

**Interfaces:**
- Consumes: a package name string
- Produces: `ValidatePythonPackage(name string) error` — returns nil for valid Python identifiers, error otherwise

- [ ] **Step 1: Write the failing test**

Add to `internal/project/project_test.go`:

```go
func TestValidatePythonPackage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid underscore separated", input: "my_cool_api", wantErr: false},
		{name: "valid single word", input: "app", wantErr: false},
		{name: "valid leading underscore", input: "_private_pkg", wantErr: false},
		{name: "invalid starts with digit", input: "3d_printer", wantErr: true},
		{name: "invalid empty", input: "", wantErr: true},
		{name: "invalid uppercase", input: "MyPackage", wantErr: true},
		{name: "invalid hyphen", input: "my-package", wantErr: true},
		{name: "invalid dot", input: "my.package", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePythonPackage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePythonPackage(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/project/ -run TestValidatePythonPackage -v -count=1`
Expected: FAIL with "undefined: ValidatePythonPackage"

- [ ] **Step 3: Implement `ValidatePythonPackage`**

Add to `internal/project/project.go` (after the existing `nonAlphaNumeric` regex at line 67):

```go
var validPythonIdentifier = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

func ValidatePythonPackage(name string) error {
	if name == "" {
		return fmt.Errorf("python package name is empty")
	}
	if !validPythonIdentifier.MatchString(name) {
		return fmt.Errorf("python package name %q is not a valid Python identifier (must match [a-z_][a-z0-9_]*)", name)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/project/ -run TestValidatePythonPackage -v -count=1`
Expected: PASS (8 subtests)

- [ ] **Step 5: Run full test suite**

Run: `GOCACHE=$PWD/.cache/go-build go test ./... -count=1`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/project/project.go internal/project/project_test.go
git commit -m "feat(project): add ValidatePythonPackage identifier validation

Ensures derived or overridden python package names are valid Python
identifiers (lowercase + underscores only, cannot start with a digit).

Co-Authored-By: Peter O'Connor <poconnor@stackoverflow.com>
Co-Authored-By: Claude Code <noreply@anthropic.com> - databricks-claude-opus-4-6"
```

---

### Task 3: Wire validation into prompt resolution and add `PythonPackageOverride`

**Files:**
- Modify: `internal/project/project.go:18-32` (add `PythonPackageOverride` to `Input`)
- Modify: `internal/project/project.go:129` (apply override in `ResolveVariables`)
- Modify: `internal/prompt/prompt.go` (validate derived name, prompt if invalid)
- Test: `internal/project/project_test.go` (test override behavior)

**Interfaces:**
- Consumes: `Input.PythonPackageOverride` (new field), `ValidatePythonPackage` (from Task 2)
- Produces: Updated `ResolveVariables` that respects overrides; updated `Resolve` that validates/prompts for Python package names

- [ ] **Step 1: Write the failing test for override**

Add to `internal/project/project_test.go`:

```go
func TestResolveVariablesAppliesPythonPackageOverride(t *testing.T) {
	t.Parallel()

	vars, err := ResolveVariables(Input{
		ProjectName:           "StackOverflow.CostInvestigator",
		Language:              "python",
		ProjectType:           "service",
		Stack:                 "python-fastapi",
		AuthorName:            "Ada Lovelace",
		AuthorEmail:           "ada@example.com",
		Remote:                RemoteNone,
		PythonPackageOverride: "cost_investigator",
	})
	if err != nil {
		t.Fatalf("ResolveVariables() error = %v", err)
	}
	if vars.PythonPackage != "cost_investigator" {
		t.Fatalf("PythonPackage = %q, want %q", vars.PythonPackage, "cost_investigator")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/project/ -run TestResolveVariablesAppliesPythonPackageOverride -v -count=1`
Expected: FAIL (field `PythonPackageOverride` does not exist)

- [ ] **Step 3: Add `PythonPackageOverride` to `Input` struct**

In `internal/project/project.go`, add to the `Input` struct (after `BdPrefix`):

```go
type Input struct {
	ProjectName           string
	Language              string
	ProjectType           string
	Stack                 string
	Frontend              string
	APIBaseURL            string
	AuthorName            string
	AuthorEmail           string
	GitHubUser            string
	Remote                RemoteKind
	RemoteURL             string
	ModulePath            string
	BdPrefix              string
	PythonPackageOverride string
}
```

- [ ] **Step 4: Apply override in `ResolveVariables`**

In `internal/project/project.go`, replace line 129:

```go
// Before (line 129):
PythonPackage:   strings.Join(slugWords, "_"),

// After:
PythonPackage:   derivePythonPackage(slugWords, input.PythonPackageOverride),
```

Add helper function after `ResolveVariables`:

```go
func derivePythonPackage(slugWords []string, override string) string {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return trimmed
	}
	return strings.Join(slugWords, "_")
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/project/ -run TestResolveVariablesAppliesPythonPackageOverride -v -count=1`
Expected: PASS

- [ ] **Step 6: Wire validation into prompt resolution**

In `internal/prompt/prompt.go`, after the stack is resolved and the language is known to be Python, add validation of the derived package name. Find the end of the `Resolve` function (before the final return) and add:

```go
if resolved.Language == "python" {
	pythonPkg := derivedPythonPackage(resolved.ProjectName)
	if err := project.ValidatePythonPackage(pythonPkg); err != nil {
		override, promptErr := resolveValue(input.PythonPackageOverride, "python-package",
			"Derived package name is invalid. Enter a valid Python package name:", nil, "", input.IsTTY, prompter)
		if promptErr != nil {
			return project.Input{}, promptErr
		}
		if err := project.ValidatePythonPackage(override); err != nil {
			return project.Input{}, fmt.Errorf("python package override %q: %w", override, err)
		}
		resolved.PythonPackageOverride = override
	}
}
```

Where `derivedPythonPackage` is a local helper:

```go
func derivedPythonPackage(projectName string) string {
	words := strings.Fields(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(projectName), " "))
	return strings.Join(words, "_")
}
```

Note: The exact integration point depends on the existing `Resolve` function structure. The field `PythonPackageOverride` needs to be threaded from `Inputs` to `project.Input`. Add `PythonPackageOverride string` to the `Inputs` struct in prompt.go, and set `resolved.PythonPackageOverride` from it.

- [ ] **Step 7: Run full test suite**

Run: `GOCACHE=$PWD/.cache/go-build go test ./... -count=1`
Expected: ALL PASS

- [ ] **Step 8: Commit**

```bash
git add internal/project/project.go internal/project/project_test.go internal/prompt/prompt.go
git commit -m "feat(prompt): validate python package name, prompt for override if invalid

Adds PythonPackageOverride to Input and applies it in ResolveVariables.
When running interactively and the derived name is invalid, prompts the
user for a valid package name.

Co-Authored-By: Peter O'Connor <poconnor@stackoverflow.com>
Co-Authored-By: Claude Code <noreply@anthropic.com> - databricks-claude-opus-4-6"
```

---

### Task 4: Restructure template directories and update file content

**Files:**
- Rename: `templates/golden/python-fastapi/app/` → `templates/golden/python-fastapi/{{.PythonPackage}}/`
- Rename: `templates/golden/python-cli-typer/src/app/` → `templates/golden/python-cli-typer/src/{{.PythonPackage}}/`
- Rename: `templates/golden/python-web-jinja/app/` → `templates/golden/python-web-jinja/{{.PythonPackage}}/`
- Rename: `templates/golden/python-web-jinja/.forge-overlay/app/` → `templates/golden/python-web-jinja/.forge-overlay/{{.PythonPackage}}/`
- Modify: `templates/golden/python-fastapi/pyproject.toml.tmpl` (update packages line)
- Modify: `templates/golden/python-cli-typer/pyproject.toml.tmpl` (update packages and scripts)
- Rename+Modify: `templates/golden/python-fastapi/.forge-overlay/tests/test_health.py` → `test_health.py.tmpl`
- Rename+Modify: `templates/golden/python-cli-typer/.forge-overlay/tests/test_cli.py` → `test_cli.py.tmpl`
- Rename+Modify: `templates/golden/python-web-jinja/.forge-overlay/tests/test_health.py` → `test_health.py.tmpl`
- Rename+Modify: `templates/golden/python-web-jinja/.forge-overlay/tests/test_index.py` → `test_index.py.tmpl`
- Delete: `templates/golden/python-cli-typer/src/app/__pycache__/main.cpython-314.pyc`

**Interfaces:**
- Consumes: `renderPath` from Task 1 (now active — paths contain `{{`)
- Produces: Template directories that scaffold to `<package_name>/` instead of `app/`

- [ ] **Step 1: Delete stale `__pycache__` artifact**

```bash
rm -f templates/golden/python-cli-typer/src/app/__pycache__/main.cpython-314.pyc
rmdir templates/golden/python-cli-typer/src/app/__pycache__/
```

- [ ] **Step 2: Rename `app/` directories to `{{.PythonPackage}}/`**

```bash
# python-fastapi: app/ -> {{.PythonPackage}}/
mv templates/golden/python-fastapi/app "templates/golden/python-fastapi/{{.PythonPackage}}"

# python-cli-typer: src/app/ -> src/{{.PythonPackage}}/
mv templates/golden/python-cli-typer/src/app "templates/golden/python-cli-typer/src/{{.PythonPackage}}"

# python-web-jinja: app/ -> {{.PythonPackage}}/
mv templates/golden/python-web-jinja/app "templates/golden/python-web-jinja/{{.PythonPackage}}"

# python-web-jinja overlay: .forge-overlay/app/ -> .forge-overlay/{{.PythonPackage}}/
mv "templates/golden/python-web-jinja/.forge-overlay/app" "templates/golden/python-web-jinja/.forge-overlay/{{.PythonPackage}}"
```

- [ ] **Step 3: Update `python-fastapi/pyproject.toml.tmpl`**

Replace the `[tool.hatch.build.targets.wheel]` section:

```toml
[tool.hatch.build.targets.wheel]
packages = ["{{.PythonPackage}}"]
```

- [ ] **Step 4: Update `python-cli-typer/pyproject.toml.tmpl`**

Replace the `[tool.hatch.build.targets.wheel]` and `[project.scripts]` sections:

```toml
[tool.hatch.build.targets.wheel]
packages = ["src/{{.PythonPackage}}"]

[project.scripts]
{{.PythonPackage}} = "{{.PythonPackage}}.main:main"
```

- [ ] **Step 5: Update `python-web-jinja/pyproject.toml.tmpl`**

Replace the `[tool.hatch.build.targets.wheel]` section:

```toml
[tool.hatch.build.targets.wheel]
packages = ["{{.PythonPackage}}"]
```

- [ ] **Step 6: Convert test files to `.tmpl` with package reference**

**python-fastapi** — rename and update:

```bash
mv templates/golden/python-fastapi/.forge-overlay/tests/test_health.py \
   templates/golden/python-fastapi/.forge-overlay/tests/test_health.py.tmpl
```

Write `test_health.py.tmpl`:
```python
from fastapi.testclient import TestClient

from {{.PythonPackage}}.main import app


def test_health_endpoint_walks() -> None:
    client = TestClient(app)

    response = client.get("/health")

    assert response.status_code == 200
    assert response.json() == {"status": "ok"}
```

**python-cli-typer** — rename and update:

```bash
mv templates/golden/python-cli-typer/.forge-overlay/tests/test_cli.py \
   templates/golden/python-cli-typer/.forge-overlay/tests/test_cli.py.tmpl
```

Write `test_cli.py.tmpl`:
```python
from typer.testing import CliRunner

from {{.PythonPackage}}.main import app


def test_hello_command_walks() -> None:
    runner = CliRunner()

    result = runner.invoke(app, ["--name", "Peter"])

    assert result.exit_code == 0
    assert "hello, Peter!" in result.stdout
```

**python-web-jinja test_health** — rename and update:

```bash
mv templates/golden/python-web-jinja/.forge-overlay/tests/test_health.py \
   templates/golden/python-web-jinja/.forge-overlay/tests/test_health.py.tmpl
```

Write `test_health.py.tmpl`:
```python
from fastapi.testclient import TestClient

from {{.PythonPackage}}.main import app


def test_health_endpoint_walks() -> None:
    client = TestClient(app)

    response = client.get("/health")

    assert response.status_code == 200
    assert response.json() == {"status": "ok"}
```

**python-web-jinja test_index** — rename and update:

```bash
mv templates/golden/python-web-jinja/.forge-overlay/tests/test_index.py \
   templates/golden/python-web-jinja/.forge-overlay/tests/test_index.py.tmpl
```

Write `test_index.py.tmpl`:
```python
from fastapi.testclient import TestClient

from {{.PythonPackage}}.main import app


def test_index_page_renders_the_skeleton() -> None:
    client = TestClient(app)

    response = client.get("/")

    assert response.status_code == 200
    assert "API health" in response.text
    assert "/static/htmx.min.js" in response.text


def test_htmx_is_served_locally() -> None:
    client = TestClient(app)

    response = client.get("/static/htmx.min.js")

    assert response.status_code == 200
```

- [ ] **Step 7: Run full test suite**

Run: `GOCACHE=$PWD/.cache/go-build go test ./... -count=1`
Expected: ALL PASS — the writer now renders `{{.PythonPackage}}` in directory paths via `renderPath`, so scaffolded output writes to e.g. `my_cool_api/` instead of `app/`

- [ ] **Step 8: Commit**

```bash
git add -A templates/golden/python-fastapi/ templates/golden/python-cli-typer/ templates/golden/python-web-jinja/
git commit -m "feat(templates): rename app/ to {{.PythonPackage}}/ in all Python stacks

Directory paths now use template expressions resolved at scaffold time.
Test files converted to .tmpl with dynamic import paths.
Removes stale __pycache__ artifact from python-cli-typer.

Co-Authored-By: Peter O'Connor <poconnor@stackoverflow.com>
Co-Authored-By: Claude Code <noreply@anthropic.com> - databricks-claude-opus-4-6"
```

---

### Task 5: Add `applyTemplatePlaceholdersToPath` in update command

**Files:**
- Modify: `internal/update/update.go:334` (add path placeholder call after `committedRelPath` assignment)
- Modify: `internal/update/update.go` (add `applyTemplatePlaceholdersToPath` function)
- Test: `internal/update/update_test.go` (add unit test)

**Interfaces:**
- Consumes: `project.Variables` struct, `committedRelPath` string
- Produces: `applyTemplatePlaceholdersToPath(relPath string, vars project.Variables) string` — replaces exact path segments matching variable values with their template placeholders

- [ ] **Step 1: Write the failing test**

Add to `internal/update/update_test.go`:

```go
func TestApplyTemplatePlaceholdersToPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		path   string
		vars   project.Variables
		want   string
	}{
		{
			name: "replaces python package segment",
			path: "my_cool_api/__init__.py",
			vars: project.Variables{PythonPackage: "my_cool_api"},
			want: "{{.PythonPackage}}/__init__.py",
		},
		{
			name: "replaces nested python package segment",
			path: "src/my_cool_api/main.py",
			vars: project.Variables{PythonPackage: "my_cool_api"},
			want: "src/{{.PythonPackage}}/main.py",
		},
		{
			name: "replaces csharp namespace segment",
			path: "MyCoolApi/Program.cs",
			vars: project.Variables{CSharpNamespace: "MyCoolApi"},
			want: "{{.CSharpNamespace}}/Program.cs",
		},
		{
			name: "does not replace substring matches",
			path: "my_cool_api_extra/main.py",
			vars: project.Variables{PythonPackage: "my_cool_api"},
			want: "my_cool_api_extra/main.py",
		},
		{
			name: "leaves non-matching paths unchanged",
			path: "tests/test_health.py",
			vars: project.Variables{PythonPackage: "my_cool_api"},
			want: "tests/test_health.py",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := applyTemplatePlaceholdersToPath(tt.path, tt.vars)
			if got != tt.want {
				t.Errorf("applyTemplatePlaceholdersToPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/update/ -run TestApplyTemplatePlaceholdersToPath -v -count=1`
Expected: FAIL with "undefined: applyTemplatePlaceholdersToPath"

- [ ] **Step 3: Implement `applyTemplatePlaceholdersToPath`**

Add to `internal/update/update.go` (near the existing `applyTemplatePlaceholders` function around line 540):

```go
func applyTemplatePlaceholdersToPath(relPath string, vars project.Variables) string {
	type replacement struct {
		value       string
		placeholder string
	}
	replacements := []replacement{
		{value: vars.PythonPackage, placeholder: "{{.PythonPackage}}"},
		{value: vars.CSharpNamespace, placeholder: "{{.CSharpNamespace}}"},
	}
	segments := strings.Split(relPath, "/")
	for i, seg := range segments {
		for _, r := range replacements {
			if r.value != "" && seg == r.value {
				segments[i] = r.placeholder
				break
			}
		}
	}
	return strings.Join(segments, "/")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/update/ -run TestApplyTemplatePlaceholdersToPath -v -count=1`
Expected: PASS (5 subtests)

- [ ] **Step 5: Wire into `snapshotVanilla`**

In `internal/update/update.go`, after line 334 (`committedRelPath := filepath.ToSlash(relPath)`), add:

```go
committedRelPath = applyTemplatePlaceholdersToPath(committedRelPath, vars)
```

This ensures directory segments like `my_cool_api/` are reverse-mapped to `{{.PythonPackage}}/` in the snapshot output, so that `forge update` round-trips correctly.

- [ ] **Step 6: Run full test suite**

Run: `GOCACHE=$PWD/.cache/go-build go test ./... -count=1`
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add internal/update/update.go internal/update/update_test.go
git commit -m "feat(update): reverse-map package directories to template placeholders

Adds applyTemplatePlaceholdersToPath which uses exact segment matching
to replace rendered directory names (e.g. my_cool_api/) back to their
template expressions (e.g. {{.PythonPackage}}/). Prevents forge update
from regressing template directories to literal values.

Co-Authored-By: Peter O'Connor <poconnor@stackoverflow.com>
Co-Authored-By: Claude Code <noreply@anthropic.com> - databricks-claude-opus-4-6"
```

---

### Task 6: Update integration test and add end-to-end assertion

**Files:**
- Modify: `internal/scaffold/writer_test.go:101-132` (update `TestWriterRendersPythonPackageFromEmbeddedTemplates`)
- Modify: `internal/scaffold/writer_test.go` (add new integration test for FastAPI stack)

**Interfaces:**
- Consumes: `forge.Assets()`, `project.ResolveVariables`, `Writer.Write`
- Produces: Updated test assertions proving directory rendering works end-to-end

- [ ] **Step 1: Update existing Python test assertions**

In `internal/scaffold/writer_test.go`, update `TestWriterRendersPythonPackageFromEmbeddedTemplates` (line 101-132). After the existing assertions, add:

```go
// Verify package directory is rendered (not literal app/ or {{.PythonPackage}}/)
pkgDir := filepath.Join(tempDir, "src", "my_cool_api")
if _, err := os.Stat(pkgDir); err != nil {
	t.Fatalf("expected rendered package dir %q to exist: %v", pkgDir, err)
}
initFile := filepath.Join(pkgDir, "__init__.py")
if _, err := os.Stat(initFile); err != nil {
	t.Fatalf("expected %q to exist: %v", initFile, err)
}

// Verify literal app/ does NOT exist
literalApp := filepath.Join(tempDir, "src", "app")
if _, err := os.Stat(literalApp); err == nil {
	t.Fatal("literal src/app/ directory should not exist after rendering")
}

// Verify literal template expression does NOT exist in output
templateLiteral := filepath.Join(tempDir, "src", "{{.PythonPackage}}")
if _, err := os.Stat(templateLiteral); err == nil {
	t.Fatal("literal {{.PythonPackage}} directory should not exist in output")
}
```

- [ ] **Step 2: Add FastAPI integration test**

Add to `internal/scaffold/writer_test.go`:

```go
func TestWriterRendersPythonFastAPIPackageDirectory(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	vars, err := project.ResolveVariables(project.Input{
		ProjectName: "Cost Investigator",
		Language:    "python",
		ProjectType: "service",
		Stack:       "python-fastapi",
		AuthorName:  "Ada Lovelace",
		AuthorEmail: "ada@example.com",
		Remote:      project.RemoteNone,
	})
	if err != nil {
		t.Fatalf("ResolveVariables() error = %v", err)
	}

	writer := Writer{Assets: forge.Assets()}
	if err := writer.Write(tempDir, vars); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Package directory uses derived name
	pkgDir := filepath.Join(tempDir, "cost_investigator")
	if _, err := os.Stat(pkgDir); err != nil {
		t.Fatalf("expected package dir %q: %v", pkgDir, err)
	}
	assertFileContains(t, filepath.Join(pkgDir, "__init__.py"), "")

	// No literal app/ directory
	if _, err := os.Stat(filepath.Join(tempDir, "app")); err == nil {
		t.Fatal("literal app/ should not exist")
	}

	// pyproject.toml references the package name
	assertFileContains(t, filepath.Join(tempDir, "pyproject.toml"), `packages = ["cost_investigator"]`)

	// test file has correct import
	assertFileContains(t, filepath.Join(tempDir, "tests", "test_health.py"), "from cost_investigator.main import app")
}
```

- [ ] **Step 3: Run the tests**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/scaffold/ -run "TestWriterRendersPython" -v -count=1`
Expected: PASS (both tests)

- [ ] **Step 4: Run full test suite**

Run: `GOCACHE=$PWD/.cache/go-build go test ./... -count=1`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/scaffold/writer_test.go
git commit -m "test(scaffold): assert directory rendering for Python stacks end-to-end

Verifies that {{.PythonPackage}} directories resolve to the derived name,
app/ no longer appears in output, and test imports reference the correct
package.

Co-Authored-By: Peter O'Connor <poconnor@stackoverflow.com>
Co-Authored-By: Claude Code <noreply@anthropic.com> - databricks-claude-opus-4-6"
```

---

### Task 7: Final verification and cleanup

**Files:**
- Verify: All files modified in Tasks 1-6
- Build: `cmd/forge/main.go`

**Interfaces:**
- Consumes: Everything from Tasks 1-6
- Produces: Verified green build with no regressions

- [ ] **Step 1: Build the binary**

Run: `go build ./cmd/forge`
Expected: Clean build, no errors

- [ ] **Step 2: Run full test suite with race detector**

Run: `GOCACHE=$PWD/.cache/go-build go test -race ./... -count=1`
Expected: ALL PASS, no race conditions

- [ ] **Step 3: Verify no stale `app/` references in templates**

```bash
find templates/golden -type d -name "app" 2>/dev/null
```

Expected: No output (no `app/` directories remain)

- [ ] **Step 4: Verify no stale `__pycache__` artifacts**

```bash
find templates/ -name "*.pyc" -o -name "__pycache__" 2>/dev/null
```

Expected: No output

- [ ] **Step 5: Spot-check with a manual scaffold (optional smoke test)**

```bash
tmpdir=$(mktemp -d)
./forge init --project-name "Hello World" --language python --project-type service --stack python-fastapi --author-name "Test" --author-email "test@test.com" --remote none "$tmpdir"
ls "$tmpdir"/hello_world/
cat "$tmpdir"/tests/test_health.py | grep "from hello_world"
rm -rf "$tmpdir"
```

Expected: `hello_world/` directory exists with `__init__.py` and `main.py`; test imports `from hello_world.main import app`

- [ ] **Step 6: Final commit (if any fixups needed)**

Only needed if Steps 1-5 reveal issues. Otherwise, Task 7 produces no commit.
