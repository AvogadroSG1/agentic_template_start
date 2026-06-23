# mkproj V1 Remaining Execution Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Finish the remaining v1 `mkproj` implementation slices using independently testable feature branches that each land through PRs before the next slice begins.

**Architecture:** Use one long-lived execution worktree as the orchestration spine and create one short-lived feature branch per independently landable slice. Each slice starts with BDD scenario work, gets an implementation-phase skills check plus a review-phase skills check, merges to `main`, and is then pulled back into the execution branch before work continues.

**Tech Stack:** Go 1.24, embedded templates via `embed.FS`, Bash hooks, `bats`, GitHub PR workflow, Beads (`bd`), existing `cmd/mkproj`, `internal/*`, `templates/*`, and cross-repo guideline files in `~/peter_code/ai_support`.

## Global Constraints

- Every task MUST use `docs/SPEC.md`, `CONTEXT.md`, and the Beads issue text as the source of truth.
- Every implementation slice MUST be independently testable and land through its own PR before the next slice starts.
- Every implementation slice MUST write or extend BDD scenarios before or alongside production changes, never after the fact.
- Every slice MUST run two skill checks: one during implementation and one during review.
- `7eu` MUST remain out of scope for this run because it is post-v1 backlog work.
- If a slice needs cross-repo changes or external credentials, the blocker MUST be reported explicitly instead of being silently bypassed.

---

### Task 1: Establish the execution control plane

**Files:**
- Create: `.superpowers/sdd/progress.md`
- Create: `docs/initial_implementation/reviews/2026-06-23-mkproj-execution-report.md`

**Interfaces:**
- Consumes: `docs/SPEC.md`, `CONTEXT.md`, `bd show <issue>`
- Produces: a durable progress ledger and a running report file that later tasks append to

- [ ] **Step 1: Create the progress ledger**

Write `.superpowers/sdd/progress.md` with one section per slice:
- secret-scan-and-hooks
- gate-pipeline
- walking-skeleton-smoke
- allowlist-reconciler
- maintainer-refresh
- guideline-publication-conformance

- [ ] **Step 2: Create the running report shell**

Write `docs/initial_implementation/reviews/2026-06-23-mkproj-execution-report.md` with four headings:
- `Completed items`
- `Assumptions made`
- `Errors / blockers encountered`
- `Integration notes`

- [ ] **Step 3: Verify the clean baseline**

Run: `GOCACHE=$PWD/.cache/go-build go test ./... -count=1`  
Expected: PASS

- [ ] **Step 4: Record the branch-sync loop**

Use this exact branch loop for every later task:

```bash
git switch codex-mkproj-v1-execution-2026-06-23
git pull --ff-only origin main
git switch -c <slice-branch>
```

- [ ] **Step 5: Commit**

```bash
git add .superpowers/sdd/progress.md docs/initial_implementation/reviews/2026-06-23-mkproj-execution-report.md
git commit -m "docs: add mkproj execution control plane" -m "Co-Authored-By: Peter O'Connor <poconnor@stackoverflow.com>\nCo-Authored-By: Codex <noreply@anthropic.com> - GPT-5"
```

### Task 2: Finish the secret-scan and hook-chain slice

**Files:**
- Modify: `.claude/hooks/secret-scan.sh`
- Modify: `templates/common/claude/hooks/secret-scan.sh`
- Modify: `templates/common/claude/hooks/guard`
- Modify: `templates/common/claude/settings.json`
- Modify: `templates/common/codex/hooks.json`
- Modify: `test/secret-scan.bats`
- Modify: any focused Go or shell verification files needed for `485`

**Interfaces:**
- Consumes: `02o`, `485`, `79r`, `7sn`; SPEC §11-§13; existing `scan-command` behavior from `aa9`
- Produces: shared scanner parity across source/template, verified chain-mode hooks, deny-only guard parity for Claude and Codex

- [ ] **Step 1: Claim the slice issues**

Run:

```bash
bd update agentic_template_start-02o --claim
bd update agentic_template_start-485 --claim
bd update agentic_template_start-79r --claim
bd update agentic_template_start-7sn --claim
```

- [ ] **Step 2: Create the slice branch**

Run:

```bash
git switch codex-mkproj-v1-execution-2026-06-23
git pull --ff-only origin main
git switch -c codex-mkproj-secret-scan-hooks
```

- [ ] **Step 3: Write or extend the failing BDD scenarios first**

Update or add focused scenarios for:
- `scan-staged` staged-secret blocking
- template/source scanner parity
- chain-mode hook survival after `lefthook install`
- guard deny-only behavior for D1-D16 with D9/D10 delegated to `scan-command`

