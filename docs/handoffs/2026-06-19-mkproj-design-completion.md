# Handoff — mkproj design completion (grill-with-docs)

**Date:** 2026-06-19 · **Branch:** `chore/remove-calm-guard-refs` · **Repo:** agentic_template_start

## What this session did

Ran `grill-with-docs` to close the design gaps a junior engineer flagged in the mkproj
scaffolding system. Resolved **17 design questions** (Q1–Q17), each ending in runnable
Given/When/Then acceptance criteria so a guided implementer can prove a slice works by
running it. All decisions are committed and pushed (working tree 0/0 with origin).

## Artifacts produced (read these, don't re-derive)

- `docs/superpowers/specs/2026-06-19-mkproj-cli-prompt-render-contract.md` — Q1–Q9:
  variable schema (seed vs derived), command surface, prompt order + three-state flag
  precedence, render contract (`missingkey=error`, `.tmpl` convention, verbatim security
  scripts), starter skill manifest (universal 23 + per-stack slices), reconciler staleness
  (`ALLOW_VERSION` int) + SessionStart notify, overlay tool table per ecosystem, `mkproj
  update` automation.
- `docs/superpowers/specs/2026-06-19-mkproj-init-lifecycle-and-topology.md` — Q10–Q17:
  strict empty-dir precondition, fail-fast/no-rollback, Phase-3 remote-failure (leave &
  instruct), deterministic gitignore merge, 3-block managed-block topology, dual-agent
  SessionStart (full parity, non-fatal, silent-missing-binary), single shared skill
  manifest feeding both `.claude/skills/` + `.agents/skills/`, language-agnostic CI,
  two-tier smoke verification.
- `docs/adr/0004-init-fail-fast-no-rollback.md` — the no-rollback decision + why.
- `CONTEXT.md` — "security overlay" retired; one overlay (audit tooling is a *part* of it).
- `~/peter_code/ai_support/guidelines/csharp.md` — **cross-repo fix**, committed+pushed
  separately (commit `fadb201` on `ai_support` main): NUnit → xUnit in all 3 spots.
- bd notes on `s5x`, `wer`, `yuw`, `ud1`, `uuw`, `3tt` carry the per-issue decisions.
- New bd issue `6ms` (guideline Audit/Coverage sections) created, wired to **block `ebp`**.

## Design-completeness audit (the open question at handoff)

15 of 18 issues are build-ready (🟢). Three 🟡:
- `x2k` — not a gap; correctly blocked on the `485` chain-mode verification spike.
- **`3tt` (Gap A) — REAL undefined surface.** No issue has pinned the six concrete
  stack-key → upstream-source mappings (exact `cobra-cli`/`uv`/`dotnet` invocations, which
  `github/gitignore` file per lang, which ref/SHA in `sources.yaml`, what goes in
  `go-api-chi`). This is a design choice with trade-offs, not mechanical copy.
- **`cjl` (Gap B) — softer.** Q9 settled *what* `mkproj update` does and that the overlay
  is untouched, but not the snapshot **normalization/determinism contract** (how to strip
  timestamps/generated-comment variance so re-runs diff cleanly).

User declined to grill A/B further this session and asked for this handoff. My standing
recommendation: **grill Gap A before implementing `3tt`** (re-vendoring is expensive if the
source mapping is wrong); **Gap B is acceptable as implementation-discovery** (find the
normalization rules by running the scaffolders).

## Recommended next move

Either (a) **grill Gap A** to fully close design, or (b) **start building** the first
runnable slice. Best first build slice: **`aa9`** (secret-scan `scan-command`) — a leaf,
pure bash + bats, 6 sharp BDD criteria, spec at
`docs/superpowers/specs/2026-06-18-shared-secret-scan-design.md`. Trivial warm-up: **`6ms`**
(20-min doc task, unblocks `ebp`).

## Environment notes / gotchas

- This repo enforces **bd (beads)** for all task tracking — NOT TaskCreate/TodoWrite.
  Run `bd ready` / `bd show <id>`. Issues prefixed `agentic_template_start-`.
- A **guard hook** (`guardrails-bin`) blocks: literal `rm -rf` in command strings (use
  `trash` or explicit paths), reading secret paths, and **writing files while there are
  staged changes** — commit/stash before Write/Edit if the guard complains.
- bd auto-commits `.beads/issues.jsonl`; an auto-checkpoint hook commits doc edits as
  `[claude-auto]`. Files may already be committed before you `git add`. No Dolt remote
  configured — `git push` carries bd data.
- Pre-existing untracked noise (leave alone): `.DS_Store`, `.agents/`,
  `.claude/skill-manifest.json` modifications from session start.
- Commit co-authors required (both): Peter O'Connor + the CLI tool/model.

## Suggested skills for the next session

- `superpowers:test-driven-development` — for building `aa9`/any slice (red-green-refactor
  is mandated; write failing bats tests first).
- `ai-workflow:grill-with-docs` — if closing Gap A (`3tt` source mapping) before building.
- `superpowers:verification-before-completion` — before claiming any slice done.
- `productivity:mise` — when wiring `mise.toml` tasks (`x2k`) or the bats harness.
- `superpowers:finishing-a-development-branch` — when the build work is ready to integrate.
