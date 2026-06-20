# mkproj init is fail-fast with no transactional rollback

**Status:** accepted · 2026-06-19

## Context

`mkproj init` is a multi-step mutation: Phase 1 (render/copy/link) → guard install →
`bd init` → `instill init`/`pick-skills`/`check-skills` → `lefthook install` → Phase 3
remote. Any step can fail (a delegated tool errors, a network hiccup, bad input). The
question is what state the target directory is left in, and whether mkproj should undo its
own work.

Two facts shape the answer:

1. **`mkproj init` requires an empty directory** (ADR-0008; the strict precondition). The
   entire resulting tree is therefore mkproj's own creation.
2. **Phase 3 (remote creation) runs last**, after every local step succeeds, so the only step
   with an external side effect cannot fire before a local-step failure (ADR-0009).

## Decision

On any step failure, `mkproj init` **stops at that step, leaves the partial state in place,
and prints the failed step plus a single recovery command** (recursive-force-delete the
directory, then retry). There is **no transactional rollback** — mkproj does not track and
unwind its mutations.

## Considered Options

- **Transactional rollback** — track every file/dir/external action and undo all of them on
  failure, restoring the empty directory. Rejected: substantial complexity (a mutation
  journal, ordered teardown) for a blast radius that the empty-dir precondition already bounds
  to a disposable directory. Worse, some steps have side effects that are ugly or impossible
  to cleanly undo (`bd init` git state, a created GitHub repo) — partial rollback would be
  *more* surprising than none.
- **Best-effort, silent partial** — stop and leave the mess with no guidance. Rejected: a
  confusing half-built directory with no instruction violates the "fail loud, name the thing"
  posture used everywhere else in mkproj.

## Consequences

- Recovery is trivially "delete the directory and retry," and it is **safe** precisely because
  the directory was empty before init (ADR depends on the strict empty-dir precondition — if
  that precondition is ever relaxed via `--force`/`--in-place`, this decision must be revisited,
  because the directory would then contain pre-existing user files that a delete would destroy).
- Because Phase 3 runs last, a local-step failure never orphans a remote; the one external
  side effect is reached only after everything local has succeeded.
- The implementation stays simple: ordered steps, fail at the first error, emit a
  step-specific message and the recovery command. No mutation journal.
- Phase 3's *own* failure has a separate rule (leave the remote and instruct, never
  auto-delete) — see the init-lifecycle spec §1.3.
