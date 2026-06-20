> **⚠️ SUPERSEDED 2026-06-20 by [`/docs/SPEC.md`](../../../SPEC.md).** Retained for history only — do not implement from this document. See SPEC.md §0.1 for the section that subsumes this content.

# mkproj — Reusable Git Project Initialization Template System

**Status:** Design approved (brainstorming complete) · **Date:** 2026-06-16
**Author:** Peter O'Connor with Claude Code (databricks-claude-opus-4-8)

---

## 1. Problem & Goal

Every new project begins with 10–15 minutes of identical manual setup: `git init`,
copy a known-good Claude/beads config from a prior project, prune unwanted skills,
re-add starter skills, link the agents file via a Claude `.md` symlink, and install +
configure beads with a known-good `settings.local.json`. This is pure, repeatable
overhead.

**Goal:** one command → a fully initialized project, indistinguishable from what the
author would have built by hand, with **zero** follow-up configuration.

**Verification test:** run the init command in an empty folder; the result must be
indistinguishable from a hand-built project — no manual steps after.

### Terminology reconciliation

The original brief used exploratory names. They map to concrete artifacts in this repo:

| Brief term | Actual artifact |
|---|---|
| "Beats" / "BSRC" | **beads** (`bd`) — issue tracker |
| "agents file linked to a Claude `.md`" | `CLAUDE.md` → `AGENTS.md` symlink |
| "`.local.json` with known-good defaults" | `.claude/settings.local.json` `permissions` block |
| skill prune/re-add workflow | **`instill`** CLI (`init`, `pick-skills`, `check-skills`) |

A key early finding: **`instill` and `bd` already perform most of the scaffolding work.**
`mkproj` therefore orchestrates them rather than reimplementing them.

---

## 2. Engine & Distribution Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Scaffolding engine | **Standalone Go binary** using `text/template` + `embed.FS` | Self-contained, single tool owns templating end-to-end; sibling to existing `gw`/`instill` tools. |
| Distribution | **Installed binary** (`~/.local/bin/mkproj`), templates embedded | No clone, no machine-path dependency; works in any empty dir. |
| Template/asset sourcing | **Vendored + refreshable** | Init is fully **offline** and **reproducible**; `mkproj update` re-fetches and re-pins upstreams under maintainer control. |
| Selection UX | **Interactive picker** (language → type → stack), flags may pre-answer prompts | Simple mental model; flags preserved so automation stays possible. |
| Deny-list enforcement | **Both** `settings.local.json` permissions **and** a self-contained PreToolUse guard hook | Defense in depth. |

**Explicitly rejected:** a from-scratch templating engine (reinvents `bd`/`instill`);
pure shell `init.sh` (weaker templating/validation); Cookiecutter/Copier as the engine
(adds a Python runtime dependency and a second templating syntax — kept only as
acknowledged prior art we deliberately do not depend on).

---

## 3. System Architecture

`mkproj` runs init in three ordered phases. It **owns** templating, verbatim copy,
symlink creation, permissions, and guard-hook installation; it **delegates** to `bd`
and `instill` for what they already do well.

```mermaid
flowchart TD
    A["mkproj  (interactive picker: lang -> type -> stack)"] --> B

    subgraph P1["Phase 1 — Scaffold (mkproj owns)"]
        B["git init + git config identity"] --> C["Render templated files (text/template + embed.FS)"]
        C --> D["Copy verbatim files (.gitignore base, hooks, settings.json)"]
        D --> E["Overlay vendored OSS .gitignore + golden starter snapshot for stack"]
        E --> F["Create CLAUDE.md -> AGENTS.md symlink"]
        F --> G["Write .claude/settings.local.json (allow/deny permissions — Layer A)"]
        G --> H["Install self-contained guard hook (PreToolUse — Layer B)"]
    end

    subgraph P2["Phase 2 — Delegate to existing tools"]
        H --> I["bd init -> .beads/, git hooks, AGENTS.md beads block"]
        I --> J["instill init + pick-skills (minimal set) + check-skills -> skill symlinks"]
    end

    subgraph P3["Phase 3 — Remote (optional)"]
        J --> K{"--remote?"}
        K -->|"gh"| L["gh repo create + git remote add origin"]
        K -->|"url"| M["git remote add origin <url>"]
        K -->|"none"| N["local-only"]
        L --> O["initial commit + push"]
        M --> O
        N --> Pp["initial commit (no push)"]
    end
```

**Phase-ordering note:** the guard hook installs **before** `bd init` so that beads'
own git-hook installation runs under the guard, not the reverse.

