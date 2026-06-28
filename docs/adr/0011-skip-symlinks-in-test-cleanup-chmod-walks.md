# Test cleanup chmod walks MUST skip symlinks

**Status:** accepted · 2026-06-28

## Context

Integration tests create isolated runtime directories under `.cache/local-release/mkproj-runtime-*`
and register a `t.Cleanup` that walks the directory tree with `filepath.WalkDir`, restores
permissions (0755 dirs, 0644 files), then calls `os.RemoveAll`. The permission restoration
exists because mise and scaffolded projects may restrict file permissions during their runs,
and `os.RemoveAll` cannot delete entries it cannot stat.

During `mise install`, mise records which config files it trusts by creating symlinks in
`<MISE_STATE_DIR>/trusted-configs/` that point to the directory containing each trusted
`.mise.toml`. When the test's `MISE_STATE_DIR` lives inside the runtime directory and the
repo root's `mise.toml` is discovered by mise, the symlink target is the **repo root itself**:

```
.cache/local-release/mkproj-runtime-*/mise-state/trusted-configs/<hash>
  → /path/to/agentic_template_start        ← repo root
```

Go's `os.Chmod` uses the `chmod(2)` syscall, which **always follows symlinks** — there is no
`lchmod` equivalent in the Go standard library. `filepath.WalkDir` visits symlinks as entries
(reporting `d.Type()&fs.ModeSymlink != 0`) but does NOT report them via `d.IsDir()`. The
original cleanup treated every non-directory entry as a file and applied `0o644`. Because
`os.Chmod` followed the symlink, this set the **repo root directory** to mode `0o644` —
stripping its execute bit and making it (and everything below it) inaccessible.

The failure was catastrophic: the agent's shell hooks could not spawn `/bin/sh`, no commands
could execute, and the entire development environment was locked until permissions were
manually restored.

## Decision

All `filepath.WalkDir` callbacks that call `os.Chmod` MUST skip symlink entries:

```go
if d.Type()&fs.ModeSymlink != 0 {
    return nil
}
```

Symlinks do not own the permissions of their targets. `os.RemoveAll` already removes symlinks
without following them, so skipping them in the chmod pass loses nothing and prevents
escaping the intended directory boundary via symlink-following.

This rule applies to any future WalkDir+Chmod pattern anywhere in the test infrastructure or
the generator itself.

## Considered Options

- **Skip symlinks in the walk callback** — a single type-check before `os.Chmod`. Simple,
  targeted, no false positives. Adopted.

- **Path boundary check** (`filepath.Rel` + reject `..` prefix) — validates that each path
  is within the intended root. Does not help here because the *path* reported by `WalkDir`
  IS within the directory — it is the symlink entry, not the target. The escape happens
  inside `os.Chmod`, after the path check would pass. Rejected: addresses the wrong layer.

- **Replace WalkDir+Chmod with a single `os.RemoveAll`** — drop the permission restoration
  entirely and rely on the OS to handle removal. Rejected: `os.RemoveAll` itself fails on
  entries with restricted permissions (e.g. `0o000` dirs created by the scaffolded project's
  test of permission errors). The chmod pass exists to make `RemoveAll` succeed reliably.

- **Use `os.Lchown`/`fchmodat` with `AT_SYMLINK_NOFOLLOW`** — Go's standard library does
  not expose `lchmod` or `fchmodat` with the nofollow flag for permissions. Would require
  `syscall` package use, platform-specific code, and gains nothing over simply skipping the
  entry. Rejected: unnecessary complexity.

## Consequences

- The symlink guard prevents `os.Chmod` from ever modifying a target outside the runtime
  directory, regardless of what symlinks mise (or any other tool) creates during test runs.
- `os.RemoveAll` still removes the symlink entries themselves (it uses `unlinkat`, not
  `chmod`), so cleanup remains complete.
- Any future `WalkDir`+`Chmod` code added to this repo MUST include the symlink check. This
  ADR serves as the precedent — reviewers should flag any `os.Chmod` inside a walk callback
  that does not first check `d.Type()&fs.ModeSymlink`.
- The root cause (Go `os.Chmod` following symlinks) is a language-level behavior that cannot
  be configured away — defensive code is the only mitigation.
