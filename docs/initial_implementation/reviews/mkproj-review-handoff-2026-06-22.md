# Handoff: mkproj v1 — Post-Review Bug-Fix Sprint
**Date:** 2026-06-22  
**From:** Peter O'Connor + Claude Code (review session)  
**Next session focus:** Address the 7 critical blockers and important issues found in the compliance review of the junior engineer's `codex-mkproj-template-system` worktree. Save verified findings to `docs/initial_impelmentation/reviews/`.

---

## What Just Happened

A critical compliance review was completed against the junior engineer's implementation in `.worktrees/codex-mkproj-template-system`. Three parallel subagents audited:

1. **Spec / CONTEXT / beads compliance** — compared closed issue ACs against implementation
2. **Go implementation quality** — build, test, error handling, security, template correctness
3. **BDD test coverage** — scenario-by-scenario comparison against SPEC.md

**Verdict: PARTIALLY COMPLIANT — multiple shipping blockers present.**

The junior's claim ("complete, working, tested, and verified with BDD tests") is incorrect. Build passes and 25 unit tests pass, but the passing tests do not verify the product's specification.

---

## Key Artifacts

| Artifact | Path |
|---|---|
| Specification | `docs/SPEC.md` |
| PRD | `docs/PRD.md` |
| Context / glossary | `CONTEXT.md` |
| Junior's implementation (worktree) | `.worktrees/codex-mkproj-template-system/` |
| Junior's commits | `b910b00`, `10f0027` (atop `ca2c74f`) |
| Prior junior brief | `docs/handoffs/2026-06-20-junior-engineer-brief.md` |
| Beads issues | Run `bd list --status=open` and `bd list --status=closed` |

---

## Findings Summary

Full findings were reported in the review session. The next session should **first save the review report** to `docs/initial_impelmentation/reviews/compliance-review-2026-06-22.md` (preserving the misspelling in the path argument as given). The report text is in the review session output. Key points:

### 7 Critical Blockers (must fix before v1 can ship)

1. **`embed.FS` absent** — `cmd/mkproj/main.go` uses `os.DirFS(".")` at runtime; no `//go:embed` directive exists anywhere. SPEC §3 / ADR-0007 require a self-contained offline binary. Issue `iha` was closed but the embed is missing.

2. **Walking-skeleton integration test absent** — SPEC §1 names this the canonical product acceptance test. `mkproj init → mise install → mise run ci` is not tested anywhere.

3. **Golden stacks missing gate-pipeline files** — none of the 6 stacks contain `mise.toml`, `lefthook.yml`, or `.github/workflows/ci.yml`. Issues `x2k`, `02o`, `ud1` are unclosed.

4. **`sync-allowlist` is a data-loss bug** — `defaultAllowlistBlock()` in `cmd/mkproj/main.go` returns 5 entries; the template managed block has 81. Running `mkproj sync-allowlist` silently destroys 76 valid permissions.

5. **Template `secret-scan.sh` broken** — `templates/common/claude/hooks/secret-scan.sh` is missing `scan_staged()`. Also has an extra `}` in `normalize_token()` line ~50 breaking D9 token detection. Security gap in every generated repo.

6. **Python templates use wrong variable** — both Python `.tmpl` files use `{{.ProjectName}}` for the PEP 508 `name` field. Correct variable is `{{.PythonPackage}}` (computed in `internal/project/project.go`, never used in templates).

7. **C# templates hardcode `"Project"`** — `{{.CSharpNamespace}}` computed but never injected. All C# projects are named `Project` regardless of user input.

### Important Issues (fix before merging to main)

8. `skill-manifest.json.tmpl` includes Go-specific skills unconditionally for all languages
9. `CONTEXT.md` stub and ADR scaffold missing from `templates/common/` (SPEC §14)
10. `sources.yaml` has placeholder SHAs (`1111111...`–`6666666...`) — `mkproj update` cannot work
11. `--github-user` not prompted interactively when `--remote gh` selected (SPEC §6)
12. `allowlist.Detect()` silently swallows `fmt.Sscanf` parse errors
13. `mustGetwd()` panics instead of returning error
14. `ensureEmptyDir()` in `scaffold/writer.go` is dead code (never called in production path)

