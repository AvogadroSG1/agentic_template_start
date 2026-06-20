# CONTEXT — mkproj

Glossary of domain terms for the `mkproj` scaffolding system. Definitions only — no implementation details, no decisions (those live in `docs/adr/`).

## Terms

### mkproj
The scaffolding CLI. One command initializes a fully configured project (git, agentic config, beads, skills) with zero follow-up steps.

### Golden snapshot
A pinned, vendored copy of a recipe's output captured at maintainer time, unpacked and templated at init. Not a live invocation. The recipe is most often a single ecosystem-native scaffolder run (e.g. `cobra-cli init`), but may be a multi-step recipe — a pinned git checkout plus dependency pins (e.g. `go-api-chi` = `golang-standards/project-layout` at a SHA + `go get` of chi/zap/viper/testify). Either way the captured output is the vanilla layer beneath the overlay.

### Walking skeleton
The acceptance guarantee on a shipped stack's *composed* output (vanilla layer + overlay): after `mkproj init` the repo runs end-to-end and has at least one real passing test — a thin vertical slice that actually walks (router up, logger wired, one green handler test for an API stack). It is an emergent property of vanilla+overlay composed, **not** a third layer. The vanilla/overlay split is preserved beneath it so `mkproj update` can refresh the vanilla layer without re-vetting the wiring.

### Allowlist
The curated set of Bash command prefixes the author has vetted as safe to run without a confirmation prompt. Convenience-oriented and **always growing**. Refreshes from a canonical source embedded in the `mkproj` binary, written into a versioned managed block in each project. Distinct from the deny floor.

### Deny floor
The small, stable, safety-oriented set of rules that block irreversible or dangerous commands. Enforced by the guard hook. Rarely changes. Distinct from the allowlist; shares a single canonical source file (different sections) but refreshes on its own cadence.

### Managed block
A delimited region (`<!-- BEGIN MKPROJ ALLOW v:N --> … <!-- END --> `) that the reconciler may rewrite in place, leaving surrounding hand-edited content untouched. Mirrors the existing beads integration block in `AGENTS.md`.

### Reconciler
A `mkproj` subcommand (`sync-allowlist`) that rewrites a project's managed block from the canonical embedded source. Triggered notify-only: a SessionStart hook detects staleness and prompts the author to run it; it never auto-mutates the repo.

### Guard hook
A self-contained PreToolUse hook shipped in-repo (`.claude/hooks/guard`), wired identically by both Claude Code and Codex via their `PreToolUse` events (both honor `exit 2` / deny). Enforces the deny floor only — it is a **deny-only net**, never an allow-decider. Runs in every permission mode; auto mode bypasses the confirmation prompt but never the guard.

### Constituent
One command within a compound shell line (split on `&&`, `||`, `;`, pipes). The guard judges each constituent for deny rules; one denied constituent blocks the whole line. The guard does NOT approve compounds — auto-run of a compound depends solely on an allow glob matching the whole line.

### Secret-exposure guard
A target-aware deny rule: blocks display/search commands (`cat`, `grep`, `rg`, `head`, `tail`, `less`, `awk`, `xxd`) against secret-path patterns (`.env*`, `*.pem`, `*.key`, `credentials`, `*.tfstate`, …) and blocks unfiltered environment dumps (`env`, `printenv`, `set`). Path patterns are configurable at the top of the guard. Distinct from content secret-scan, which guards commits.

### Overlay (.mkproj-overlay/)
The single layer on top of a vanilla golden snapshot that adds the author's vetted
*opinions*: formatter, linter, test framework, mocking framework, coverage tool, type
checker, audit/security tooling, recommended packages, gate wiring, CI. The snapshot is
inert scaffolding; the overlay is where the value lives. Its tool/package choices are
governed by (and tested against) the canonical language guideline files. There is exactly
**one** overlay per stack — the earlier name "security overlay" is retired; audit/security
tooling is one *part* of the overlay, not a separate layer. `mkproj update` regenerates the
snapshot beneath it but never touches the overlay.

### Gate
An automated quality check (lint, format, test) defined once as a `mise` task and invoked
by multiple callers (lefthook locally, GitHub Actions in CI) so the definition never
drifts. Fast gates (lint/format) run on pre-commit; full tests run on pre-push and in CI.

### Guideline file (canonical)
`~/peter_code/ai_support/guidelines/{golang,python,csharp}.md` — the author's written
language standards. The **minimum source of truth** (the *floor*) for what a template's
overlay installs: the overlay MUST implement every guideline MUST/SHOULD and MAY add vetted
extras the guideline is silent on. A lightweight conformance test asserts the floor — it
fails when a guideline MUST has no corresponding overlay tool, not when the overlay ships an
extra. A template may only ship for a language that has a guideline file (v1: Go, Python, C#).

### Skill manifest vs. symlinks
The `instill` manifest (`.claude/skill-manifest.json`) is the **committed, portable** declaration of which skills the repo uses (the lockfile). The `.claude/skills/` symlinks are **machine-local** and gitignored; `instill check-skills` regenerates them on clone (the node_modules).
