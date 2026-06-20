# The canonical guideline file is the conformance floor, not the ceiling

**Status:** accepted · 2026-06-20 (grill-with-docs)

## Context

ADR-0003 makes the language guideline files the canonical source of truth for what a stack's
overlay installs, enforced by the `ebp` conformance test. But "source of truth" is ambiguous in
one direction: when the overlay ships a vetted tool the guideline does not mention
(`FluentAssertions` for C#, `pytest-mock` for Python both surfaced this way), is that a
**violation** (the overlay drifted from the guideline) or **allowed** (a vetted extra above the
written minimum)? The answer determines whether the conformance test is a build-breaker for those
tools and whether publishing guideline edits (`zz8`) must expand to add them.

## Decision

The guideline is the **floor (minimum), not the ceiling (exact contract).** The overlay MUST
implement every guideline MUST/SHOULD; it MAY add vetted extras the guideline is silent on. The
`ebp` conformance test **fails only when a guideline MUST has no corresponding overlay tool** — it
does **not** fail when the overlay ships an extra absent from the guideline. Consequently
`FluentAssertions` and `pytest-mock`, as vetted extras, require no guideline edit.

## Considered Options

- **Exact-match (guideline is the ceiling).** Every overlay tool must be named in the guideline;
  extras fail the test. Rejected for v1: stricter and more auditable, but it forces every vetted
  convenience into the written standard before it can ship, and would expand `zz8` to add
  FluentAssertions/pytest-mock now — friction with no safety benefit, since an *extra* vetted tool
  is not a risk the way a *missing* mandated tool is.

## Consequences

- The conformance test's failure condition is one-directional: missing-MUST fails; extra-tool
  passes. This is reflected in `CONTEXT.md` ("minimum source of truth") and SPEC §10.2.
- `zz8` is publish-only — it republishes the `6ms` edits, not new tool mandates.
- If an overlay extra later proves it *should* be a standard, that is a normal guideline edit, not
  a conformance failure forcing the issue.
