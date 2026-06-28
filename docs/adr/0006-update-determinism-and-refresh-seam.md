# mkproj update determinism is enforced by an idempotence test; the refresh seam fails loud

**Status:** accepted · 2026-06-20

## Context

`mkproj update` (maintainer-only, online) regenerates a stack's vanilla layer from its
recipe (ADR-0005) and re-pins `sources.yaml`. `cjl`'s requirement is that refresh output be
"deterministic enough to rebuild the binary from committed sources" — concretely, a re-run
with no upstream change must produce a **byte-identical** `templates/golden/<key>/` tree, so
a maintainer can trust `git diff` to mean "upstream actually changed" rather than tool
noise. Native scaffolders inject non-determinism: generated timestamps, absolute temp-dir
paths, GUIDs (`dotnet`), tool-version strings, line-ending/ordering variance.

A second problem lives at the **seam** where overlay files layer onto vanilla dirs
(ADR-0005 / packaging convention on `3tt`): an upstream refresh can **orphan** an overlay
file (its vanilla parent dir disappears) or **collide** with one (upstream starts emitting a
file at an overlay-occupied path). `update` must never write the overlay (the byte-identical
invariant from prompt-render-contract §8), so it cannot auto-resolve — but resolving these
*silently* would defeat the trust-the-diff goal.

## Decision

**1. Determinism is contracted by an idempotence test, not an exhaustive up-front rule
list.** `update` runs a **normalization pass over the vanilla layer only** (the overlay is
hand-authored and already deterministic) before writing the snapshot. Normalization rules
are **per-stack pattern sets declared in a `normalize:` block in the `sources.yaml` row**,
because the variance is tool-specific — there is no universal stripper. The canonical rule
classes are: neutralize generated timestamps/dates to a sentinel; replace the temp build
dir's absolute path with the canonical placeholder; neutralize generated GUIDs/random IDs to
fixed sentinels; force LF + canonical file ordering + trailing-newline; pin tool-version
strings to the `resolved:` version rather than the maintainer's locally-installed version.
New variance is found empirically — run update twice, diff, add a rule until clean. The
acceptance gate is:

```gherkin
Scenario: update is idempotent when upstream is unchanged
  When the maintainer runs `mkproj update --stack <key>` twice with no upstream change
  Then templates/golden/<key>/ is byte-identical to the committed tree
   And .mkproj-overlay/ is untouched (byte-identical)
   And git diff is empty
```

**2. `captured:` means last-*changed*, not last-*run*.** A no-op update does not rewrite the
date, so a `git diff` after an unchanged-upstream refresh is truly empty.

**3. The refresh seam fails loud, never auto-mutates the overlay.** After regenerating
vanilla, `update` checks committed overlay paths against the new vanilla tree:
- **Orphan** (overlay file's vanilla parent dir no longer exists) → **hard fail**: exit
  non-zero, name each orphaned path, write nothing. The composed skeleton would break, so the
  maintainer must fix the overlay or the recipe and re-run.
- **Collision** (new vanilla file at an overlay-occupied path) → **loud warn**: complete the
  vanilla write, list each collision. Init's overlay-wins composition still yields a correct
  result, but the maintainer should review whether the upstream file makes the overlay
  version redundant.

## Considered Options

- **Fixed universal normalization spec written up front.** Rejected: variance is
  tool-specific and unknowable ahead of running each scaffolder; an up-front spec would be
  both over- and under-inclusive. The idempotence test makes "done" objective and lets rules
  grow by discovery — matching the handoff's "Gap B as implementation-discovery" stance.
- **Rewrite `captured:` on every run (last-run semantics).** Rejected: breaks byte-identity
  of `sources.yaml`, so a no-op update always shows a diff — the exact noise this ADR exists
  to remove.
- **Seam: both orphan and collision hard-fail.** Rejected: a collision still composes
  correctly (overlay wins); blocking every upstream file addition would make routine refreshes
  needlessly painful.
- **Seam: both warn (never block).** Rejected: an orphan produces a broken composed skeleton;
  shipping that on a warning violates the walking-skeleton guarantee.

## Consequences

- `update` needs the normalization pass plus a seam-check that walks the committed
  `.mkproj-overlay/` against the regenerated vanilla tree — no new manifest, since the overlay
  dir is already the source of truth for expected layout.
- The idempotence test is the gate `cjl` must pass to close; `3tt`'s vendored snapshots must
  already be normalized so the first committed state is the fixed point.
- Determinism rules are scoped to vanilla, so hand-authored overlay churn (the value layer)
  is never at the mercy of a stripper.
