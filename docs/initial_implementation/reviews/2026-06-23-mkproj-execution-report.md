# mkproj V1 Execution Report

## Completed items

- Secret-scan plus hook-chain slice completed on branch `codex-mkproj-secret-scan-hooks`.
- Review findings status for this slice:
  - `79r` was partially correct: shared scanner existed, but D10 coverage missed `launchctl getenv` and `/proc/*/environ`; fixed in both repo-root and template scanner assets.
  - `7sn` was correct: repo-root Claude/Codex hook wiring and deny-floor guard parity were incomplete; fixed with shared guard asset plus Claude/Codex `PreToolUse` and non-blocking `SessionStart` wiring.
  - `485` was partially correct: the missing seam coverage was real, but the suggested remedy to remove `--force` was wrong. Real Lefthook behavior under Beads requires `--force`, then an explicit chain repair; fixed in `internal/init` plus seam tests.
- BDD/spec evidence added:
  - `test/security_hooks_test.go` covers root/template hook wiring, D1-D16 deny-floor behavior, safe compound allowance, and delegated D10 expansions.
  - `internal/init/init_test.go` now proves post-install chain repair keeps Beads hooks first and Lefthook second after forced install.
  - `internal/scaffold/writer_test.go` now proves template guard parity with the shared repo-root guard.
- Skills validated:
  - Implementation check: `receiving-code-review`, `systematic-debugging`
  - Review gate: `verification-before-completion`

## Assumptions made

- The execution branch remains long-lived while each independent slice lands through its own temporary PR branch.
- `agentic_template_start-7eu` remains intentionally out of scope because SPEC §9 and the Beads graph both treat it as post-v1 work.
- The hook-chain fix MUST preserve `lefthook install --force` because `bd init` sets `core.hooksPath` to `.beads/hooks`; removing `--force` would fail installation instead of preserving Beads.

## Errors / blockers encountered

- `bd dolt pull` reported no configured Dolt remote, so Beads state cannot be synced through Dolt in this environment.
- A live `mkproj init` fixture could not be carried to full completion because `instill init` in this environment reports `manifest already exists; use --force to reinitialize`, even in an external temp fixture. The hook-chain seam is therefore verified by the focused init test plus prior real Lefthook reproduction of the broken default behavior.

## Integration notes

- Each slice MUST merge to `main` before the next slice starts so downstream seams build on merged behavior rather than stacked local commits.
- The execution branch is the orchestration spine and SHOULD be fast-forwarded from `origin/main` after every merged slice.
- Downstream generated repos now receive matching repo-root/template scanner and guard assets, plus Claude/Codex hook wiring that shares one deny-only guard seam.
- `internal/init` now assumes any post-Beads forced Lefthook install may leave `.old` wrappers behind and repairs them into `*.old` + `*.lefthook` + wrapper form before the initial commit.
