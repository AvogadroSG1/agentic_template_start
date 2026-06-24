# mkproj V1 Execution Progress

## Branch

- Execution worktree: `codex-mkproj-v1-execution-2026-06-23`
- Baseline verification: `GOCACHE=$PWD/.cache/go-build go test ./... -count=1` PASS on 2026-06-23
- Beads sync note: `bd dolt pull` reported no configured Dolt remote on 2026-06-23

## Slices

### secret-scan-and-hooks

- Status: complete
- Issues: `02o`, `485`, `79r`, `7sn`
- Landed: PR `#3` plus tracker sync PR `#4`

### gate-pipeline

- Status: complete
- Issues: `x2k`, `ud1`, `wqh.2`
- Landed: slice issues closed before the current conformance run; tracker sync PR `#7` is on `main`

### walking-skeleton-smoke

- Status: complete
- Issues: `uuw`, `wqh.1`
- Landed: slice issues are closed in Beads; execution worktree history retains branch `codex-mkproj-walking-skeleton`

### allowlist-reconciler

- Status: complete
- Issues: `rom`
- Landed: PR `#13`; issue closed in Beads and merged result pulled back onto the execution branch

### maintainer-refresh

- Status: complete
- Issues: `cjl`, `wqh.3`
- Landed: PR `#8` plus tracker sync PR `#9`

### guideline-publication-conformance

- Status: complete
- Issues: `zz8`, `ebp`
- Landed: `ai_support` PR `#3` published the canonical guideline files; mkproj PR `#10` enforced the floor and aligned shipped overlays

## Remaining work

- In-scope v1 slices: none
- Post-v1 backlog intentionally open: `7eu`
