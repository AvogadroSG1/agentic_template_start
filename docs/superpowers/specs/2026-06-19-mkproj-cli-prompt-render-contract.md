# mkproj CLI, Prompt & Render Contract + remaining §9 open items

**Status:** decided 2026-06-19 (grill-with-docs) · **Author:** Peter O'Connor with Claude Code (databricks-claude-opus-4-8)

Closes issue **agentic_template_start-83n** and resolves the remaining open items in
[the mkproj design spec §9](2026-06-16-mkproj-scaffolding-system-design.md#9-open-items-for-implementation-planning):
minimal starter skill set, overlay contents per ecosystem, the exact `text/template`
variable schema + prompt sequence, `mkproj update` automation, and reconciler
staleness detection + SessionStart wiring.

This document is written for an implementer who can build with guidance but is not the
designer: **every decision ends in a Given/When/Then acceptance scenario that defines
"done."** If the scenarios pass, the slice works.

---

## 1. Variable schema — seed vs. derived

**Seed (prompted/flagged, un-derivable):** project name, language, type, stack, remote
choice. **Author identity** auto-defaults from `git config --global user.{name,email}`
and is prompted only if absent.

**Derived** (computed from the project-name slug; each has an override flag, never a
prompt):

| Variable | Derivation | Override flag |
|---|---|---|
| `BdPrefix` | lowercased, non-alphanumeric-stripped slug | `--bd-prefix` |
| `ModulePath` | `github.com/<gh-user>/<ProjectName>` when `--remote gh`; else editable placeholder | `--module-path` |
| Go module / Python package (`snake_case`) / C# namespace (`PascalCase`) | normalized slug | (language-specific) |

```gherkin
Scenario: Derivations from a single project-name seed
  Given the project name "My Cool API"
  When mkproj resolves the variable set
  Then BdPrefix == "mycoolapi"
    And the Python package == "my_cool_api"
    And the C# namespace == "MyCoolApi"

Scenario: A derived value can be overridden without a prompt
  Given the project name "My Cool API" and flag --bd-prefix=mcapi
  When mkproj resolves the variable set
  Then BdPrefix == "mcapi"
    And no prompt for the bd prefix is shown
```

## 2. Command surface

- `mkproj` — **defaults to `init`** (bare invocation == `mkproj init`).
- `mkproj init` — scaffold a new project in the current dir (Phases 1–3).
- `mkproj update` — maintainer-only snapshot refresh (§7).
- `mkproj sync-allowlist [--check]` — reconciler (§6).

```gherkin
Scenario: Bare invocation defaults to init
  Given an empty directory
  When the user runs `mkproj`
  Then it behaves identically to `mkproj init`
```

## 3. Prompt flow & precedence

**Order:** project name → language → type → stack → author identity (confirm-or-edit of
the git-config default) → remote (last, because it is the one outward-facing action).
Type is filtered to what exists for the chosen language; stack to the chosen
language × type. Only the shipped v1 stacks appear.

**Three-state precedence** (per input):

1. flag present → use it, skip the prompt silently (no re-confirm)
2. flag absent + TTY → prompt
3. flag absent + no TTY → **error naming the missing flag** (never fall back to a default)

Invalid flag value → **fail immediately listing valid choices**, never drop to a prompt.

```gherkin
Scenario: Fully non-interactive run when every value is flagged
  Given flags supplying name, language, type, stack, author, and remote
  When mkproj runs with no TTY
  Then no prompt is shown and the project is scaffolded

Scenario: Missing required value with no TTY fails loudly
  Given no --stack flag and no TTY
  When mkproj runs
  Then it exits non-zero naming --stack as the missing flag

Scenario: Invalid stack fails with the valid list
  Given --stack=nonexistent
  When mkproj runs
  Then it exits non-zero listing the valid v1 stack keys
    And no interactive prompt is shown

Scenario: Only v1 stacks are selectable
  Given an interactive run with language=Go
  When the stack picker renders
  Then only Go v1 stacks (go-cli-cobra, go-api-chi) are offered
```

## 4. Render contract

- **Engine:** Go `text/template` with **`Option("missingkey=error")`** — an unresolved
  variable is a hard failure, never `<no value>`.
- **Renderable marker:** filename ends in **`.tmpl`** → render and strip the suffix.
  Every other file is copied **verbatim** (byte-for-byte).
- **File roles** (from design §7): `render` (`.tmpl`), `verbatim`, `link` (symlink),
  `delegate` (produced by `bd`/`instill`/native tooling).
- The guard hook and `secret-scan.sh` are **`verbatim`** (security-critical scripts are
  identical and independently auditable across every repo; never templated).

```gherkin
Scenario: Phase-1 writer produces a clean tree (iha definition-of-done)
  Given a fixture variable set and the embedded templates/ tree
  When the Phase-1 scaffold writer runs into a temp dir
  Then no rendered file contains "<no value>" or a residual "{{"
    And every verbatim file is byte-identical to its source
    And CLAUDE.md is a symlink whose target is AGENTS.md
    And .claude/hooks/guard and .claude/hooks/secret-scan.sh are mode 0755

Scenario: A missing variable fails the render
  Given a template referencing {{.Undefined}}
  When the writer renders it
  Then init fails with an error naming the missing key
    And no partial output file is written
```

## 5. Starter skill manifest

**Universal tier (23 skills, every repo, committed identically):**
`ai-workflow/{grill-with-docs, handoff, improve-codebase-architecture, tdd, to-issues,
to-prd, triage}`, `obsidian/obsidian-cli`, `productivity/{mise,
ubiquitous-language-registry}`, `superpowers/{brainstorming, dispatching-parallel-agents,
executing-plans, finishing-a-development-branch, receiving-code-review,
requesting-code-review, subagent-driven-development, systematic-debugging,
test-driven-development, using-git-worktrees, using-superpowers,
verification-before-completion, writing-plans}`.

**Per-stack slice (pre-selected; the user edits in the TUI):**

| Stack | Slice |
|---|---|
| Go | `golang/{golang-cli (CLI only), golang-code-style, golang-testing, golang-error-handling, golang-project-layout, golang-linter}` |
| Python | `python/{python-pro, python-code-style, uv-package-manager, python-testing-patterns, python-type-safety, python-project-structure}` |
| C# | `dotnet/{csharp, dotnet-test-frameworks, dotnet-webapi (API only), convert-to-cpm, run-tests}` |

**Skill-picking behavior (Phase 2):**

- **Interactive (TTY):** Phase 2 shells out to `instill init` and lets its **TUI take
  over, pre-seeded with the stack defaults** (universal + slice already selected).
- **Non-interactive (flags / no TTY):** `instill init --skills <universal + slice>` — no
  TUI.

```gherkin
Scenario: Interactive skill pick is pre-seeded then user-driven
  Given an interactive Go-CLI run
  When Phase 2 reaches skill picking
  Then instill's TUI launches with the universal set + Go slice pre-selected

Scenario: Non-interactive skill pick bypasses the TUI
  Given a no-TTY Go-CLI run
  When Phase 2 reaches skill picking
  Then `instill init --skills <universal+go-slice>` runs with no TUI
    And skill-manifest.json contains exactly that set
```

## 6. Reconciler staleness detection + SessionStart notify (both agents)

**Detection: monotonic integer.** The binary embeds `ALLOW_VERSION`; the managed block
carries `v:N`. `embedded > block` → stale. A guard test asserts "if the seed file changed
in a commit, `ALLOW_VERSION` changed too" (forgot-to-bump backstop).

**Notify: SessionStart hook, both agents, always exit 0.** A shared
`mkproj sync-allowlist --check` is wired into Claude (`.claude/settings.json`
SessionStart) and Codex (`.codex/hooks.json` SessionStart). On staleness it prints a
one-line notice; when current it prints nothing. It **never** mutates and **never** blocks
session start. The mutating `mkproj sync-allowlist` (no `--check`) stays human-invoked.

```gherkin
Scenario: Stale allowlist notifies without blocking
  Given a managed block at v:5 and embedded ALLOW_VERSION = 7
  When the SessionStart hook runs `mkproj sync-allowlist --check`
  Then it prints "allowlist is 2 versions behind; run `mkproj sync-allowlist` to refresh"
    And it exits 0
    And the managed block is unchanged

Scenario: Current allowlist is silent
  Given a managed block at v:7 and embedded ALLOW_VERSION = 7
  When the SessionStart hook runs `mkproj sync-allowlist --check`
  Then nothing is printed and it exits 0

Scenario: Mutation is human-invoked only
  Given a stale repo
  When the SessionStart hook runs
  Then the managed block is never rewritten by the hook
```

## 7. Overlay contents per ecosystem (one overlay; "security overlay" retired)

There is exactly **one** `.mkproj-overlay/` per stack. Every tool traces to a canonical
guideline file (ADR-0003); the conformance test (`ebp`) fails the build on drift.

| | Format | Lint | Test | Mock | Coverage | Type | Audit |
|---|---|---|---|---|---|---|---|
| **Go** | `gofmt` | `golangci-lint` | `testing`+`go-cmp` | *(none — idiomatic fakes)* | `go test -cover` | — | `govulncheck` |
| **Python** | `ruff format` | `ruff` | `pytest` | `pytest-mock` | `pytest-cov` | `pyright` | `pip-audit` |
| **C#** | `dotnet format` | StyleCop.Analyzers | **xUnit** | NSubstitute | coverlet | nullable refs | `dotnet list package --vulnerable` |

> **Precursor task (blocks `ebp`):** several tools above are not yet named in the guideline
> files. Add an "Audit/Security + Coverage" section to `golang.md`, `python.md`, `csharp.md`
> naming StyleCop, govulncheck/pip-audit/`dotnet list --vulnerable`, the coverage tools,
> pyright (Python picks pyright over mypy), and `dotnet format` — so every overlay tool
> traces to a written MUST/SHOULD. The NUnit→xUnit contradiction is already fixed in
> `csharp.md`.

```gherkin
Scenario: Overlay installs exactly the guideline-mandated tools
  Given a scaffolded Python project with its overlay applied
  When the ebp conformance test parses python.md against pyproject.toml + mise tasks
  Then ruff, pyright, pytest, pytest-cov, pytest-mock, and pip-audit are present
    And no tool absent from python.md is installed
```

## 8. `mkproj update` snapshot-capture (maintainer path)

Refreshes all stacks by default; `--stack <key>` scopes to one. Each run: invoke the
pinned native scaffolder in a temp dir → strip build artifacts (`node_modules`, `bin/obj`,
`target/`) → write the snapshot into `templates/golden/<key>/` → record the ref/SHA in
`sources.yaml`. It **regenerates the vanilla snapshot only — never `.mkproj-overlay/`** —
and **fails loudly** naming any missing native scaffolder. Rebuilding the binary
(`go build`) is a separate manual step after the maintainer reviews the diff.

```gherkin
Scenario: Update refreshes the snapshot but preserves the overlay
  Given the maintainer repo with a pinned go-cli-cobra snapshot and overlay
  When the maintainer runs `mkproj update --stack go-cli-cobra`
  Then templates/golden/go-cli-cobra/ reflects new cobra-cli output
    And .../go-cli-cobra/.mkproj-overlay/ is byte-identical to before
    And sources.yaml records the captured ref/SHA

Scenario: Update fails loudly when a scaffolder is missing
  Given cobra-cli is not on PATH
  When the maintainer runs `mkproj update --stack go-cli-cobra`
  Then it exits non-zero naming cobra-cli as missing
    And no snapshot or sources.yaml entry is modified
```

---

## 9. Open-item closure map

| §9 open item | Resolved in |
|---|---|
| minimal starter skill set per stack | §5 |
| overlay contents per ecosystem | §7 |
| exact template variable schema + prompt sequence | §1, §3, §4 |
| mkproj update snapshot-capture automation | §8 |
| reconciler staleness detection + SessionStart wiring | §6 |
| (design line 212 partial prompt/render detail) | §3, §4 |

---

*Authored By Peter O'Connor with Assistance from Claude Code (databricks-claude-opus-4-8) · 2026-06-19 · mkproj CLI, prompt & render contract*