Run:

```bash
GOCACHE=$PWD/.cache/go-build go test ./... -count=1
```

Expected: at least one targeted scenario fails for the intended missing behavior.

- [ ] **Step 4: Dispatch the implementation subagent**

Implement only this slice, then run an implementation-phase skills check against:
- `superpowers:test-driven-development`
- `superpowers:systematic-debugging` if verification fails

- [ ] **Step 5: Run focused verification**

Run:

```bash
GOCACHE=$PWD/.cache/go-build go test ./... -count=1
bats test/secret-scan.bats
```

Expected: PASS, or explicit blocker if `bats` is unavailable in the environment.

- [ ] **Step 6: Dispatch the review subagent**

Run the review-phase skills check against:
- `superpowers:receiving-code-review`
- `superpowers:requesting-code-review`

- [ ] **Step 7: Land and pull back**

Run:

```bash
git push -u origin codex-mkproj-secret-scan-hooks
gh pr create --base main --head codex-mkproj-secret-scan-hooks --fill
gh pr merge --auto --squash
git switch codex-mkproj-v1-execution-2026-06-23
git pull --ff-only origin main
```

- [ ] **Step 8: Update progress**

Append the landed commit and verification result to `.superpowers/sdd/progress.md` and the running report.

### Task 3: Finish the gate-pipeline seam

**Files:**
- Modify: `templates/golden/*/.mkproj-overlay/mise.toml`
- Modify: `templates/golden/*/.mkproj-overlay/lefthook.yml`
- Modify: `templates/golden/*/.mkproj-overlay/.github/workflows/ci.yml`
- Modify: `internal/scaffold/gate_assets_test.go`
- Modify: any focused writer or CLI tests needed for `x2k` / `ud1`

**Interfaces:**
- Consumes: `x2k`, `ud1`, `wqh.2`, plus the landed hook-chain slice
- Produces: a real cross-stack `mise run ci` contract shared by local hooks and CI

- [ ] **Step 1: Claim the slice issues**

Run:

```bash
bd update agentic_template_start-x2k --claim
bd update agentic_template_start-ud1 --claim
bd update agentic_template_start-wqh.2 --claim
```

- [ ] **Step 2: Create the slice branch**

Run:

```bash
git switch codex-mkproj-v1-execution-2026-06-23
git pull --ff-only origin main
git switch -c codex-mkproj-gate-pipeline
```

- [ ] **Step 3: Write or extend the failing BDD scenarios first**

Cover:
- each shipped v1 stack has `fmt`, `lint`, `test`, and `ci`
- `lefthook.yml` extends the secret-only baseline instead of recreating it
- CI delegates only to `mise install` then `mise run ci`
- the `wqh.2` seam test resolves a real `ci` task for every shipped stack

- [ ] **Step 4: Dispatch the implementation subagent and implementation-phase skills check**

The implementation check MUST confirm that no stack-specific inline CI logic bypasses the shared `mise` contract.

- [ ] **Step 5: Run focused verification**

Run:

```bash
GOCACHE=$PWD/.cache/go-build go test ./internal/scaffold ./cmd/mkproj ./test -count=1
GOCACHE=$PWD/.cache/go-build go test ./... -count=1
```

Expected: PASS

- [ ] **Step 6: Dispatch the review subagent**

The review check MUST validate both the seam contract (`wqh.2`) and the issue boundaries (`x2k` vs. `ud1`).

- [ ] **Step 7: Land and pull back**

Run:

```bash
git push -u origin codex-mkproj-gate-pipeline
gh pr create --base main --head codex-mkproj-gate-pipeline --fill
gh pr merge --auto --squash
git switch codex-mkproj-v1-execution-2026-06-23
git pull --ff-only origin main
```

- [ ] **Step 8: Update progress**

Append the landed commit and verification result to the ledger and report.

### Task 4: Finish the walking-skeleton smoke seam

**Files:**
- Modify: `cmd/mkproj/walking_skeleton_test.go`
- Create or modify: smoke-specific tests under `cmd/mkproj/` or `test/`
- Modify: any helper fixtures used to assert offline/no-network behavior

**Interfaces:**
- Consumes: `uuw`, `wqh.1`, plus the landed gate-pipeline slice
- Produces: local hermetic smoke coverage for the composition seam and a documented remote-smoke boundary

- [ ] **Step 1: Claim the slice issues**

Run:

```bash
bd update agentic_template_start-uuw --claim
bd update agentic_template_start-wqh.1 --claim
```

- [ ] **Step 2: Create the slice branch**

Run:

