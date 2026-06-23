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
- The repin logic now plans and applies the mutable `sources.yaml` change before the vanilla swap, then rolls it back if the committed vanilla replacement fails, so seam and write failures still leave the repo consistent.
- The mutable row writer preserves the existing `captured` value on a no-op refresh, which keeps the second run byte-stable.
- The YAML encoder now omits empty step and normalize fields so the repo-root row stays within the intended recipe schema.

## Concerns
- I left the existing untracked `templates/golden/recipe-stack/keep.txt` artifact untouched per instruction. The final focused and full test runs passed without depending on it, so I did not spend time cleaning it up in this pass.


## Review Fix Wave
- Narrowed mutable `sources.yaml` repin to a text-preserving `captured` line update, so a no-op refresh leaves custom quoting and inline normalize formatting untouched.
- Moved repin into a planned apply/rollback path ahead of the vanilla swap, so refresh aborts before touching `templates/golden/<stack>` when `sources.yaml` planning fails, and rolls back `sources.yaml` if the subsequent vanilla replacement fails.
- Strengthened the seam proof with three additional cases:
  - no-op refresh preserves custom-styled `sources.yaml` byte-for-byte
  - orphan failure leaves `sources.yaml` untouched as well as vanilla files
  - collision emits the required warning on `stderr`
- Reframed the repin-failure proof around invalid mutable `sources.yaml` planning rather than a flaky filesystem permission trigger.

### Review Fix RED
Command:
```bash
GOCACHE=$PWD/.cache/go-build go test ./internal/update -count=1
```
Relevant failing output before the fix wave:
```text
--- FAIL: TestRunPreservesCustomStyledSourcesOnNoOpRefresh (0.04s)
    update_test.go:352: sources.yaml changed on no-op refresh
--- FAIL: TestRunLeavesVanillaUntouchedWhenSourcesRepinFails (0.05s)
    update_test.go:504: root.go.tmpl changed after repin failure
FAIL
```
Why expected:
- The first failure proved the repo-root `sources.yaml` repin still churned formatting and quoting on a no-op refresh.
- The second failure proved the write ordering still let committed vanilla content change even when the mutable `sources.yaml` repin path failed.

### Review Fix Verification
Commands:
```bash
GOCACHE=$PWD/.cache/go-build go test ./internal/update -count=1
GOCACHE=$PWD/.cache/go-build go test ./internal/update ./cmd/mkproj -count=1
GOCACHE=$PWD/.cache/go-build go test ./... -count=1
```
Results:
```text
ok   mkproj/internal/update
ok   mkproj/internal/update
ok   mkproj/cmd/mkproj
ok   mkproj/cmd/mkproj
ok   mkproj/internal/allowlist
ok   mkproj/internal/catalog
ok   mkproj/internal/init
ok   mkproj/internal/project
ok   mkproj/internal/prompt
ok   mkproj/internal/remote
ok   mkproj/internal/scaffold
ok   mkproj/internal/update
ok   mkproj/test
```

## Final Fix Wave
- Expanded the refresh catalog contract beyond `.tmpl` files so committed directory sentinels such as `.keep` and `.gitkeep` survive checkout-based refreshes when the refreshed vanilla tree still contains the corresponding directory.
- Added an end-to-end checkout refresh proof that snapshots into `templates/golden/recipe-stack/` and verifies vendored behavior: stripped checkout noise stays out, stale vanilla is removed, and the committed `.keep` sentinel remains in place.
- Replaced the implicit wall-clock capture with an explicit UTC-normalized date source via `nowUTC` plus `currentDateStringAt`, and added a focused unit test that proves the date rolls forward by UTC rather than local time zone.
- Kept the existing seam proofs intact while removing the deadlock-prone first draft of the injected clock override.

### Final Fix RED
Command:
```bash
GOCACHE=$PWD/.cache/go-build go test ./internal/update -count=1
```
Relevant failing output before the final implementation:
```text
# mkproj/internal/update [mkproj/internal/update.test]
internal/update/update_test.go:422:20: undefined: nowUTC
internal/update/update_test.go:423:2: undefined: nowUTC
internal/update/update_test.go:427:3: undefined: nowUTC
FAIL	mkproj/internal/update [build failed]
FAIL
```
Why expected:
- The new UTC-stability proof referenced an explicit clock seam that did not exist yet, so the package failed to compile until `internal/update/update.go` exposed the injected date source.

