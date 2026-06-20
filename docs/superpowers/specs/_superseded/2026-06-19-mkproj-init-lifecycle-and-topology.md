> **⚠️ SUPERSEDED 2026-06-20 by [`/docs/SPEC.md`](../../../SPEC.md).** Retained for history only — do not implement from this document. See SPEC.md §0.1 for the section that subsumes this content.

# mkproj Init Lifecycle, Asset Topology & Verification Contract

**Status:** decided 2026-06-19 (grill-with-docs) · **Author:** Peter O'Connor with Claude Code (databricks-claude-opus-4-8)

Companion to [the CLI/prompt/render contract](2026-06-19-mkproj-cli-prompt-render-contract.md).
Where that document pinned *what mkproj asks and how it renders*, this one pins *how a run
behaves over its lifecycle* (preconditions, failure, remote), *how the scaffolded assets are
laid out across two agents*, and *how the whole system is proven to work*.

Audience: an implementer who builds with guidance. Every decision ends in a Given/When/Then
scenario that defines "done."

Touches issues: `iha`, `s5x`, `wer`, `yuw`, `ud1`, `uuw`, `rom`, `3tt`.

---

## 1. Init preconditions & failure semantics (`s5x`, `wer`, `iha`)

### 1.1 Strict empty-directory precondition

`mkproj init` refuses to run unless the target directory is empty, ignoring only inert
cruft (`.DS_Store`). It errors loudly rather than merging into existing content. A
`--force`/`--in-place` relaxation is explicitly out of scope for v1.

### 1.2 Failure posture — fail-fast with recovery instruction (no rollback)

Init is an ordered mutation: Phase 1 (render/copy/link) → guard install → `bd init` →
`instill init`/`pick-skills`/`check-skills` → `lefthook install` → Phase 3 remote. On any
step failure, mkproj **stops at that step, leaves the partial state, and prints the failed
step plus the single recovery command** (recursive-force-delete of the dir, then retry).

There is **no transactional rollback** — and none is needed, because §1.1 guarantees the
whole directory is mkproj's own creation, so the blast radius of "delete and retry" is a
known-empty dir. The one external side effect (Phase 3 remote) runs last, so a local-step
failure never orphans a remote.

```gherkin
Scenario: Init refuses a non-empty directory
  Given the current directory contains a file
  When the user runs `mkproj init`
  Then it exits non-zero with "directory not empty"
    And nothing is created or modified

Scenario: Local-step failure leaves a recoverable partial and no remote
  Given an empty directory and an interactive run targeting --remote gh
  When `instill init` fails during Phase 2
  Then init stops with "init failed at step 'instill init'"
    And the message names the dir and the recursive-force-delete recovery command
    And no GitHub repo was created (Phase 3 never started)
```

### 1.3 Phase 3 remote failure — leave & instruct, never auto-delete

Phase 3 is the only step with an external side effect, so it has its own rule:

- **`gh repo create` fails** (name taken / auth / network) → report the reason; the project
  remains a complete local-only repo; tell the user how to add a remote manually.
- **repo created but the first push fails** (gate trips / network) → **leave the remote**,
  print its URL and the `git push -u origin <branch>` retry; mention `gh repo delete` as
  *their* option. mkproj never auto-deletes a remote (irreversible outward action; the most
  common cause is a gate working as designed). The local repo is left complete and intact.

```gherkin
Scenario: Remote created but first push fails (gate or network)
  Given Phase 3 created the GitHub repo successfully
  When the initial `git push` fails
  Then mkproj reports the remote URL and the push failure reason
    And it prints `git push -u origin <branch>` as the retry
    And it does NOT delete the remote
    And the local repo is left complete and intact

Scenario: Remote creation itself fails
  Given `gh repo create` fails
  When Phase 3 runs
  Then mkproj reports the reason and the project remains a complete local-only repo
    And the user is told how to add a remote manually
```

## 2. Deterministic `.gitignore` merge (`iha`)

Output = base section, a banner, then the language section — **concatenated verbatim**,
base first. No interleaving, no sorting, **no dedup** (git treats duplicate patterns as
harmless; parsing/normalizing would add bugs and non-determinism). Each section is
byte-identical to its source file, so the same inputs always produce byte-identical output.

```gherkin
Scenario: Gitignore merge is deterministic and sectioned
  Given the base .gitignore and the vendored Python github/gitignore
  When the Phase-1 writer produces the project .gitignore for a Python stack
  Then the output is the base section, a "# ===== python ... =====" banner, then the python section
    And each section is byte-identical to its source file
    And running the merge twice produces byte-identical output

Scenario: Overlapping patterns are kept, not deduped
  Given both source files contain ".DS_Store"
  When the merge runs
  Then ".DS_Store" appears in both sections
    And git ignore behavior is unaffected
```

## 3. Managed-block topology (`yuw`, `rom`)

Three managed blocks across two files, coexisting by **unique named marker**. Each
reconciler rewrites only between its own markers; outside-block and inter-block content is
sacred and hand-editable.

| File | Block | Owner | Reconciled? |
|---|---|---|---|
| `AGENTS.md` | beads block | `bd` | bd's own cadence |
| `AGENTS.md` | `MKPROJ CONVENTIONS` (Co-Authored-By, conventional-comments, <300-line PR norm) | mkproj | **render-once** — no version, no reconciler (v1) |
| `.claude/settings.local.json` | `MKPROJ ALLOW v:N` | `mkproj sync-allowlist` | versioned + reconciled |

