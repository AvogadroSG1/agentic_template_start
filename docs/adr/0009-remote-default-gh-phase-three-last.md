# Remote creation defaults to gh, runs last, and is never auto-deleted

**Status:** accepted · 2026-06-17 (decided in system-design §3; consolidated as its own ADR 2026-06-20)

## Context

Publishing a remote is the one init step with an external, outward-facing side effect. Three
sub-decisions about it were scattered across system-design §3, ADR-0004's context, and the
init-lifecycle spec §1.3: what the default remote mode is, *when* in the lifecycle remote
creation runs, and what happens when remote creation or the first push fails. They belong in
one record because they are interlocking and because the failure rule (never auto-delete) is a
deliberate, surprising choice a future reader will question.

## Decision

1. **Default mode is `--remote gh`** for an interactive run: `gh repo create <name> --source .
   --remote origin` followed by `git push -u origin <branch>`. `--remote url` (explicit URL)
   and `--remote none` (local-only; the smoke tier's primary consumer) remain available.
2. **Phase 3 (remote) runs last**, after every local step succeeds and the lefthook gates are
   wired — so the initial commit + push passes the full pre-commit / pre-push pipeline
   (secret-scan, lint, format, tests) before anything reaches the remote.
3. **mkproj never auto-deletes a remote.** If `gh repo create` succeeds but the first push
   fails (gate trips or network), mkproj **leaves the remote**, prints its URL and the
   `git push -u origin <branch>` retry, and mentions `gh repo delete` as the *user's* option.
   If `gh repo create` itself fails, mkproj reports the reason and leaves a complete
   local-only repo.

## Considered Options

- **Default to local-only (`--remote none`).** Rejected: publishing is part of the "day one it
  just works" path; making it the default removes a manual afterthought for the common case.
- **Create the remote early, then build locally.** Rejected: a local-step failure would orphan
  a remote, and the first push would not have passed the gate pipeline — defeating the
  "complete repo before it reaches the remote" guarantee.
- **Auto-delete the remote on a failed first push.** Rejected: deletion is an irreversible
  outward action, and the most common push failure is a gate working as designed. Phase-3-last
  ordering already guarantees the failure is recoverable by a manual retry; deleting the user's
  freshly-created repo would be more surprising and more destructive than leaving it.

## Consequences

- The product's never-delete-remote rule is a hard invariant. **Note for the smoke harness:**
  the full-tier end-to-end test creates and *must* delete its own throwaway remote — that
  teardown is a separate code path in the test harness, never wired into product code (see
  the SPEC verification section and `uuw`).
- Because the one external side effect runs last, ADR-0004's fail-fast-no-rollback never has to
  unwind a remote.
