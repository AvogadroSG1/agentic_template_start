# Value Stream First Slices Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` for task execution and `superpowers:requesting-code-review` plus `superpowers:verification-before-completion` before any task is marked done.

**Goal:** Land the three leaf slices from `docs/handoffs/2026-06-20-junior-engineer-brief.md` so downstream work can build on explicit guideline coverage, an executable v1 catalog boundary, and a tested `scan-command` secret-scan slice.

**Architecture:** Keep each slice independently testable and narrowly scoped. `6ms` edits the canonical cross-repo guideline files only, `01b` creates a small Go catalog package with BDD-style tests that future prompt and scaffold code can consume, and `aa9` creates a standalone shell scanner with a Bats harness and no guard/lefthook coupling beyond its CLI contract.

**Tech Stack:** Markdown guidelines, Go unit tests, Bash, `bats`, `jq`

## Global Constraints

- The junior engineer brief at `docs/handoffs/2026-06-20-junior-engineer-brief.md` is the source of truth.
- BDD and TDD are mandatory: tests/scenarios MUST be written before or alongside implementation, never after the fact.
- `aa9` MUST NOT create `lefthook.yml` or `.claude/hooks/guard`.
- `01b` MUST make only the six v1 stacks selectable: `go-cli-cobra`, `go-api-chi`, `python-cli-typer`, `python-fastapi`, `csharp-cli`, `csharp-webapi`.
- `6ms` MUST update the canonical guideline files under `/Users/poconnor/peter_code/ai_support/guidelines/`.
- Each task MUST expose a clean seam for downstream work rather than baking later phases into this slice.

---

### Task 1: `6ms` Guideline Audit/Coverage Coverage

**Files:**
- Modify: `/Users/poconnor/peter_code/ai_support/guidelines/golang.md`
- Modify: `/Users/poconnor/peter_code/ai_support/guidelines/python.md`
- Modify: `/Users/poconnor/peter_code/ai_support/guidelines/csharp.md`

**Interfaces:**
- Consumes: overlay tool table from `docs/superpowers/specs/2026-06-19-mkproj-cli-prompt-render-contract.md` section 7
- Produces: canonical MUST/SHOULD lines that downstream conformance tests (`ebp`) can parse against overlay installs

- [ ] Write/update the BDD scenario text in the worker notes so the edit is traced to the brief:
  `Given the three guideline files, Then each overlay tool traces to a MUST/SHOULD line.`
- [ ] Add an `Audit/Security + Coverage` section to each guideline file covering the missing tools:
  Go: `govulncheck`, `go test -cover`
  Python: `pip-audit`, `pytest-cov`, `pyright`
  C#: `StyleCop.Analyzers`, `dotnet list package --vulnerable`, `coverlet`, `dotnet format`
- [ ] Verify by reading the edited sections back and matching them against the overlay table in spec section 7.

### Task 2: `01b` Executable V1 Catalog Boundary

**Files:**
- Create: `go.mod`
- Create: `internal/catalog/catalog.go`
- Create: `internal/catalog/catalog_test.go`

**Interfaces:**
- Consumes: v1 catalog decisions from `docs/superpowers/specs/2026-06-16-mkproj-scaffolding-system-design.md` and picker behavior from `docs/superpowers/specs/2026-06-19-mkproj-cli-prompt-render-contract.md`
- Produces:
  - `type Stack struct { Key, Language, ProjectType string }`
  - `func V1Stacks() []Stack`
  - `func SelectableStacks(language, projectType string) []Stack`
  These functions are the seam future prompt/catalog code can call without re-encoding the v1 list.

- [ ] Write failing Go tests for the six-stack exact list and filtered picker behavior before creating implementation code.
- [ ] Implement the minimal catalog package and module metadata needed to make those tests pass.
- [ ] Run `go test ./...` and confirm only Go, Python, and C# stacks are selectable while TS/Rust/Bash remain non-selectable.

### Task 3: `aa9` Secret Scan Command Slice

**Files:**
- Create: `.claude/hooks/secret-scan.sh`
- Create: `mise.toml`
- Create: `test/secret-scan.bats`

**Interfaces:**
- Consumes: D9/D10 rules from `docs/superpowers/specs/2026-06-18-allowlist-deny-floor-seed.md` and the command-mode contract from `docs/superpowers/specs/2026-06-18-shared-secret-scan-design.md`
- Produces:
  - `.claude/hooks/secret-scan.sh scan-command`
  - `--command "<line>"` fallback for direct testing
  - Exit contract: `0` clean, `2` blocked, `64` usage

- [ ] Write failing Bats scenarios first for D9 secret-path blocking, D10 env-dump blocking, carve-outs, empty stdin, and bad subcommand handling.
- [ ] Implement the smallest script that passes those tests while documenting the irreducible gaps called out in the spec.
- [ ] Run the focused Bats suite and then the full repo verification commands used by this slice.

### Review Gates

- [ ] For each task, dispatch a worker with its exact file ownership and ask it to report tests run plus changed files.
- [ ] For each completed task, dispatch a review agent to validate both brief compliance and code quality before integrating the result.
- [ ] Before completion, run fresh verification locally for every changed slice and only then close beads issues, commit, and push.
