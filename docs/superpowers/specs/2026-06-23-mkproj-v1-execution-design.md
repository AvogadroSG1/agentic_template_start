# mkproj V1 Remaining Value-Stream Execution Design

**Date:** 2026-06-23  
**Branch:** `codex-mkproj-v1-execution-2026-06-23`  
**Primary sources:** `docs/SPEC.md`, `CONTEXT.md`, `bd`

## Goal

Finish the remaining v1 `mkproj` implementation work as a sequence of independently landable feature slices, where each slice is BDD-first, reviewed twice, merged to `main`, and then pulled back into the execution branch before the next slice starts.

## Scope

This run covers the remaining v1 implementation units still open or in progress in Beads:

1. Secret-scan and hook chain slices: `02o`, `485`, `79r`, `7sn`
2. Gate-pipeline seam: `x2k`, `ud1`, `wqh.2`
3. Walking-skeleton smoke seam: `uuw`, `wqh.1`
4. Allowlist reconciler and stale notify: `rom`
5. Maintainer refresh seam: `cjl`, `wqh.3`
6. Guideline publication and conformance: `zz8`, `ebp`

`7eu` is explicitly out of scope for this run because Beads and SPEC both mark it as post-v1 expansion work rather than a v1 shipping dependency.

## Execution Model

The execution branch is an integration spine, not the landing branch. Every independently testable feature slice follows the same loop:

1. Claim the Beads issue(s) for the slice.
2. Create a short-lived feature branch from the execution branch.
3. Write or extend the failing BDD scenarios before implementation.
4. Dispatch an implementation subagent for the slice.
5. Run an implementation-phase skills check to confirm the approach is still aligned with SPEC and the issue contract.
6. Run slice verification locally.
7. Dispatch a review subagent for the slice.
8. Fix any Critical or Important findings.
9. Open a PR, merge it into `main`, and fast-forward the execution branch from `origin/main`.

This keeps every value-stream piece bounded, leaves a clean seam for downstream work, and prevents a long-lived branch from hiding integration mistakes.

## Value Streams

### 1. Secret-Scan and Hook Chain

**Beads:** `02o`, `485`, `79r`, `7sn`  
**SPEC / CONTEXT anchors:** SPEC §11, §12, §13, §18.1; CONTEXT terms "guard hook", "secret-exposure guard", "deny floor"

This stream completes the shared secret-scan script, proves beads-plus-lefthook chain mode, and wires the deny-only guard for both Claude and Codex. The integration seam is that the same shared scanner powers both staged-file blocking and command-line blocking without letting hook-side allow logic creep in.

### 2. Gate-Pipeline Seam

**Beads:** `x2k`, `ud1`, `wqh.2`  
**SPEC / CONTEXT anchors:** SPEC §10, §18.1; CONTEXT term "gate"

This stream owns the one-definition-many-callers contract: the shipped overlays must define `mise` tasks, `lefthook` must invoke them locally, and CI must delegate only to `mise run ci`. The seam closes only when every v1 stack resolves the same gate pipeline.

### 3. Walking-Skeleton Smoke

**Beads:** `uuw`, `wqh.1`  
**SPEC / CONTEXT anchors:** SPEC §1, §9, §16, §18.1; CONTEXT term "walking skeleton"

This stream proves the product-level promise that init in an empty directory yields a working repo with at least one real passing test and a functioning local gate path. The local hermetic smoke is the primary slice; the full remote golden path is opt-in and must not leak repos.

### 4. Allowlist Reconciler

**Beads:** `rom`  
**SPEC / CONTEXT anchors:** SPEC §11, §13; CONTEXT terms "allowlist", "managed block", "reconciler"

This stream completes the managed-block sync story and the notify-only SessionStart contract. It must preserve manual configuration outside the managed block and keep personal rules opt-in.

### 5. Maintainer Refresh Seam

**Beads:** `cjl`, `wqh.3`  
**SPEC / CONTEXT anchors:** SPEC §15, §18.1; CONTEXT terms "recipe", "vanilla layer", "refresh seam"

This stream finishes the maintainer-only `mkproj update` path on top of the already-landed steps interpreter. The seam is idempotence plus explicit orphan/collision handling, not merely "command runs once".

### 6. Guideline Publication and Conformance

**Beads:** `zz8`, `ebp`  
**SPEC / CONTEXT anchors:** SPEC §10.2, §18.1; CONTEXT term "guideline file"

This stream crosses repo boundaries. First publish the already-authored guideline updates safely from `ai_support`, then make conformance executable in this repo. `ebp` remains blocked until canonical guideline files are reachable at their stable path.

## Verification Model

Every slice must satisfy four checks before merge:

1. BDD scenarios exist first or are extended first.
2. Focused tests for the slice pass.
3. Full repo verification remains green unless the slice is blocked by external state.
4. A review subagent confirms both code quality and issue/spec compliance.

The final completion report to Peter must list completed items, assumptions, blockers, and downstream integration notes without requiring him to reconstruct the session from Git history.

## Assumptions

1. "Make sure each independent feature MUST be pr, auto merged into main, and then pulled back into this branch" means the execution branch stays alive across slices, while each slice lands through a temporary PR branch.
2. `7eu` should not be implemented in this run because both its issue text and SPEC mark it as deferred until after v1 delivery.
3. Cross-repo work in `ai_support` is allowed when required by `zz8`, but if that repo cannot be safely isolated the slice must stop and report a blocker rather than force a dirty publish.