### Final Fix Verification
Commands:
```bash
GOCACHE=$PWD/.cache/go-build go test ./internal/update -count=1
GOCACHE=$PWD/.cache/go-build go test ./internal/update ./cmd/mkproj -count=1
GOCACHE=$PWD/.cache/go-build go test ./... -count=1
```
Results:
```text
ok   mkproj/internal/update	0.681s
ok   mkproj/internal/update	0.326s
ok   mkproj/cmd/mkproj	8.216s
ok   mkproj/cmd/mkproj	9.227s
ok   mkproj/internal/allowlist	1.318s
ok   mkproj/internal/catalog	0.359s
ok   mkproj/internal/init	4.071s
ok   mkproj/internal/project	0.680s
ok   mkproj/internal/prompt	0.916s
ok   mkproj/internal/remote	1.143s
ok   mkproj/internal/scaffold	0.555s
ok   mkproj/internal/update	2.012s
ok   mkproj/test	12.835s
```


## Final Review Fix Wave
- Preserved non-template catalog sentinels during refresh by carrying committed `.keep` / `.gitkeep` files forward when the refreshed vanilla tree still materializes the owning directory.
- Added a checkout-based end-to-end refresh proof that snapshots a vendored recipe stack back into `templates/golden/<stack>` and verifies stripped files, stale-file removal, and sentinel preservation.
- Made `resolved.captured` UTC-stable through an explicit `nowUTC` seam plus a direct UTC test, so the date logic no longer depends on the maintainer's local timezone.

### Final Review Fix Verification
Commands:
```bash
GOCACHE=$PWD/.cache/go-build go test ./internal/update -count=1
GOCACHE=$PWD/.cache/go-build go test ./internal/update ./cmd/mkproj -count=1
GOCACHE=$PWD/.cache/go-build go test ./... -count=1
```
Results:
```text
ok   mkproj/internal/update
ok   mkproj/internal/update
ok   mkproj/cmd/mkproj
ok   mkproj/cmd/mkproj
ok   mkproj/internal/allowlist
ok   mkproj/internal/catalog
ok   mkproj/internal/init
ok   mkproj/internal/project
ok   mkproj/internal/prompt
ok   mkproj/internal/remote
ok   mkproj/internal/scaffold
ok   mkproj/internal/update
ok   mkproj/test
```

## Final Bounded Fix Wave
- Made the current maintainer refresh scope explicit: `internal/update` now supports Go stacks only, covering the representative `go-cli-cobra` seam and the vendored Go recipe path. Non-Go stacks fail clearly before any workspace creation or repo writes.
- Replaced the in-place vanilla rewrite with a staged copy-and-swap flow that copies refreshed vanilla into a sibling temp tree, copies the committed `.mkproj-overlay/` byte-for-byte into that staged tree, and only then swaps the committed stack root. This closes the partial-rewrite risk on mid-copy failure while preserving overlay content exactly.
- Added focused proof for the two reviewer concerns:
  - unsupported non-Go stack leaves `sources.yaml`, the committed stack tree, and runner/git activity untouched
  - staged replacement failure leaves the committed vanilla tree, overlay, and mutable `sources.yaml` unchanged
  - the existing vendored Go checkout refresh proof still passes under the narrowed scope

### Final Bounded RED
Command:
```bash
GOCACHE=$PWD/.cache/go-build go test ./internal/update -count=1
```
Relevant failing output before the last implementation:
```text
--- FAIL: TestRunFailsClearlyForUnsupportedNonGoStackBeforeWrites (0.04s)
    update_test.go:371: Run() error = nil, want unsupported language failure
FAIL
FAIL	mkproj/internal/update	0.609s
FAIL
```
Why expected:
- The new scope proof showed that `Run` still attempted a Python stack instead of rejecting it before mutable work, so the representative-proof boundary was still implicit rather than enforced in code.

### Final Bounded Verification
Commands:
```bash
GOCACHE=$PWD/.cache/go-build go test ./internal/update -count=1
GOCACHE=$PWD/.cache/go-build go test ./internal/update ./cmd/mkproj -count=1
GOCACHE=$PWD/.cache/go-build go test ./... -count=1
```
Results:
```text
ok   mkproj/internal/update	0.664s
ok   mkproj/internal/update	0.352s
ok   mkproj/cmd/mkproj	9.264s
ok   mkproj/cmd/mkproj	9.015s
ok   mkproj/internal/allowlist	0.513s
ok   mkproj/internal/catalog	1.049s
ok   mkproj/internal/init	4.239s
ok   mkproj/internal/project	0.679s
ok   mkproj/internal/prompt	1.538s
ok   mkproj/internal/remote	0.815s
ok   mkproj/internal/scaffold	1.722s
ok   mkproj/internal/update	2.045s
ok   mkproj/test	13.439s
```