Only the allowlist block carries version + reconciler machinery (keeps `rom` scoped to one
block). The conventions block is written once at init and is then the user's to edit.

```gherkin
Scenario: Multiple managed blocks coexist without clobbering
  Given an AGENTS.md with a beads block, a MKPROJ CONVENTIONS block, and hand-written prose between them
  When `mkproj sync-allowlist` runs (targets settings.local.json, not AGENTS.md)
  Then AGENTS.md is untouched
    And only the allowlist block in settings.local.json is rewritten
    And hand-written prose and the beads block are byte-identical
```

## 4. Dual-agent SessionStart wiring (`yuw`)

### 4.1 Hook chain — both agents, full parity

Both Claude (`.claude/settings.json`) and Codex (`.codex/hooks.json`) get the **same three
SessionStart hooks in the same order**:

1. `bd prime` (beads context)
2. `instill check-skills` (skill-symlink reconciliation)
3. `mkproj sync-allowlist --check` (staleness notice — advisory, last so it is most visible)

Order is for readability; no hook depends on another's output. Codex parity is real: instill
manages Codex skills too (it regenerates `.agents/skills/` as the Codex-side equivalent of
`.claude/skills/`).

### 4.2 Non-fatal + silent-missing-binary

Every mkproj-authored SessionStart hook **exits 0 always**, and a **missing `mkproj` binary
no-ops silently** (`command -v mkproj || exit 0`) — no "not installed" hint, zero noise. A
scaffolded repo opens cleanly on a collaborator's machine that never installed mkproj.

```gherkin
Scenario: SessionStart hooks run in order on both agents
  Given a scaffolded repo opened in Claude Code (and likewise in Codex)
  When SessionStart fires
  Then `bd prime`, then `instill check-skills`, then `mkproj sync-allowlist --check` run in that order

Scenario: Missing mkproj binary never breaks session start and is silent
  Given a collaborator's machine without mkproj on PATH
  When SessionStart fires the `mkproj sync-allowlist --check` hook
  Then the hook exits 0 and prints nothing
    And bd prime and instill check-skills still ran
```

## 5. Skill-manifest topology (`yuw`)

A **single** committed `.claude/skill-manifest.json` is the sole lockfile. `instill
check-skills` regenerates **both** `.claude/skills/` and `.agents/skills/` from it (instill
lands only one manifest file and populates both trees). Both symlink trees are gitignored;
the manifest is the committed portable lockfile (design §8). No per-agent manifest, no drift.

```gherkin
Scenario: One manifest feeds both agents' skill trees
  Given a single committed .claude/skill-manifest.json
  When `instill check-skills` runs on a fresh clone
  Then .claude/skills/ and .agents/skills/ are both regenerated from the manifest
    And both trees contain the same skill set
```

## 6. CI workflow shape (`ud1`)

One **language-agnostic** `ci.yml`, identical across all stacks: checkout → install mise →
`mise install` → `mise run ci`. No inline lint/test/format (mise absorbs all language
variance). Triggers: `on: [push, pull_request]` — no scheduled run in v1. Verification
asserts every shipped overlay's `mise.toml` defines a `ci` task that `mise run ci` resolves
to (the contract coupling `ud1` to `x2k`).

```gherkin
Scenario: One CI workflow calls mise run ci and references a real task
  Given any shipped v1 stack
  When its scaffolded .github/workflows/ci.yml is inspected
  Then it checks out, installs mise, runs `mise install`, then `mise run ci`
    And it contains no inline lint/test/format commands
    And the stack's mise.toml defines a `ci` task that `mise run ci` resolves to
```

## 7. Smoke verification scope (`uuw`, `3tt`)

Two tiers proving the spec's "indistinguishable from hand-built" promise:

- **Local smoke (every run, hermetic):** `mkproj init --stack <key> --remote none` (smoke is
  the primary consumer of `--remote none`) → `mise install` → `mise run ci` exits 0 → clean
  initial commit through the pre-commit gates → assert **no network access** occurred.
- **Full golden path (gated behind gh credentials; nightly/opt-in):** `--remote gh` end to
  end → remote created → first push passes the pre-push pipeline → CI green on the remote.

Because `mise run ci` includes `test` (ADR-0003), green CI inherently proves the snapshot's
starter test passes — therefore **every shipped golden snapshot MUST ship at least one real
passing test** (owned by `3tt`), so `mise run test` is never vacuously green.

```gherkin
Scenario: Local smoke proves a scaffolded stack works offline
  Given an empty dir
  When `mkproj init --stack <key> --remote none` runs, then `mise install`, then `mise run ci`
  Then the repo contains mise.toml, lefthook.yml, and ci.yml
    And `mise run ci` exits 0 (running >=1 real test)
    And an initial commit succeeds through the pre-commit gates
    And no network/GitHub access occurred

Scenario: Snapshot tests are not vacuous
  Given any shipped v1 stack
  When its starter test is removed and `mise run test` is run
  Then `mise run test` fails
```

---

*Authored By Peter O'Connor with Assistance from Claude Code (databricks-claude-opus-4-8) · 2026-06-19 · mkproj init lifecycle, asset topology & verification contract*
