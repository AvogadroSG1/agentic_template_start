# Per-project allowlist with embedded canonical source

**Status:** accepted · 2026-06-17

## Decision

The vetted Bash allowlist lives **per-project** inside a versioned managed block in each
repo's Claude/Codex settings — not in global `~/.claude`. The canonical definition is
**embedded in the `mkproj` binary** (`embed.FS`, like the golden templates). A reconciler
(`mkproj sync-allowlist`) rewrites the managed block from that source, triggered
**notify-only** on SessionStart (it detects staleness and prompts; it never auto-mutates).

## Considered Options

- **Global `~/.claude` only** — simplest, no per-repo duplication. Rejected: not portable
  to collaborators, fresh machines, or cloud agents — the exact contexts where "it just
  works on day one" matters most.
- **Committed per-repo source file** — self-contained but no cross-repo propagation; the
  allowlist would rot independently in every repo.

## Consequences

- The allowlist is duplicated into every repo (accepted cost) in exchange for portability
  and self-containment.
- Allowlist and deny floor are **two concepts with one shared embedded source file**
  (separate sections), refreshing on independent cadences.
- Refresh requires the `mkproj` binary present on the machine doing the refresh.
- Notify-only means a repo can be briefly stale until the author runs the reconciler — a
  deliberate trade against surprise mutations / commit-diff churn.
