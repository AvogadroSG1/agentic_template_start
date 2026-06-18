# Guard hook is a deny-only net; native matcher handles compound allow

**Status:** accepted · 2026-06-17 · amended 2026-06-18 (research correction) · supersedes rule A4 in the mkproj design spec §6

## Decision

The shared guard hook (wired via `PreToolUse` by both Claude Code and Codex, which both
honor `exit 2` / `permissionDecision: deny`) is a **deny-only net**. It blocks the line if
any constituent matches a deny rule, but it **never approves** anything. The guard catches
dangerous exceptions only.

> **Amended 2026-06-18 — the compound-prompt premise was false.** This ADR originally
> claimed vetted compounds like `git status && grep foo` "will still prompt … accepted
> friction." Research into the official docs
> ([code.claude.com/docs/en/permissions](https://code.claude.com/docs/en/permissions))
> proved otherwise: **Claude Code's matcher is shell-operator-aware and splits compound
> commands**, matching each subcommand independently against the allow rules (separators:
> `&&`, `||`, `;`, `|`, `|&`, `&`, newlines; commands inside `$()` are also inspected). So
> `git status && grep foo` **auto-approves natively** when both `git status` and `grep` are
> on the allowlist — no hook allow-logic needed, no residual prompt. The guard stays
> deny-only purely for *safety*, not because the native layer can't allow compounds. The
> seed's job is therefore to make the **allowlist complete**; compounds then work for free.

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