**Remote-creation note (decided 2026-06-17):** Phase 3 runs **last — after everything is
installed and the lefthook gates are wired** — so `gh repo create` publishes a repo that
is already complete, and the *initial commit + push* passes through the full pre-commit /
pre-push pipeline (secret-scan, lint, format, tests) before anything reaches the remote.
The default for an interactive run is **`--remote gh`**: `gh repo create <name> --source .
--remote origin` followed by `git push -u origin <branch>`. `--remote url` and
`--remote none` remain available for explicit-URL and local-only flows. Creating the
remote is part of the standard "day one it just works" path, not a manual afterthought.

### `mkproj update` (maintainer path)

Keeps init offline/reproducible while allowing controlled refresh: re-fetches vendored
`.gitignore` files and regenerates golden snapshots from native scaffolders, then
re-pins them into the source tree (followed by a rebuild).

---

## 4. Template Catalog — the "Golden Snapshot" Model

> **Revised 2026-06-17 (grill-with-docs) — opinions & v1 scope:**
>
> A golden template has **two layers**: (1) the **vanilla snapshot** from the
> ecosystem-native scaffolder (`cobra-cli init`, `uv init`, `dotnet new` — knows nothing
> of personal opinions), and (2) the **`.mkproj-overlay/`** that adds the *vetted
> opinions*: linters, formatters, test framework, recommended packages, CI. **The overlay
> is where the value lives** — the snapshot is inert scaffolding underneath it.
>
> **Source of truth (decided):** the overlay's tool/package choices are governed by the
> author's language guideline files (`~/peter_code/ai_support/guidelines/{golang,python,csharp}.md`).
> Those `.md` files are **canonical**; a lightweight template test asserts the overlay
> installs the tools they mandate (e.g. Python → `ruff`, `mypy`, `pytest`+`pytest-cov`,
> `uv`, `src/` layout; Go → `go-cmp`, no assertion libs, table-driven tests; C# →
> NUnit + FluentAssertions + NSubstitute + ErrorOr + Central Package Management). Drift is
> caught by the test.
>
> **v1 catalog (decided):** **Go, Python, C# only** — the three languages with guideline
> files. Every shipped template traces to a written guideline. **TS, Rust, Bash are
> deferred** to a "write the guideline → then the template follows" backlog; the table
> below retains them as future scope, struck through.

Scope is a **language × project-type matrix**. Each golden template is a **pinned
snapshot of an ecosystem-native scaffolder's output**, not a live invocation at init.

| Stack key | Lang | Type | Native scaffolder captured |
|---|---|---|---|
| `go-cli-cobra` | Go | CLI | `cobra-cli init` + `add serve/config` |
| `go-api-chi` | Go | API | `golang-standards/project-layout` + chi/zap/viper/testify |
| `ts-cli-commander-tsup` | TS | CLI | `commander` + `tsup` |
| `nestjs-api` | TS | API | `nest new` |
| `react-vite-ts` | TS | Frontend | `npm create vite` (React + TS) |
| `angular-standard` | TS | Frontend | `ng new` |
| `rust-cli-clap` | Rust | CLI | `cargo new` + `clap`/`anyhow` |
| `rust-api-axum` | Rust | API | `cargo new` + axum/tokio/serde/sqlx |
| `python-cli-typer` | Python | CLI | `uv add typer` |
| `python-fastapi` | Python | API | `uv init` + fastapi/uvicorn |
| `csharp-cli` | C# | CLI | `dotnet new console` + System.CommandLine |
| `csharp-webapi` | C# | API | `dotnet new webapi` |
| `bash-utility` | Bash | Util | shellcheck/shfmt/bats layout |

```mermaid
flowchart LR
    subgraph U["mkproj update — maintainer, occasional, ONLINE, needs toolchains"]
        A["Run native scaffolder in clean dir (cobra-cli / nest / cargo / dotnet ...)"] --> Bp["Strip build artifacts (node_modules, target/, bin/obj)"]
        Bp --> Cp["Apply security overlay (linters, audit tools, CI)"]
        Cp --> Dp["Pin snapshot + tool versions into embed.FS + sources.yaml"]
    end
    subgraph I["mkproj init — every project, OFFLINE, needs nothing"]
        E["Unpack vendored snapshot"] --> F["text/template substitutes module path, project name, author"]
    end
    Dp -.->|"committed, rebuilt binary"| E
```

Native scaffolders (`cobra-cli`, `nest`, `cargo`, .NET SDK, …) are required only on the
**maintainer** path, never at init.

### Enforcement layer — the overlay wires opinions, doesn't just list them

Decided 2026-06-17 (grill-with-docs). The overlay makes the guideline opinions
**enforced**, not merely present:

- **One gate definition (shared task runner).** `mise` tasks (`mise run lint`, `test`,
  `fmt`, `ci`) are the single source of truth for "the gates." Per-language commands live
  here (Python `ruff`+`mypy`+`pytest`; Go `gofmt`+`golangci-lint`+`go test`; C# `dotnet
  format`+`dotnet test`).
- **Local enforcement (lefthook).** A committed `lefthook.yml` runs **fast checks
  (lint+format) on `pre-commit`** and **full tests on `pre-push`** — honoring "commit
  early and often" *and* "tests pass before merge." lefthook chosen for language-agnostic,
  single-config composition that coexists with beads' git hooks. lefthook calls the `mise`
  tasks (no duplicated logic).
- **CI enforcement (GitHub Actions).** `.github/workflows/ci.yml` calls the **same** `mise
  run ci` target. Local hooks and CI cannot drift — one definition, two callers.

This satisfies "day one it just works": a fresh repo enforces the author's standards
locally and in CI before a line of feature code is written.

### Layer composition — agentic × language hooks converge on one pipeline

Decided 2026-06-17 (grill-with-docs). The agentic layer and the language overlay must not
fight over `.git/hooks` or duplicate commit-time work:

- **`.git/hooks` ownership — lefthook chaining.** Phase order: `bd init` (installs beads
  hooks) → `lefthook install` in **non-clobber/chain mode** (calls beads' pre-existing
  hooks). No coupling to beads' hook internals; no beads non-hook-mode dependency.
  *Verification item:* confirm beads' specific hooks survive `lefthook install` during
  implementation (lefthook's call-existing-hooks mechanism varies by hook type).
- **`mise.toml` is the single bootstrap.** `[tools]` pins the toolchain (go/python/dotnet,
  lefthook itself); `[tasks]` defines the gates. `mise install` provisions everything;
  lefthook and CI both call `mise run`.
- **One commit-time pipeline.** `lefthook pre-commit` = **secret-scan + lint + format**;
  `lefthook pre-push` = full tests. The **secret-scan is one shared script** called by
  *both* the agent guard hook (PreToolUse, earliest tripwire — agent layer) and the
  lefthook pre-commit (git layer — also catches human commits). Defense in depth,
  mirroring the deny-floor's Layer A + Layer B (ADR-0002).

`.gitignore` per language is sourced from the canonical **`github/gitignore`** repo,
merged with this repo's existing multi-language base `.gitignore`.

---

## 5. Template Variables

Collected via interactive prompts (flags may pre-answer). Everything else is verbatim.

- **Project name** — directory, bd issue prefix, module/repo path, `AGENTS.md` header.
- **Primary language/stack** — selects golden snapshot, `.gitignore` section, starter
  skill set, and test command.
- **Author identity** — name + email for `git config` and the `Co-Authored-By` footer
  (defaults from global config).
- **Git remote** — `none` | explicit URL | **`gh`** (create remote via `gh repo create`).

---

## 6. Allow/Deny Ruleset for Auto Mode

> **Revised 2026-06-17 (grill-with-docs).** The original section conflated two
> distinct concepts. They are now separated:
>
> - **Allowlist** — convenience, *always growing*, author-vetted command prefixes that
>   remove the confirmation prompt. Lives **per-project** inside a versioned managed
>   block; the canonical definition is **embedded in the `mkproj` binary**; a
>   reconciler (`mkproj sync-allowlist`) rewrites the block, triggered **notify-only**
>   on SessionStart (never auto-mutates).
> - **Deny floor** — safety, *stable*, blocks the irreversible. Enforced by the guard
>   hook as a **deny-only net** (it never approves anything).
>
> Both are defined in **one canonical embedded source file** (separate sections), so
> there is a single place to look, but they refresh on independent cadences.
>
> **Glob semantics (decided):** allow globs do the *bulk* of approval; the guard is a
> narrow net for *dangerous exceptions only*. Default per-tool rule is **broad prefix +
> targeted deny**: one `Bash(tool*)` allow covers every subcommand at any depth (Claude
> Code `*` is greedy across spaces — no per-subcommand jam); if a single subcommand is
> dangerous, add *that* to the deny floor rather than narrowing the allow.
>
> **Compound commands (decided):** the guard is **deny-only** — it splits a compound and
> blocks if any constituent is denied, but it does **not** approve compounds. A compound
> auto-runs only when an allow glob matches the whole line; otherwise it prompts, and
> that residual friction is accepted. **This supersedes rule A4's allow-side below.**
>
> **Codex parity (decided + verified):** Codex *does* support `PreToolUse` with the same
> `exit 2` / `permissionDecision: deny` contract (covers Bash, apply_patch, MCP). The
> deny floor is therefore **one shared guard script wired by both** Claude and Codex.
> Known risk: Codex docs note "some shell interception remains incomplete."

**Invariant — the guard hook is the terminal authority.** On every Bash tool call:

```mermaid
flowchart TD
    A["Agent emits Bash command"] --> B["settings.local.json permissions (allow/deny globs)"]
    B -->|"allowed / auto"| C["GUARD HOOK (PreToolUse, exit 2 = block) — self-contained, vendored"]
    B -->|"denied"| C
    C -->|"exit 0"| D["Command runs"]
    C -->|"exit 2"| E["Blocked — reason on stderr"]
    style C fill:#7a1f1f,color:#fff
```

Auto mode bypasses the **confirmation prompt** but **never** the hook: Claude Code runs
PreToolUse hooks in every permission mode, and a hook `exit 2` blocks the call. The
guard is genuinely the floor.

### Layer A — `settings.local.json` permissions (declarative, coarse)

```json
{
  "permissions": {
    "allow": [
      "Bash(bd*)", "Bash(instill*)",
      "Bash(git add*)", "Bash(git commit*)", "Bash(git push*)",
      "Bash(git pull*)", "Bash(git checkout*)", "Bash(git branch*)",
      "Bash(git status*)", "Bash(git diff*)", "Bash(git log*)"
    ],
    "ask": [],
    "deny": [
      "Bash(rm -rf*)", "Bash(git rm -rf*)",
      "Bash(*--force*)", "Bash(git push --force*)",
      "Bash(*DROP DATABASE*)", "Bash(*dropdb*)"
    ]
  }
}
```

### Layer B — guard hook rule table (authoritative, constituent-aware)

The hook reads the JSON payload, splits compound commands on `&&`, `||`, `;`, and pipes,
judges **each** constituent, and **allows only if every constituent is allowed** (a
single denied constituent blocks the whole line). Blocks with a clear stderr reason.

| # | Rule | Decision | Notes |
|---|---|---|---|
| A1 | `bd …`, `instill …` | allow | All BSRC/beads commands by default |
| A2 | `git add/commit/push/pull/checkout/branch/status/diff/log/fetch/merge/rebase/stash` | allow | Except deny carve-outs below |
| A3 | File reads/writes (`cat`, `ls`, `mkdir`, `touch`, `cp`, `mv`, single-file `rm -f`) | allow | Standard file ops |
| A4 | Compound `a && b && c` | allow **iff** every constituent allowed | One denied constituent blocks the line |
| D1 | `rm -rf` / `rm -r` recursive force delete (any flag order) | **block** | Destructive recursive delete |
| D2 | `git rm -rf` / mass cached removal | **block** | |
| D3 | `git push --force`/`-f` to a **protected branch** | **block** | `--force-with-lease` to non-protected branch allowed |
| D4 | Any history-rewriting push targeting a protected branch | **block** | Protected: `main master develop release/* production` |
| D5 | `DROP DATABASE`/`DROP SCHEMA`/`dropdb`/`TRUNCATE` (no WHERE) | **block** | Dropping databases |
| D6 | `mkfs*`, `dd of=/dev/*`, `> /dev/sd*`, fork bombs, `chmod -R 777 /` | **block** | Catch-all irreversible device/fs ops |
| D7 | `git reset --hard` / `git clean -fdx` / `git checkout .` discarding uncommitted work | **block** | Irreversible local data loss |
| D8 | `git commit --no-verify`/`-n`, `--no-gpg-sign` | **block** | Never bypass commit hooks (carries forward the one useful `calm-git-guard` behavior, now self-contained) |
| D9 | Display/search of secret-bearing paths: `cat`/`grep`/`rg`/`head`/`tail`/`less`/`awk`/`xxd` targeting `.env*`, `*.pem`, `*.key`, `credentials`, `.aws/credentials`, `*.tfstate`, `id_rsa` | **block** | **Secret-exposure guard.** Prevents agents/junior devs surfacing secrets into the transcript. Path list configurable at top of guard. |
| D10 | Unfiltered environment dumps: bare `env`, `printenv`, `set` (no filtering pipe) | **block** | Sprays every environment secret into the transcript |

> **A4 is superseded** (see Section 6 revision note): the guard never approves
> compounds; it only blocks a compound when a constituent matches D1–D10. Auto-run of a
> compound depends solely on an allow glob matching the whole line.

**Default posture:** commands matching neither an explicit allow nor deny rule **run**
(deny-list is the safety net, not an allowlist jail) — matching the constraint "allow by
default, block the irreversible."

**Self-containment:** the hook ships inside the project (`.claude/hooks/guard`) and is
wired with a **project-relative** command — never an absolute machine path. The
protected-branch list is a configurable array at the top of the hook.

---

## 7. File Manifest — the `mkproj` source repo

Roles: **[embed]** compiled in · **[render]** templated at init · **[verbatim]** copied
as-is · **[link]** symlinked · **[delegate]** produced by `bd`/`instill`/native tooling.

```
mkproj/
├── cmd/mkproj/main.go              # CLI entrypoint: `init` (default) + `update`
├── internal/
│   ├── scaffold/                   # Phase 1: render, copy, symlink, settings, guard
│   ├── delegate/                   # Phase 2: shell out to bd + instill
│   ├── remote/                     # Phase 3: gh repo create / git remote add
│   ├── prompt/                     # interactive picker (language -> type -> stack)
│   └── catalog/                    # stack-key matrix + resolution
├── sources.yaml                    # [maintainer] pinned upstreams (repo, path, ref/SHA)
├── templates/                      # ── all embed.FS ──
│   ├── common/
│   │   ├── gitignore.base          # [verbatim] multi-language base .gitignore
│   │   ├── AGENTS.md.tmpl          # [render]  {{.ProjectName}} header
│   │   ├── claude/
│   │   │   ├── settings.json        # [verbatim] hooks: bd prime, instill check-skills
│   │   │   ├── settings.local.json.tmpl  # [render] allow/deny permissions (Layer A)
│   │   │   └── hooks/guard           # [verbatim] self-contained guard hook (Layer B)
│   │   ├── codex/hooks.json          # [verbatim] codex SessionStart: bd prime
│   │   └── skill-manifest.json.tmpl # [render]  minimal starter skill set per stack
│   ├── gitignore/                  # [embed] vendored github/gitignore per lang
│   └── golden/                     # [embed] pinned native-scaffolder snapshots
│       ├── go-cli-cobra/  go-api-chi/  ts-cli-commander-tsup/  nestjs-api/
│       ├── react-vite-ts/  angular-standard/  rust-cli-clap/  rust-api-axum/
│       ├── python-cli-typer/  python-fastapi/  csharp-cli/  csharp-webapi/  bash-utility/
│       └── <each>/.mkproj-overlay/  # security overlay: linters, audit, CI
└── docs/superpowers/specs/         # this design doc
```

### Output in a scaffolded project (what the verification test inspects)

```
myproject/
├── .git/                    [delegate] git init + identity
├── .gitignore               base + vendored lang gitignore, merged
├── AGENTS.md                rendered, + bd's injected beads block
├── CLAUDE.md -> AGENTS.md    [link] exact symlink pattern preserved
├── .claude/{settings.json, settings.local.json, hooks/guard, skill-manifest.json, skills/->}
├── .codex/hooks.json
├── .beads/                  [delegate] bd init
└── <golden template files>  rendered: module path, project name, author substituted
```

---

## 8. Agentic Baseline — every repo gets these day one

Decided 2026-06-17 (grill-with-docs). Beyond permissions, every scaffolded repo ships:

- **ADR scaffold** — `docs/adr/` + MADR template (global CLAUDE.md mandates MADR; repos must have a home for it from commit 1).
- **CONTEXT.md glossary stub** — empty domain-glossary home for grill-with-docs / ubiquitous-language skills to write into.
- **Codex full parity** — Codex gets the shared guard (PreToolUse), allowlist equivalent, and skill access — not just `bd prime`.
- **Commit/PR conventions block** — Co-Authored-By footer, conventional-comments, <300-line PR norm wired into `AGENTS.md` per-repo (managed block).
- **instill artifacts gitignored** — `.claude/skills/` symlinks gitignored (machine-local); `skill-manifest.json` committed (portable lockfile). `instill check-skills` regenerates symlinks on clone.

## 9. Open Items for Implementation Planning

- Minimal starter **skill set per stack** (which `instill` skills are defaults).
- Security overlay contents per ecosystem (linters, audit tools, CI workflow files).
- Exact `text/template` variable schema and prompt sequence.
- `mkproj update` snapshot-capture automation (which toolchains, version pinning).
- Canonical embedded allowlist/deny-floor **seed contents** (bd, instill, git read/write,
  grep/rg/find, mise, gw, slack-cli, rtk — broad prefixes; D1–D10 deny floor).
- Reconciler **version-staleness detection** + SessionStart notify-only wiring for both agents.

---

*Authored By Peter O'Connor with Assistance from Claude Code (databricks-claude-opus-4-8) · 2026-06-16 · mkproj scaffolding system design*
