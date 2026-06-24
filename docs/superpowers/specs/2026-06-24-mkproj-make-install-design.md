# mkproj Make System — Design

- **Beads issue:** agentic_template_start-0ic
- **Date:** 2026-06-24
- **Status:** Approved

## Problem

`mkproj` has no Makefile. Building and installing the CLI is a manual
`go build` plus a hand copy. A contributor running `make install` SHOULD get a
working `mkproj` on their PATH in one step.

## Goal

`make install` MUST build the `mkproj` binary for the host platform and place it
in `$HOME/.local/bin`, which is already on the user's PATH.

## Scope

### In scope

- A `Makefile` at the repository root with build / test / install / uninstall /
  clean / help targets.
- Placing the binary in `$HOME/.local/bin` (overridable via `BINDIR`).
- Ensuring the local build output directory `bin/` is gitignored.

### Out of scope (explicitly deferred)

- **Shell completions.** The CLI uses Go's stdlib `flag` package, which emits no
  completions. Deciding the completion source (a `completion` subcommand, static
  files, or a cobra migration) is deferred. No completion targets are added.
- **Version/ldflags embedding.** There are no git tags yet; no version injection.
- **cobra migration.** The command surface stays on stdlib `flag`.
- **Cross-compilation.** Host platform only (YAGNI).

## Targets

| Target | Behavior |
|--------|----------|
| `help` (default) | Self-documenting list of targets. Runs when `make` is called with no arguments. |
| `build` | `mkdir -p bin $(CURDIR)/.cache/tokf`, then exports `GOCACHE=$(CURDIR)/.cache/go-build`, `TOKF_HOME=$(CURDIR)/.cache/tokf`, and `TOKF_DB_PATH=$(CURDIR)/.cache/tokf/tracking.db` before running `go build -o bin/mkproj ./cmd/mkproj`. Produces the host binary in `bin/` while keeping cache and tokf writes inside the repo under `make -C`. |
| `test` | `GOCACHE=$(CURDIR)/.cache/go-build go test ./... -count=1`. Matches the project's documented verification command while keeping cache writes inside the target repo under `make -C`. |
| `install` | Depends on `build`. Creates `$(BINDIR)` if absent, then `install -m 0755 bin/mkproj $(BINDIR)/mkproj`. |
| `uninstall` | Removes `$(BINDIR)/mkproj`. |
| `clean` | Removes `bin/`. |

## Key decisions

- **`BINDIR ?= $(HOME)/.local/bin`** — fixed default per the user's choice, but
  `?=` leaves an override escape hatch (`make install BINDIR=/somewhere`) at no
  added complexity.
- **`install(1)` over `cp`** — sets the `0755` mode atomically; standard for
  installing executables.
- **`install` depends on `build`** — a single `make install` always builds fresh.
- **Repo-local build caches** — the build target exports `GOCACHE`, `TOKF_HOME`,
  and `TOKF_DB_PATH` into `$(CURDIR)/.cache/` so restricted environments do not
  attempt writes under `$HOME` during `make -C`.
- **All targets `.PHONY`** — none of the target names correspond to on-disk files
  (build output lives under `bin/`).
- **`bin/` added to `.gitignore`** — it is not currently ignored.

## Data flow

```mermaid
flowchart LR
    A[make install] --> B[make build]
    B --> C["mkdir -p bin and .cache/tokf"]
    C --> D["export GOCACHE TOKF_HOME TOKF_DB_PATH under $(CURDIR)/.cache/"]
    D --> E["go build -o bin/mkproj ./cmd/mkproj"]
    A --> F["mkdir -p $BINDIR"]
    F --> G["install -m 0755 bin/mkproj $BINDIR/mkproj"]
```

## Verification

1. `make build` → `bin/mkproj` exists and is executable.
2. `make -n -C <repo> build` (or the matching Go contract test) shows
   `GOCACHE`, `TOKF_HOME`, and `TOKF_DB_PATH` resolved inside `<repo>/.cache/`.
3. `make install` → `$HOME/.local/bin/mkproj` exists; `which mkproj` resolves
   there; `mkproj` runs.
4. `make test` → full Go suite passes.
5. `make clean` → `bin/` removed.
6. `make uninstall` → installed binary removed.
7. `make help` (or bare `make`) → lists targets.
8. `git status` → `bin/` not shown as untracked.

## Error handling

- Standard Make failure semantics: any failed recipe line aborts the target with
  a non-zero exit code.
- `mkdir -p` makes the install idempotent regardless of whether `$(BINDIR)`
  already exists.
