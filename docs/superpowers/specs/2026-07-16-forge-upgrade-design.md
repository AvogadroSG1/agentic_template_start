# forge upgrade — static infrastructure propagation

**Date:** 2026-07-16
**Status:** approved
**Scope:** New CLI command to propagate managed infrastructure files to existing forge-scaffolded repos

## Problem

`forge init` writes infrastructure files (guard hook, secret-scan, hook configs) once at scaffold time. When forge ships fixes to these files (e.g., ADR-0015 relaxing D11 for HTTP reads), already-scaffolded repos retain the old versions indefinitely. There is no mechanism to bring them forward.

## Decision

Add `forge upgrade [--check]` — an unconditional overwrite of all managed static infrastructure files from the embedded template, versioned by a single integer constant.

## Command Surface

```
forge upgrade [--check]
```

- **Without `--check`:** Overwrites all managed infrastructure files from the embedded template. Prints what was updated. Exit 0 on success.
- **With `--check`:** Compares on-disk version to embedded version. If stale, prints advisory and exits 1. If current, exits 0 silently. No mutation.

## Managed File Set

| Embedded source path | Target in repo | Mode |
|---|---|---|
| `templates/common/claude/hooks/guard` | `.claude/hooks/guard` | 0755 |
| `templates/common/claude/hooks/secret-scan.sh` | `.claude/hooks/secret-scan.sh` | 0755 |
| `templates/common/claude/settings.json` | `.claude/settings.json` | 0644 |
| `templates/common/codex/hooks.json` | `.codex/hooks.json` | 0644 |

Additionally, `.forge-infra-version` (repo root, mode 0644) stores the on-disk version integer.

## Package Structure

**New package:** `internal/upgrade/`

```go
// upgrade.go
package upgrade

const Version = 1

type Status struct {
    Version int      // embedded version constant
    OnDisk  int      // version read from .forge-infra-version (0 if missing)
    Stale   bool     // OnDisk < Version
    Updated []string // paths written (empty in --check mode)
}

func Run(assets fs.FS, targetDir string, checkOnly bool) (Status, error)
```

**CLI integration:** `cmd/forge/main.go` adds `"upgrade"` to the command switch. `runUpgrade(args, assets)` parses `--check`, resolves cwd, calls `upgrade.Run`.

## Version Tracking

A single `Version` constant in `internal/upgrade/upgrade.go`. Bumped in any commit that changes a managed file in `templates/common/`.

On-disk state lives in `.forge-infra-version` — a one-line file containing just the integer. Written last after all other files so it acts as a commit marker: interrupted upgrades stay stale and re-apply on next run.

When `.forge-infra-version` is missing (all existing repos today), `OnDisk` defaults to 0, triggering the upgrade path on first encounter.

## Detection Heuristic

To confirm we're in a forge-scaffolded repo, check for at least ONE of:
- `.claude/hooks/guard` exists
- `.forge-infra-version` exists
- `.claude/settings.json` exists

If none exist: `"not a forge-managed repository (run from a project created with forge init)"`

## SessionStart Hook Integration

The managed `settings.json` and `codex/hooks.json` gain an advisory SessionStart hook:

```json
{
  "type": "command",
  "command": "command -v forge >/dev/null 2>&1 && forge upgrade --check || true"
}
```

This is self-bootstrapping: existing repos get the hook when they first run `forge upgrade` (since `settings.json` is a managed file).

## Output Format

```
# Normal mode, files updated:
forge upgrade: updated 3 files to infrastructure v4
  .claude/hooks/guard
  .claude/hooks/secret-scan.sh
  .claude/settings.json

# Normal mode, already current:
forge upgrade: infrastructure is current (v4)

# --check, stale:
forge upgrade: infrastructure is at v2, current is v4; run `forge upgrade`

# --check, current:
(silent, exit 0)
```

## Behavioral Invariants

1. **Idempotency:** If on-disk version equals embedded version, no files are written and the command reports "current". Running upgrade twice with no forge binary change in between produces no git diff.
2. **Partial presence:** Missing target files/directories are created. Pre-existence is not required.
3. **Write order:** Version file is written last — acts as a transaction commit marker.
4. **No templating:** All managed files are static byte-copies. No template variables needed.
5. **Byte-identical guarantee:** Per ADR-0015, source and generated guard copies MUST remain byte-identical. `forge upgrade` enforces this by construction.
6. **Output lists only managed infrastructure files** — the `.forge-infra-version` bookkeeping file is written silently and not listed in user-facing output.

## Behavioral Scenarios

```gherkin
Scenario: First upgrade on a legacy repo
  Given a forge-scaffolded repo with no .forge-infra-version file
  When the user runs `forge upgrade`
  Then all managed files are overwritten from the embedded template
  And .forge-infra-version is created with the current version

Scenario: Upgrade is idempotent
  Given a repo at infrastructure version N
  When the user runs `forge upgrade` with embedded version N
  Then no files are modified
  And output reports "infrastructure is current (vN)"

Scenario: Check mode reports staleness
  Given a repo at infrastructure version 1
  When the user runs `forge upgrade --check` with embedded version 3
  Then output reports the gap
  And exit code is 1
  And no files are modified

Scenario: Interrupted upgrade re-applies
  Given a repo where upgrade wrote guard and secret-scan but crashed before writing .forge-infra-version
  When the user runs `forge upgrade` again
  Then all managed files are written (including the ones already current)
  And .forge-infra-version is written last

Scenario: Non-forge directory is rejected
  Given a directory with no .claude/hooks/guard, no .forge-infra-version, and no .claude/settings.json
  When the user runs `forge upgrade`
  Then the command exits with an error naming the detection heuristic
```

## Version Bumping Discipline

The `Version` constant MUST be incremented in any commit that modifies a file under `templates/common/claude/hooks/`, `templates/common/claude/settings.json`, or `templates/common/codex/hooks.json`. This is a manual invariant enforced by reviewer discipline, testable by a gate test that compares embedded file hashes against a pinned set for the declared version.