```bash
git switch codex-mkproj-v1-execution-2026-06-23
git pull --ff-only origin main
git switch -c codex-mkproj-walking-skeleton-smoke
```

- [ ] **Step 3: Write or extend the failing BDD scenarios first**

Cover:
- `mkproj init --stack <key> --remote none`
- `mise install`
- `mise run ci`
- a clean initial commit through hooks
- explicit no-network assertions for the local hermetic tier

- [ ] **Step 4: Dispatch the implementation subagent and implementation-phase skills check**

The implementation check MUST confirm the smoke tests exercise the seam without re-testing chain mode already owned by `485`.

- [ ] **Step 5: Run focused verification**

Run:

```bash
GOCACHE=$PWD/.cache/go-build go test ./cmd/mkproj ./test -count=1
GOCACHE=$PWD/.cache/go-build go test ./... -count=1
```

Expected: PASS, or a concrete blocker if the environment cannot support the hermetic assertions.

- [ ] **Step 6: Dispatch the review subagent**

The review check MUST verify that `wqh.1` closes the composition seam and does not leak remote-smoke behavior into product code.

- [ ] **Step 7: Land and pull back**

Run:

```bash
git push -u origin codex-mkproj-walking-skeleton-smoke
gh pr create --base main --head codex-mkproj-walking-skeleton-smoke --fill
gh pr merge --auto --squash
git switch codex-mkproj-v1-execution-2026-06-23
git pull --ff-only origin main
```

- [ ] **Step 8: Update progress**

Append the landed commit and verification result to the ledger and report.

### Task 5: Finish the allowlist reconciler slice

**Files:**
- Modify: `cmd/mkproj/main.go`
- Modify: `cmd/mkproj/main_test.go`
- Modify: `internal/allowlist/sync.go`
- Modify: `internal/allowlist/*_test.go`
- Modify: `templates/common/claude/settings.local.json.tmpl`
- Modify: `templates/common/claude/settings.json`
- Modify: `templates/common/codex/hooks.json`

**Interfaces:**
- Consumes: `rom`; SPEC §11 and §13
- Produces: managed-block sync, stale detection, notify-only SessionStart hooks, opt-in personal rules

- [ ] **Step 1: Claim the slice issue**

Run:

```bash
bd update agentic_template_start-rom --claim
```

- [ ] **Step 2: Create the slice branch**

Run:

```bash
git switch codex-mkproj-v1-execution-2026-06-23
git pull --ff-only origin main
git switch -c codex-mkproj-allowlist-reconciler
```

- [ ] **Step 3: Write or extend the failing BDD scenarios first**

Cover:
- managed block rewrite preserves manual content
- stale check notifies without mutating
- `--include-personal` is opt-in
- conflicting or missing language markers fail loudly

- [ ] **Step 4: Dispatch the implementation subagent and implementation-phase skills check**

The implementation check MUST confirm the guard remains deny-only and that no SessionStart hook mutates project files automatically.

- [ ] **Step 5: Run focused verification**

Run:

```bash
GOCACHE=$PWD/.cache/go-build go test ./cmd/mkproj ./internal/allowlist -count=1
GOCACHE=$PWD/.cache/go-build go test ./... -count=1
```

Expected: PASS

- [ ] **Step 6: Dispatch the review subagent**

The review check MUST validate managed-block scope, stale-notify semantics, and personal-rule opt-in behavior.

- [ ] **Step 7: Land and pull back**

Run:

```bash
git push -u origin codex-mkproj-allowlist-reconciler
gh pr create --base main --head codex-mkproj-allowlist-reconciler --fill
gh pr merge --auto --squash
git switch codex-mkproj-v1-execution-2026-06-23
git pull --ff-only origin main
```

- [ ] **Step 8: Update progress**

Append the landed commit and verification result to the ledger and report.

### Task 6: Finish the maintainer refresh seam

**Files:**
- Modify: `internal/update/update.go`
- Modify: `internal/update/update_test.go`
- Modify: `cmd/mkproj/main.go`
- Modify: `cmd/mkproj/main_test.go`
- Modify: `sources.yaml`
- Modify: any new normalization or snapshot fixtures required by the refresh path

**Interfaces:**
- Consumes: `cjl`, `wqh.3`, plus the already-landed `369`
- Produces: deterministic `mkproj update` behavior with orphan/collision seam handling

- [ ] **Step 1: Claim the slice issues**

Run:

```bash
bd update agentic_template_start-cjl --claim
bd update agentic_template_start-wqh.3 --claim
```

- [ ] **Step 2: Create the slice branch**

Run:

