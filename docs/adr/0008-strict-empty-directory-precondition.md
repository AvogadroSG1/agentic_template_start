# mkproj init requires a strictly empty directory

**Status:** accepted · 2026-06-19 (stated in the init-lifecycle spec §1.1; recorded as its own ADR 2026-06-20)

## Context

`mkproj init` is an ordered mutation that creates a whole project tree in the current
directory. What it is allowed to assume about that directory's starting state determines the
entire failure model: whether it must merge into existing content, whether it can safely tell
the user to "delete and retry," and whether it needs a transactional rollback. The precondition
was stated in the init-lifecycle spec and depended upon by ADR-0004 (fail-fast), but never
recorded as a decision in its own right.

## Decision

`mkproj init` **refuses to run unless the target directory is empty**, ignoring only inert
cruft (`.DS_Store`). It errors loudly rather than merging into pre-existing content. A
`--force` / `--in-place` relaxation is **explicitly out of scope for v1**.

## Considered Options

- **Merge into a non-empty directory.** Rejected for v1: composing scaffold output over
  arbitrary pre-existing files makes the output non-deterministic, makes "delete and retry"
  unsafe (it would destroy the user's files), and forces a transactional rollback the design
  deliberately avoids.
- **A `--force` / `--in-place` flag from day one.** Rejected for v1: adds the hardest case
  (collision resolution, partial-overwrite recovery) before the common case is proven. Left
  as a possible post-v1 extension.

## Consequences

- The entire resulting tree is mkproj's own creation, so the blast radius of the ADR-0004
  recovery instruction (recursive-force-delete the directory, then retry) is a known-disposable
  directory — this precondition is *why* fail-fast-with-no-rollback is safe.
- **If this precondition is ever relaxed** (`--force`/`--in-place`), ADR-0004 must be revisited:
  a delete-and-retry recovery would then destroy pre-existing user files.
- The check is cheap and runs first, before any mutation.
