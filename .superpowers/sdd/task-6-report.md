# Task 6 Report: Update integration test and add end-to-end assertion

## Status: DONE

## What was implemented

1. Extended `TestWriterRendersPythonPackageFromEmbeddedTemplates` in
   `internal/scaffold/writer_test.go` (stack `python-cli-typer`, project
   "My Cool API") with assertions that:
   - `src/my_cool_api/` (the rendered package directory) exists.
   - `src/my_cool_api/__init__.py` exists.
   - `src/app/` (the old literal directory name) does NOT exist.
   - `src/{{.PythonPackage}}/` (the unrendered template literal) does NOT
     exist.

2. Added a new test, `TestWriterRendersPythonFastAPIPackageDirectory`, for
   the `python-fastapi` stack (project "Cost Investigator" → package
   `cost_investigator`), asserting:
   - `cost_investigator/` package directory exists with `__init__.py`.
   - literal `app/` does not exist.
   - `pyproject.toml` contains `packages = ["cost_investigator"]`.
   - `tests/test_health.py` contains
     `from cost_investigator.main import app`.

Both tests use the real embedded assets via `forge.Assets()`, so they
exercise `renderPath` end-to-end against the actual `templates/golden/`
snapshots (not fixture `fstest.MapFS` data).

## Test results

```
GOCACHE=$PWD/.cache/go-build go test ./internal/scaffold/ -run "TestWriterRendersPython" -v -count=1
```
Result: 2 passed (`TestWriterRendersPythonPackageFromEmbeddedTemplates`,
`TestWriterRendersPythonFastAPIPackageDirectory`)

```
GOCACHE=$PWD/.cache/go-build go test ./... -count=1
```
Result: 314 passed in 13 packages (no regressions)

## Commits

The test edits landed via the environment's automatic checkpoint commit:
- `6e88d57` checkpoint: edit internal/scaffold/writer_test.go [claude-auto]
  (+63 lines: the new assertions in the existing test plus the new
  `TestWriterRendersPythonFastAPIPackageDirectory` test)

No additional commit was created for `internal/scaffold/writer_test.go`
since the working tree for that file was already clean/committed at the
time of this report (the checkpoint mechanism committed it automatically
after the edit). This report file itself is committed separately.

## Self-review notes

- Confirmed the brief's exact test code was inserted verbatim (both the
  added assertions in Step 1 and the new function in Step 2).
- Verified via `find` that the embedded template layout matches what the
  new assertions expect:
  - `templates/golden/python-cli-typer/src/{{.PythonPackage}}/__init__.py`
  - `templates/golden/python-fastapi/{{.PythonPackage}}/__init__.py`
  - `templates/golden/python-fastapi/pyproject.toml.tmpl` has
    `packages = ["{{.PythonPackage}}"]`
  - `templates/golden/python-fastapi/.forge-overlay/tests/test_health.py.tmpl`
    has `from {{.PythonPackage}}.main import app`
- No production code (`internal/scaffold/writer.go`) was touched — this
  task was test-only, as scoped.

## Concerns

None. Both targeted tests and the full suite pass with no regressions.