```bash
git switch codex-mkproj-v1-execution-2026-06-23
git pull --ff-only origin main
git switch -c codex-mkproj-maintainer-refresh
```

- [ ] **Step 3: Write or extend the failing BDD scenarios first**

Cover:
- update refreshes vendored assets and snapshots
- run-twice idempotence
- orphaned overlay path fails loud and writes nothing
- collision warns but preserves overlay-wins composition

- [ ] **Step 4: Dispatch the implementation subagent and implementation-phase skills check**

The implementation check MUST confirm normalization touches only the vanilla layer and does not auto-mutate overlays.

- [ ] **Step 5: Run focused verification**

Run:

```bash
GOCACHE=$PWD/.cache/go-build go test ./internal/update ./cmd/mkproj -count=1
GOCACHE=$PWD/.cache/go-build go test ./... -count=1
```

Expected: PASS

- [ ] **Step 6: Dispatch the review subagent**

The review check MUST validate both the `cjl` story behavior and the `wqh.3` seam gate.

- [ ] **Step 7: Land and pull back**

Run:

```bash
git push -u origin codex-mkproj-maintainer-refresh
gh pr create --base main --head codex-mkproj-maintainer-refresh --fill
gh pr merge --auto --squash
git switch codex-mkproj-v1-execution-2026-06-23
git pull --ff-only origin main
```

- [ ] **Step 8: Update progress**

Append the landed commit and verification result to the ledger and report.

### Task 7: Finish guideline publication and conformance

**Files:**
- External modify: `~/peter_code/ai_support/guidelines/golang.md`
- External modify: `~/peter_code/ai_support/guidelines/python.md`
- External modify: `~/peter_code/ai_support/guidelines/csharp.md`
- Modify: conformance test files in this repo for `ebp`
- Modify: any helper code that resolves canonical guideline paths

**Interfaces:**
- Consumes: `zz8`, `ebp`; SPEC §10.2 and §18.1
- Produces: published canonical guideline files plus executable floor-only conformance tests

- [ ] **Step 1: Claim the slice issues**

Run:

```bash
bd update agentic_template_start-zz8 --claim
bd update agentic_template_start-ebp --claim
```

- [ ] **Step 2: Create the slice branch**

Run:

```bash
git switch codex-mkproj-v1-execution-2026-06-23
git pull --ff-only origin main
git switch -c codex-mkproj-guideline-conformance
```

- [ ] **Step 3: Write or extend the failing BDD scenarios first**

Cover:
- canonical guideline files are reachable at their stable path
- missing guideline MUST/SHOULD coverage fails
- vetted overlay extras do not fail the test
- shipped template without a guideline-backed language fails

- [ ] **Step 4: Dispatch the implementation subagent and implementation-phase skills check**

The implementation check MUST stop and report a blocker if the `ai_support` repo cannot be safely isolated for `zz8`.

- [ ] **Step 5: Run focused verification**

Run:

```bash
GOCACHE=$PWD/.cache/go-build go test ./... -count=1
```

Expected: PASS in this repo, plus explicit cross-repo publication evidence for `zz8`.

- [ ] **Step 6: Dispatch the review subagent**

The review check MUST validate that `ebp` enforces floor-not-ceiling semantics and that `zz8` did not sweep unrelated `ai_support` changes into the publish branch.

- [ ] **Step 7: Land and pull back**

Run:

```bash
git push -u origin codex-mkproj-guideline-conformance
gh pr create --base main --head codex-mkproj-guideline-conformance --fill
gh pr merge --auto --squash
git switch codex-mkproj-v1-execution-2026-06-23
git pull --ff-only origin main
```

- [ ] **Step 8: Update progress**

Append the landed commit and verification result to the ledger and report.

### Task 8: Final verification and completion report

**Files:**
- Modify: `.superpowers/sdd/progress.md`
- Modify: `docs/initial_implementation/reviews/2026-06-23-mkproj-execution-report.md`

**Interfaces:**
- Consumes: all prior landed slices
- Produces: final report to Peter and a branch ready for finish/cleanup

- [ ] **Step 1: Run full verification**

Run:

```bash
GOCACHE=$PWD/.cache/go-build go test ./... -count=1
```

Expected: PASS

- [ ] **Step 2: Confirm the post-v1 exclusion**

Verify `agentic_template_start-7eu` is still open and unmodified.

- [ ] **Step 3: Final review**

Dispatch a whole-branch reviewer using `superpowers:requesting-code-review`.

- [ ] **Step 4: Finish the report**

Populate:
- completed items
- assumptions made
- errors / blockers encountered
- integration notes

- [ ] **Step 5: Push the execution branch**

Run:

```bash
git push
```
