# Principal Platform Review — `mkproj` junior-engineer implementation

**Date:** 2026-06-28
**Reviewers:** Principal Platform Engineer (P1) + independent Principal Platform Engineer peer review (P2)
**Scope:** `cmd/`, `internal/`, `templates/`, `sources.yaml`, the beads work graph, and the
intent/decision docs (SPEC / PRD / ADR / CONTEXT).
**Lens:** DevX · platform repeatability · reliability.
**Authority order (SPEC §0.2):** ADRs > SPEC > everything else.

Findings are tracked in beads. Confirmed findings have actionable IDs; the four genuine
principal-vs-principal disagreements are filed as `[NEEDS-DECISION]` issues assigned to the
maintainer to adjudicate.

---

## Headline

The implementation is substantially complete and, on the points a naive scan would flag, mostly
**correct against the spec** (the no-rollback init, the marker-based managed blocks, and the
non-deterministic walk order are all explicitly sanctioned — do not "fix" them). The real problems
are not in the happy path; they are in the **verification gates that prove the invariants**. Three
spec-named seam gates are vacuous, absent, or non-portable, and that is the structural reason work
was repeatedly "closed before verified."

The single most important defect: the conformance test — the named owner of the ADR-0010 guideline
floor — only runs on the maintainer's laptop.

---

## What the engineer got right (do not regress these)

| Item | Why it's correct |
|---|---|
| Init fails fast with **no rollback** (`rm -rf && retry`) | ADR-0004 / SPEC §4.3; safe via empty-dir precondition (ADR-0008) + Phase-3-last (ADR-0009). `failWithRecovery` is spec-accurate. |
| `fs.WalkDir` walk order is non-deterministic | Output is reassembled deterministically elsewhere; determinism is only required for the gitignore merge (§8.1) and `update` idempotence (§15). |
| Managed-block edits via string markers | Sanctioned by SPEC §8.2 / ADR-0001; `Sync` rewrites only between markers and preserves surrounding bytes. |
| `embed.FS` present (`assets.go:9`) | Remediates the 2026-06-22 handoff finding; init is offline/hermetic per ADR-0007. |
| Generated repos work without `mkproj` | Hooks no-op via `command -v mkproj … || true`. |
| Deny-only guard is **behaviorally** tested | `TestSharedGuardBlocksDenyFloorAndAllowsSafeCompound` drives the real guard for D1–D16, asserts exit 2 + `BLOCKED [Dn]`, and proves a safe compound passes; the guard only ever returns 0/2 (never approves). ADR-0002/§11.2 holds. |
| Refresh seam orphan/collision | ADR-0006 §3 implemented correctly: orphans hard-fail, collisions warn. |

> Confirmation that the guard is live: this very review had a `python -c` JSONL check blocked with
> `BLOCKED [D16]`. The deny floor does what it claims.

---

## Confirmed findings (both principals agree)

| ID | Finding | Tag | Sev |
|---|---|---|---|
| `gph` | **M1** — conformance test hardcodes `~/peter_code/ai_support/guidelines/…`; `t.Fatalf` (no skip) off-maintainer | Repeatability | **HIGH** |
| `gat` | **F9** — "closed before verified": enforce seam-test-green as a closure precondition | Process / Repeatability | High |
| `yrp` | **F1** — `update` re-pin write path is hand-rolled YAML string surgery; no format-variance tests (idempotence risk) | Repeatability | Med |
| `nna` | **F4** — SPEC §16.1 MUST "assert no network access occurred" is unguarded (stubs ≠ assertion) | Reliability | Med |
| `mka` | **F5** — "works without mkproj" only static-string-checked, never executed | DevX | Med |
| `pth` | **F8** — `delegate.go:26` leaks full PATH in errors | DevX | Low |
| `ivl` | **F11** — no format validation of `--github-user` / `--remote-url` / `--module-path` | Reliability | Low |
| `ssp` | **M3** — guard→`secret-scan.sh` exit-code propagation untested | Reliability | Low |

**F10** (the open `s81` + in-progress P1 `92i`) is **not** filed separately — it is already tracked;
it is folded into `gat`'s acceptance: *"v1 verified" is not claimed until both `s81` and `92i` close.*

Each issue carries its Given/When/Then as `acceptance_criteria` in beads.

### Why `gph` (M1) is the headline
`test/guideline_conformance_test.go:253` resolves the guideline floor from a personal home directory
and `t.Fatalf`s when absent (L18/L22). That directory exists only on the maintainer's box, so
`TestShippedV1StacksSatisfyGuidelineFloor` — the named owner of the SPEC §18 `ebp×zz8` seam and the
ADR-0010 floor invariant — **fails in CI and on every other machine.** A spec-named invariant gate
that can't run anywhere but one laptop is exactly how `wqh`/`iha` got marked done before they were
actually verified.

---

## Genuine disagreements — escalated for the maintainer to adjudicate

Filed as P1 `[NEEDS-DECISION]` beads issues assigned to the maintainer. P1 = first review,
P2 = peer review.

| ID | Topic | P1 position | P2 position |
|---|---|---|---|
| `dq2` | update abort consistency (F2) | Cross-file abort can leave inconsistent state; restore both. **Reliability.** | `sourcesPlan.rollback()` already restores (update.go:201-206); only a *failed* rollback is unsurfaced. **Low.** |
| `dq3` | sync-allowlist validation (F3) | Lacks duplicate/missing/malformed validation + tests. | missing+malformed already validated & tested; only **duplicate-marker** detection missing. **Narrow, Low.** |
| `dq6` | prompt hang (F6) | TTY-stuck hang is a Reliability risk; add timeout + `--no-interactive`. | non-TTY already fails loud; hang is interactive-only; `--no-interactive` redundant; real fix = cancellable `form.Run`. **DevX/Low.** |
| `dq7` | context propagation (F7) | Thread `ctx` into 3 FS helpers. | Bounded local FS, no blocking I/O, exec already threads `ctx` → cargo-cult; **drop F7.** |

The synthesizing principal leans toward P2 on all four (P2 verified the code paths), but per the
review mandate these are left for the maintainer to decide rather than resolved unilaterally.

---

## Systemic conclusion

`gph`, `nna`, and `mka` together mean three of the spec's named invariant gates are simultaneously
non-portable, absent, or vacuous. SPEC §18 says an unguarded seam *is* a defect. Fixing the **gates**
(not just the per-finding code) is what makes "v1 verified" a truthful claim. Recommend prioritizing
`gph` → `gat` → (`nna`, `mka`, `yrp`) before any post-v1 expansion.

---

### Method / provenance
- Two independent principal passes; code claims verified by direct read of `assets.go`, `update.go`,
  `sync.go`, `walking_skeleton_test.go`, `security_hooks_test.go`, `delegate.go`, `init.go`,
  `guideline_conformance_test.go`, `project.go`, `prompt.go`, and the guard/secret-scan tests.
- Cross-checked against SPEC §§4,5,8,10,13,16,18 and ADR-0001..0010.
- beads state at review time: 60 issues (54 closed, 1 in-progress `92i`, 5 open); 12 issues added by
  this review.
- `bd` CLI was not installable in the review environment; issues were written directly to
  `.beads/issues.jsonl` (the committed source of truth). A maintainer should run a `bd` sync/import
  to hydrate the Dolt store.
