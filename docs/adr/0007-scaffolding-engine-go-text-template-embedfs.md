# Scaffolding engine: a standalone Go binary using text/template + embed.FS

**Status:** accepted · 2026-06-16 (decision originated in the system-design spec §2; recorded as an ADR 2026-06-20 to close the decision-log gap)

## Context

`mkproj` must turn an empty directory into a fully configured project offline, with no
clone step and no machine-path dependency. The engine choice — what does the templating,
where the assets live, and how the tool is distributed — is the most foundational
architectural decision in the system, yet it lived only as a decision table in the
2026-06-16 system-design spec and had no ADR. This record backfills it.

## Decision

`mkproj` is a **standalone Go binary** that renders with the standard library's
**`text/template`** and embeds every template and asset via **`embed.FS`**. It is
distributed as an **installed binary** (`~/.local/bin/mkproj`) with all assets compiled in,
so `mkproj init` is fully **offline and reproducible**. Assets are **vendored + refreshable**:
init never reaches the network, and the maintainer-only `mkproj update` path re-fetches and
re-pins upstreams (see ADR-0005, ADR-0006).

## Considered Options

- **A from-scratch templating engine.** Rejected: reinvents work `bd`/`instill` already do
  and adds a bespoke syntax to maintain.
- **A pure-shell `init.sh`.** Rejected: weaker templating and validation than a typed binary;
  hard to test and to embed assets cleanly.
- **Cookiecutter/Copier as the engine.** Rejected: adds a Python runtime dependency and a
  second templating dialect. Kept only as acknowledged prior art the design deliberately does
  not depend on.
- **A richer third-party Go template engine (inheritance/partials).** Rejected for v1:
  `text/template` with `Option("missingkey=error")` is sufficient for the verbatim-plus-substitute
  model, and the stdlib carries zero dependency and lock-in risk.

## Consequences

- Init is hermetic: no clone, no network, byte-reproducible output — which is what makes the
  offline smoke tier (ADR-0005, init-lifecycle §7) and the `update` idempotence contract
  (ADR-0006) testable in the first place.
- `text/template` is the render contract: a missing variable is `missingkey=error` (hard fail,
  never `<no value>`); the `.tmpl` suffix marks renderable files; everything else is copied
  verbatim (CLI/render contract §4).
- Refreshing vendored assets requires the `mkproj` binary present on the maintainer machine
  (the same constraint ADR-0001 places on the allowlist reconciler).
- `mkproj` is a sibling to the author's existing `gw`/`instill` Go tools — one toolchain,
  one install pattern.
