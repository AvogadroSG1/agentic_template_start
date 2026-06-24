# Task 2 Report: Extend the install lifecycle and ignore local build output

## Scope

- Modified `Makefile`
- Modified `test/makefile_test.go`
- Modified `.gitignore`
- Left `.superpowers/sdd/progress.md` untouched because it is controller-owned

## TDD loop

### Red

- Added `TestMakefileDefinesInstallLifecycleTargets`
- Added `TestMakeInstallAndUninstallRoundTrip`
- Added `TestGitIgnoreIgnoresBinDirectory`
- Ran:

```bash
GOCACHE=$PWD/.cache/go-build go test ./test -run 'TestMakefileDefinesInstallLifecycleTargets|TestMakeInstallAndUninstallRoundTrip|TestGitIgnoreIgnoresBinDirectory' -count=1
```

- Observed the expected failures:
  - `Makefile` was missing `BINDIR ?= $(HOME)/.local/bin`
  - `Makefile` had no `install` target
  - `Makefile` had no `uninstall` target
  - `.gitignore` was missing `bin/`

### Green

- Extended `Makefile` with:
  - `BINDIR ?= $(HOME)/.local/bin`
  - `install` target that depends on `build`
  - `uninstall` target
  - expanded `.PHONY` list
- Updated the help target awk pattern so targets with prerequisites such as `install: build ## ...` appear in `make` output
- Kept `build` self-contained by exporting repo-local cache settings for `GOCACHE`, `TOKF_HOME`, and `TOKF_DB_PATH`
- Added `bin/` to `.gitignore`

### Refine / regression fixes

- The first full-suite run exposed two follow-up issues in the shared test file:
  - the older core target expectation still expected `.PHONY: help build test clean`
  - parallel makefile tests could race on shared `bin/` state in the same repo root
- Fixed both by:
  - updating the core target expectation to include `install` and `uninstall`
  - updating the default-help expectation to include `install` and `uninstall`
  - serializing the makefile tests with a package-level mutex while preserving `t.Parallel()`

## Verification

### Focused tests

```bash
GOCACHE=$PWD/.cache/go-build go test ./test -run 'TestMakefileDefinesInstallLifecycleTargets|TestMakeInstallAndUninstallRoundTrip|TestGitIgnoreIgnoresBinDirectory' -count=1
```

- Result: PASS

### Full suite

```bash
GOCACHE=$PWD/.cache/go-build go test ./... -count=1
```

- Result: PASS

### Makefile lifecycle checks

Ran:

```bash
make
make install BINDIR=$PWD/.cache/mkproj-bin
PATH=$PWD/.cache/mkproj-bin:$PATH command -v mkproj
./.cache/mkproj-bin/mkproj update
make uninstall BINDIR=$PWD/.cache/mkproj-bin
make clean
```

Observed:

- `make` listed `help`, `build`, `test`, `install`, `uninstall`, and `clean`
- `make install` created `.cache/mkproj-bin/mkproj`
- `command -v mkproj` resolved to the repo-local install path
- `./.cache/mkproj-bin/mkproj update` failed with `missing required flag: --stack`, proving the installed binary ran
- `make uninstall` removed `.cache/mkproj-bin/mkproj`
- `make clean` removed `bin/`

## Notes

- During verification I briefly ran uninstall and clean in parallel with path and binary checks, which caused a verification race. I corrected that by reinstalling and rerunning the shell lifecycle sequence in the proper order.
- The environment emits non-fatal `tokf` cache warnings for restricted global cache paths. The Task 2 change keeps `go build` working by using repo-local cache paths inside the `build` target.

## Outcome

- Task 2 requirements are implemented
- The install lifecycle works end to end
- Local build output is ignored
- The task-owned test surface is updated and green
