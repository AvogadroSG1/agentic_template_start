# Guard hook is a deny-only net; compounds may still prompt

**Status:** accepted · 2026-06-17 · supersedes rule A4 in the mkproj design spec §6

## Decision

The shared guard hook (wired via `PreToolUse` by both Claude Code and Codex, which both
honor `exit 2` / `permissionDecision: deny`) is a **deny-only net**. It splits compound
commands and blocks the line if any constituent matches a deny rule (D1–D10, including the
secret-exposure rules D9/D10), but it **never approves** anything. Allow globs do the bulk
of approval; the guard catches dangerous exceptions only.

A vetted **compound** command auto-runs only when an allow glob matches the whole line.
Compounds that globs structurally can't match (e.g. `git status && grep foo`) will still
prompt. That residual friction is accepted.

## Considered Options

- **Hook-as-brain (approves compounds)** — the hook checks each constituent against the
  allowlist and returns an allow-decision if all pass (original spec rule A4). Rejected:
  makes the safety-critical hook bidirectional and complex; we prefer a hook that can only
  ever make things *safer*, never more permissive.

## Consequences

- Some vetted compound commands still prompt. Accepted as the price of a simple,
  one-direction safety hook.
- Per-tool allow rule is **broad prefix + targeted deny**: one `Bash(tool*)` covers all
  subcommands; individual dangerous subcommands go on the deny floor.
- The deny floor includes a **target-aware secret-exposure guard**: display/search of
  secret-bearing paths and unfiltered env dumps are blocked, with a configurable path list
  at the top of the guard. This will occasionally block a legitimate read (e.g.
  `cat .env.example`) — accepted for leak-safety.
- Codex caveat: docs note "some shell interception remains incomplete" — a known risk to
  revisit as Codex matures.
