# v1 stacks are sourced as walking-skeleton recipes, not single scaffolder runs

**Status:** accepted · 2026-06-20

## Context

The catalog model (system-design §4) framed each golden template as "a pinned snapshot of
an ecosystem-native scaffolder's output" — implying one tool invocation per stack
(`cobra-cli init`, `uv init`, `dotnet new`). Five of the six v1 stacks fit that shape, but
**`go-api-chi` does not**: it is described as `golang-standards/project-layout` (a git-repo
template, not a runnable scaffolder) plus a hand-assembled chi/zap/viper/testify dependency
set. There is no single command whose output is the snapshot.

This broke two downstream contracts:

1. The `mkproj update` model (prompt-render-contract §8) assumed "invoke the pinned native
   scaffolder in a temp dir" — a single tool.
2. `sources.yaml` had no defined row shape, so the engineer vendoring assets (`3tt`) could
   not author six consistent rows.

A second, related question was *what the shipped artifact guarantees*. The author's intent
(grill-with-docs 2026-06-20) is that each stack delivers a **walking skeleton** (Cockburn):
a thin vertical slice that actually runs end-to-end with a green test — not inert
scaffolding the engineer must wire up.

## Decision

**1. A stack's vanilla layer is the output of a recipe, which may be multi-step.** The
recipe is most often a single scaffolder run, but may be a pinned git checkout plus
dependency pins. `sources.yaml` uses one uniform row schema with a `kind` discriminator and
an ordered `steps` list; a single-scaffolder stack is simply a one-step recipe. A
`resolved:` sub-block, written only by `update`, records the captured ref/SHA/version and
the last-changed date. Each row also names the upstream `github/gitignore` file stem,
resolved against one repo-wide pinned SHA.

```yaml
go-api-chi:
  kind: recipe
  steps:
    - checkout: "github.com/golang-standards/project-layout"
      ref: "<pinned-SHA>"
      strip: [".git", "README.md", "docs"]
    - run: "go mod init {{.ModulePath}}"
    - run: "go get github.com/go-chi/chi/v5@v5.1.0 go.uber.org/zap@v1.27.0 ..."
  gitignore: Go
  normalize: [ ... ]              # per-stack, see ADR-0006
  resolved: { ref: "<SHA>", captured: "2026-06-20" }
```

**2. "Walking skeleton" is the acceptance guarantee on the composed (vanilla + overlay)
output, not a third layer.** After `mkproj init` the repo runs and has ≥1 real passing
test. The vanilla/overlay split is preserved beneath it: the vanilla layer is
recipe-produced and refreshable; the wiring that makes it *walk* (router, logger, the one
green handler test) lives in `.mkproj-overlay/` so `update` can refresh vanilla without
re-vetting the wiring.

**3. C# maps to `VisualStudio.gitignore`** — GitHub's canonical .NET ignore file — rather
than a hand-curated in-repo file, to stay refreshable from upstream.

## Considered Options

- **Keep "one native scaffolder per stack" and drop `go-api-chi`.** Rejected: a Go API
  stack is core v1 scope; the gap is in the sourcing model, not the stack's value.
- **Two explicit row schemas (scaffolder vs. recipe).** Rejected: forces the writer and the
  `update` interpreter to branch on shape. The one-step-recipe special case unifies them at
  no cost.
- **Push recipes out of YAML into per-stack Go functions.** Rejected: hides the pins in
  code, defeating the point of a reviewable, diffable `sources.yaml`; raises the bar for a
  guided engineer to add a stack.
- **Bake each stack as one hand-vetted skeleton blob that `update` replaces wholesale.**
  Rejected: every dependency bump would force a full re-vet by hand; the vanilla/overlay
  split exists precisely to make refresh cheap.
- **Hand-curate `Csharp.gitignore` in-repo.** Rejected: makes the author own currency of a
  file upstream already maintains, contradicting the vendored-and-refreshable model.

## Consequences

- `mkproj update` needs a small `steps` interpreter (`run` / `checkout` / `strip`). This is
  maintainer-path code, run occasionally, never at init — complexity stays off the hot path.
- `3tt` fills one row template six times and commits one skeleton tree per stack; the
  walking-skeleton test ships in each stack's overlay (satisfying the non-vacuous-test
  requirement in `3tt`'s notes and `uuw` smoke verification).
- The determinism and refresh-seam rules for the recipe model are specified separately in
  **ADR-0006**.
