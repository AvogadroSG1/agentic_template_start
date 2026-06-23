# Task 6 Report: Maintainer Refresh Seam

## What I implemented
- Added repo-root `sources.yaml` repinning to `internal/update.Run` so maintainer refreshes now update the mutable recipe file after the vanilla write succeeds.
- Added vanilla change detection so `resolved.captured` follows last-changed semantics: it advances only when the refreshed vanilla snapshot differs from the committed vanilla layer.
- Preserved no-op stability by keeping the existing `captured` value when the refreshed vanilla output is byte-identical.
- Kept seam behavior intact: orphan overlay paths still hard-fail before writes, collisions still warn while leaving overlays untouched.
- Kept the `sources.yaml` row shape clean by omitting zero-value step and normalize fields during repin.

## What I tested and results
- `GOCACHE=$PWD/.cache/go-build go test ./internal/update -count=1`
  - PASS
- `GOCACHE=$PWD/.cache/go-build go test ./internal/update ./cmd/mkproj -count=1`
  - PASS
- `GOCACHE=$PWD/.cache/go-build go test ./... -count=1`
  - PASS

## TDD Evidence
### RED
Command:
```bash
GOCACHE=$PWD/.cache/go-build go test ./internal/update -count=1
```
Relevant failing output before implementation:
```text
--- FAIL: TestRunRepinsMutableSourcesWhenRepresentativeRefreshChanges (0.04s)
    update_test.go:299: sources.yaml = "go-cli-cobra:\n  kind: scaffolder\n  steps:\n    - run: \"cobra-cli init --pkg-name {{.ModulePath}}\"\n    - run: \"cobra-cli add serve\"\n    - run: \"cobra-cli add config\"\n  gitignore: Go\n  normalize:\n    - type: line_endings\n    - type: trailing_newline\n    - type: sort_files\n  resolved:\n    ref: \"v1.3.0\"\n    captured: \"2026-06-20\"\n", want updated captured date
FAIL
```
Why expected:
- The new test proved the missing contract gap: refresh changed the vanilla snapshot, but the mutable repo-root `sources.yaml` still kept the old `captured` date because `internal/update/update.go` did not repin the file.

### GREEN
Commands:
```bash
GOCACHE=$PWD/.cache/go-build go test ./internal/update -count=1
GOCACHE=$PWD/.cache/go-build go test ./internal/update ./cmd/mkproj -count=1
GOCACHE=$PWD/.cache/go-build go test ./... -count=1
```
Relevant passing output after implementation:
```text
ok   	mkproj/internal/update	0.558s
ok   	mkproj/internal/update	0.304s
ok   	mkproj/cmd/mkproj	8.045s
ok   	mkproj/cmd/mkproj	8.160s
ok   	mkproj/internal/allowlist	0.660s
ok   	mkproj/internal/catalog	0.982s
ok   	mkproj/internal/init	3.948s
ok   	mkproj/internal/project	1.119s
ok   	mkproj/internal/prompt	1.956s
ok   	mkproj/internal/remote	1.465s
ok   	mkproj/internal/scaffold	1.678s
ok   	mkproj/internal/update	0.960s
ok   	mkproj/test	10.903s
```

## Files changed
- `internal/update/update.go`
- `internal/update/update_test.go`
- `.superpowers/sdd/task-6-report.md`

## Self-review findings
- The repin logic now updates `sources.yaml` only after the seam checks and vanilla write succeed, so orphan failures still write nothing.
- The mutable row writer preserves the existing `captured` value on a no-op refresh, which keeps the second run byte-stable.
- The YAML encoder now omits empty step and normalize fields so the repo-root row stays within the intended recipe schema.

## Concerns
- I left the existing untracked `templates/golden/recipe-stack/keep.txt` artifact untouched per instruction. The final focused and full test runs passed without depending on it, so I did not spend time cleaning it up in this pass.