### What Actually Passed

- Go build: clean
- 25 unit tests: all pass, race detector clean
- bats `test/secret-scan.bats`: 14 tests, all pass (D9/D10, bypass, negative cases)
- Catalog boundary (`01b`): PASS
- Phase 3 remote publish (`wer`): PASS
- Core overlay composition mechanics in `internal/scaffold/writer.go`: correct

### 15 SPEC Scenarios With No Test

Walking skeleton, gate-pipeline per stack, init refuses non-empty dir at `Initializer.Run` level, remote-created-but-push-fails invariant, guard hook mode 0755, gitignore idempotence, multiple managed blocks coexist, overlay conformance floor, stale allowlist notifies without blocking, missing mkproj binary does not break session start, forgot-to-bump backstop, TTY interactive prompter, invalid flag error message lists choices, `mkproj update` idempotence.

---

## Open Beads Issues (correctly scoped, do not close prematurely)

- `79r` — Shared secret-scan core (P1, blocked)
- `ebp` — Guideline conformance tests (P1, blocked by `zz8`)
- `rom` — Allowlist reconciler (P2; core reconciler exists, `--include-personal` missing)
- `wqh.3` + children `369`, `cjl` — Maintainer stack refresh (P2; `mkproj update` returns "not implemented")
- `zz8` — Publish guideline updates (P2)
- `7eu` — TypeScript/Rust/Bash guidelines (P3, post-v1)

---

## Closed Issues With Partial/Fail Status

These issues were closed by the junior but ACs are not fully met. Consider reopening:

- `iha` — embed.FS missing (critical; ACs explicitly required it)
- `aa9` — PASS for source repo, FAIL for template version of `secret-scan.sh`
- `3tt` — Placeholder SHAs; gate-pipeline files absent
- `yuw` — CONTEXT.md stub and ADR scaffold missing from `templates/common/`
- `aoc` — `defaultAllowlistBlock()` inconsistent with template managed block

---

## Next Session Work Order

1. **Create `docs/initial_impelmentation/reviews/` directory** if it does not exist
2. **Save the compliance review report** to `docs/initial_impelmentation/reviews/compliance-review-2026-06-22.md` — full text was the last major output of the prior session
3. **Work through the 7 critical blockers** in the `codex-mkproj-template-system` worktree, one at a time, using `bd create` to track each fix
4. **Re-run `go test ./...` and `bats test/secret-scan.bats`** after each fix
5. **Write the walking-skeleton integration test** last (requires gate-pipeline files to exist first)

---

## Environment Notes

- **Implementation worktree:** `.worktrees/codex-mkproj-template-system/` — branch `codex-mkproj-template-system`, HEAD `10f0027`
- **Main repo branch:** `main` (guard hook blocks `Write` tool calls on `main` — use a feature branch for code changes)
- **`bd` beads** is the task tracker — do NOT use `TaskCreate`/`TodoWrite`/markdown TODOs
- **`bd remember`** for persistent knowledge; never use `MEMORY.md`
- **Session close protocol:** `bd dolt push` then `git push` are mandatory before ending any session

---

## Suggested Skills

Invoke these at the start of the next session as appropriate:

| Skill | When |
|---|---|
| `superpowers:test-driven-development` | Before writing any fix |
| `superpowers:verification-before-completion` | Before closing any beads issue or claiming a fix is done |
| `superpowers:systematic-debugging` | If any fix produces unexpected test failures |
| `superpowers:writing-plans` | Before tackling the walking-skeleton test (multi-step) |
| `golang:golang-cli` | When fixing `embed.FS` wiring in `cmd/mkproj/main.go` |
| `golang:golang-error-handling` | When fixing `allowlist.Detect()` and `mustGetwd()` panic |
| `ai-workflow:mattpocock:tdd` | Red-green-refactor cycle for each fix |
| `superpowers:requesting-code-review` | After each group of fixes before committing |
