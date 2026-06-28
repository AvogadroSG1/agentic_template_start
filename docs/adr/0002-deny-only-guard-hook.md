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
  the native matcher already does constituent-aware *allow* (see amendment), so a
  bidirectional hook is unnecessary; we keep a hook that can only ever make things *safer*.

## Consequences

- Per-tool allow rule format is **`Bash(tool:*)`** (the colon form), not `Bash(tool*)`.
  Two corrections from research: (1) the param-style `Bash(command:rm *)` is **silently
  ignored** by Claude Code as compound-bypassable — never use it; (2) `*` matches across
  spaces, so `Bash(tool:*)` covers all subcommands at any depth. Individual dangerous
  subcommands go on the deny floor.
- The deny floor's secret protection is a **whole-command-line path-token scan** ("protect
  the asset path, not the verb"), NOT a command-name list. Research
  ([permissions docs](https://code.claude.com/docs/en/permissions)) confirms a name list
  (cat/grep/head…) is trivially bypassed by `python3 -c 'open(".env")'`, `awk '1' .env`,
  `git show HEAD:.env`. The seed ports the proven scan from the author's existing
  `guardrails-bin`. Irreducible gaps (obfuscated paths in interpreter one-liners, raw
  `git cat-file -p <sha>`) are documented, not pretended-closed.
- **Defense in depth (Anthropic's stated doctrine — command-string denies are "fragile").**
  The seed layers: (1) native `Read()/Edit()/Write()` secret-path deny matchers; (2) the
  path-token guard hook; (3) **OS-level sandbox on both agents** (Claude
  `sandbox.enabled`, Codex `sandbox_mode=workspace-write`) for FS isolation; (4)
  interpreter-class deny (`bash -c`, `python -c`, `eval`, …) as belt-and-suspenders.
- **Sandbox network: ON, FS-isolation-only.** Network-off would break day-one
  `git push`/`gh repo create`/`bd dolt push`/dependency installs. The seed keeps network
  enabled and relies on the guard's exfil-channel deny (curl/wget/nc/scp + `/dev/tcp`) for
  egress control — accepting that OS-level exfil protection is traded for zero network
  friction.
- Codex parity verified: Codex has `PreToolUse` (same exit-2/deny contract) **and** an
  OS sandbox (Seatbelt/bubblewrap, on by default). Asymmetry to design around: Codex has
  no per-path `denyRead` and only binary `network_access` (no domain allowlist).
