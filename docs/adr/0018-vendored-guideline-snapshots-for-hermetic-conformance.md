# Vendored guideline snapshots for hermetic conformance

**Status:** accepted · 2026-08-30

## Context

`test/guideline_conformance_test.go` (the `ebp` conformance test, ADR-0010) resolved the six
canonical guideline files at `$HOME/peter_code/ai_support/guidelines/{golang,python,csharp,
typescript,rust,bash}.md` — the maintainer's machine-local writing location. That path only exists
on the author's laptop. Once the infra-v5 remediation added a `unit-tests` CI job that runs
`go test ./...` on GitHub-hosted runners, every conformance test failed there: the runner has no
`$HOME/peter_code` directory, so `os.Stat`/`os.ReadFile` against the canonical path errors before a
single MUST/SHOULD check runs. The `unit-tests` job was red on `main` and on PR #51 as a result —
not because a stack drifted from its guideline, but because the fixture the test depends on is not
part of the checkout.

The generator's own invariant is that `go test ./...` must be runnable from a bare clone with no
external state (the same property `forge init` promises generated repos). A conformance test that
only passes on one physical machine violates that invariant for the `forge` repo itself.

## Decision

Vendor byte-for-byte copies of the six canonical guideline files into
`test/testdata/guidelines/` and repoint the conformance checker's `guidelinePath` at the vendored
copies exclusively. The conformance tests (`TestCanonicalGuidelineFilesReachStablePaths`,
`TestShippedV1StacksSatisfyGuidelineFloor`, `TestConformanceIgnoresVettedExtrasOutsideTheGuidelineFloor`,
`TestPostV1GuidelineFilesExistInVendoredSnapshots`, `TestShippedTemplateWithoutGuidelineBackedLanguageFails`)
now read only files inside the repository checkout — identical behavior locally and in CI, and no
dependency on `$HOME` layout.

A companion drift test, `TestVendoredGuidelinesMatchCanonicalSource`, byte-compares each vendored
snapshot against the canonical source at `~/peter_code/ai_support/guidelines/` (or
`FORGE_CANONICAL_GUIDELINES_DIR`, when set, for a different maintainer layout). It **skips** when
the canonical directory is absent — the expected case on every CI runner and any contributor
machine without the author's guideline repository — and **fails**, naming the `cp -f` command that
refreshes the snapshot, when the canonical directory is present and a vendored file differs. This
keeps the guideline floor enforced everywhere while making staleness detectable only where the
source of truth actually lives.

## Considered Options

- **Skip the conformance tests in CI.** Gate the `ebp` suite behind a canonical-path existence
  check and skip when absent, everywhere. Rejected: this silently removes the guideline floor from
  the CI gate — the exact failure class the `unit-tests` job was added to catch. A stack could drop
  a MUST-mandated tool and CI would report green.
- **Environment-variable injection only (no vendoring).** Require CI to set
  `FORGE_CANONICAL_GUIDELINES_DIR` (or equivalent) pointing at a guideline checkout fetched as a CI
  step. Rejected: still non-hermetic by default — a contributor running `go test ./...` locally
  without the env var set reproduces the original failure, and CI now depends on an extra fetch
  step that can itself drift or fail. Vendoring keeps `go test ./...` self-sufficient with zero
  configuration.

## Consequences

- Guideline publishing (`zz8`) gains a follow-up step: after editing a canonical guideline file,
  run `cp -f ~/peter_code/ai_support/guidelines/<file>.md test/testdata/guidelines/<file>.md` and
  commit the refreshed snapshot. `TestVendoredGuidelinesMatchCanonicalSource` catches a forgotten
  refresh on the maintainer's machine, where it can actually resolve the canonical path.
- `docs/SPEC.md` §10.2 and the Conformance fixture row (§18 Seam Inventory) are updated to describe
  the vendored-snapshot precondition instead of "resolves at canonical path."
  `CONTEXT.md`'s Guideline file entry is updated the same way.
- The guideline→file map in `guidelinePath` now covers all six languages (`golang`, `python`,
  `csharp`, `typescript`, `rust`, `bash`), matching the six vendored snapshots, even though
  `languageRequirementSpecs` (the MUST/SHOULD extraction rules used by
  `TestShippedV1StacksSatisfyGuidelineFloor`) still only covers the four v1 shipped languages. A
  language absent from both maps (e.g. `zig`) is what
  `TestShippedTemplateWithoutGuidelineBackedLanguageFails` now exercises, since `rust` and `bash`
  are addressable but not (yet) shipped.
