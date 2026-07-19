# Python Package Directory Naming

## Problem Statement

All three Python stacks (python-fastapi, python-cli-typer, python-web-jinja) hardcode `app/` as the package directory. The `PythonPackage` variable (derived from project name, e.g. "StackOverflow.CostInvestigator" -> "stackoverflow_costinvestigator") is only used in pyproject.toml's `name` field. Users need the actual source directory to match their system name so imports read naturally and the installed package name matches the directory layout.

## Solution: Template Expressions in Directory Paths

### Phase 1: Scaffold Writer Enhancement

**File:** `internal/scaffold/writer.go`

Add a `renderPath` function and wire it into `copyTree` before `mapOutputPath`.

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

**Insertion point:** Line 138 of `copyTree`, replacing the current `targetPath` computation:

```go
// Before:
// targetPath := filepath.Join(targetDir, destPrefix, mapOutputPath(path))

// After:
renderedPath, err := renderPath(path, vars)
if err != nil {
    return err
}
targetPath := filepath.Join(targetDir, destPrefix, mapOutputPath(renderedPath))
```

The full relative path is rendered as a single template string (not per-segment). `text/template` handles `/` inside template input; the embedded FS always uses forward slashes. The `strings.Contains(path, "{{")` fast-path means zero overhead for non-Python stacks.

### Phase 2: Template Directory Restructure

| Stack | Before | After |
|-------|--------|-------|
| python-fastapi | `app/` | `{{.PythonPackage}}/` |
| python-cli-typer | `src/app/` | `src/{{.PythonPackage}}/` |
| python-web-jinja | `app/` | `{{.PythonPackage}}/` |
| python-web-jinja overlay | `.forge-overlay/app/` | `.forge-overlay/{{.PythonPackage}}/` |

Contents of each directory stay the same (`__init__.py`, `main.py`, etc.).

Also delete stale artifact: `templates/golden/python-cli-typer/src/app/__pycache__/main.cpython-314.pyc`

### Phase 3: File Content Updates

**python-fastapi/pyproject.toml.tmpl:**
```toml
[tool.hatch.build.targets.wheel]
packages = ["{{.PythonPackage}}"]
```

**python-cli-typer/pyproject.toml.tmpl:**
```toml
[tool.hatch.build.targets.wheel]
packages = ["src/{{.PythonPackage}}"]

[project.scripts]
{{.PythonPackage}} = "{{.PythonPackage}}.main:main"
```

**python-web-jinja/pyproject.toml.tmpl:**
```toml
[tool.hatch.build.targets.wheel]
packages = ["{{.PythonPackage}}"]
```

**Test files (convert to .tmpl):**

- `python-fastapi/.forge-overlay/tests/test_health.py` -> `test_health.py.tmpl`
- `python-cli-typer/.forge-overlay/tests/test_cli.py` -> `test_cli.py.tmpl`
- `python-web-jinja/.forge-overlay/tests/test_health.py` -> `test_health.py.tmpl`
- `python-web-jinja/.forge-overlay/tests/test_index.py` -> `test_index.py.tmpl`

Each replaces `from app.main import app` with `from {{.PythonPackage}}.main import app`.

**No changes needed:**
- `python-web-jinja/.forge-overlay/{{.PythonPackage}}/main.py` -- uses `Path(__file__)` for path resolution
- `python-fastapi/{{.PythonPackage}}/main.py` -- no self-referencing imports
- `python-cli-typer/src/{{.PythonPackage}}/main.py` -- no self-referencing imports

### Phase 4: Validation

**File:** `internal/project/project.go`

```go
var validPythonIdentifier = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

func ValidatePythonPackage(name string) error {
    if name == "" {
        return fmt.Errorf("python package name is empty")
    }
    if !validPythonIdentifier.MatchString(name) {
        return fmt.Errorf("python package name %q is not a valid Python identifier (must start with a letter or underscore)", name)
    }
    return nil
}
```

**File:** `internal/project/project.go` - `Input` struct:

Add `PythonPackageOverride string` field. In `ResolveVariables`:
```go
pythonPkg := strings.Join(slugWords, "_")
if strings.TrimSpace(input.PythonPackageOverride) != "" {
    pythonPkg = strings.TrimSpace(input.PythonPackageOverride)
}
```

**File:** `internal/prompt/prompt.go`

After the project name is resolved and language is Python, validate the derived name. If invalid and `IsTTY`, prompt for a valid package name. If not TTY, return a clear error. The prompt mechanism already exists in this file.

### Phase 5: Update Command Compatibility

**File:** `internal/update/update.go`

The `snapshotVanilla` function (line ~334) copies rendered output back as templates. It must reverse-map output directories to template expressions. After computing `committedRelPath`:

```go
committedRelPath = applyTemplatePlaceholdersToPath(committedRelPath, vars)
```

New function:
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

Uses exact segment matching (not substring replacement) to avoid corrupting paths where the package name appears as a substring of a longer directory name.

The `checkOverlaySeam` function compares paths within the same committed template tree, so both sides contain `{{.PythonPackage}}` literally -- no rendering needed there.

### Phase 6: Testing Strategy

**Unit test (writer_test.go) - focused MapFS test:**

Test that `{{.PythonPackage}}/` directories are rendered to the actual package name (e.g. `my_cool_api/`) and that the literal `{{.PythonPackage}}` directory does NOT appear in output.

**Integration test update:** Update existing `TestWriterRendersPythonPackageFromEmbeddedTemplates` to assert:
- The rendered package directory exists (e.g. `src/my_cool_api/`)
- The literal `app/` directory does NOT exist

**Validation tests (project_test.go):**
- `ValidatePythonPackage("3d_printer")` returns error
- `ValidatePythonPackage("_private_pkg")` succeeds
- `ValidatePythonPackage("my_cool_api")` succeeds
- `ValidatePythonPackage("")` returns error

**Update command test:** Verify `applyTemplatePlaceholdersToPath` correctly reverse-maps rendered directory names back to template expressions.

## Implementation Order

1. Add `renderPath` to `writer.go` and wire into `copyTree` (all existing tests pass since no paths contain `{{` yet)
2. Add `ValidatePythonPackage` to `project.go` with tests
3. Wire validation into `prompt.go` (add `PythonPackageOverride` to `Input`)
4. Restructure template directories (rename `app/` to `{{.PythonPackage}}/`)
5. Update pyproject.toml.tmpl files and convert test files to .tmpl
6. Update `internal/update/update.go` with `applyTemplatePlaceholdersToPath`
7. Update and add tests
8. Delete stale `__pycache__` artifact

## Risk Assessment

- **embed.FS compatibility:** `{{` and `}}` in path names work in Go's embed.FS.
- **Performance:** The `strings.Contains(path, "{{")` fast-path means zero overhead for non-Python stacks.
- **Breaking change surface:** The `update` command changes (Phase 5) are required for maintainer-side refresh to round-trip correctly. Without Phase 5, `forge update` would regress template directories to literal `app/`.
- **Partial write on error:** A rendering failure mid-walk could leave a partially-written output directory. The existing recovery pattern (init.go's `failWithRecovery` tells the user to delete and retry) applies unchanged.
