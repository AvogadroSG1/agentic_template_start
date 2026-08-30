# Handoff — forge upgrade co-ownership series

**Date:** 2026-08-30 · **Branch:** `fix/upgrade-coownership-v4` · **Repo:** forge

This is a point-in-time record of a five-defect incident cluster in `forge upgrade`/`forge init`
and the ownership-class architecture built to close it (ADR-0016, ADR-0017). Read this once for
the "what happened and why," then treat SPEC.md §3.1 and §19 as the living source of truth —
this document is not maintained going forward.

## The five defects

1. **`forge init` never stamped `.forge-infra-version`.** A freshly scaffolded repo was born at
   infrastructure version 0. The SessionStart `forge upgrade --check` advisory hook — wired into
   both `.claude/settings.json` and `.codex/hooks.json` at scaffold time — nagged on every single
   session from the moment the repo existed, even though it was already current with the embedded
   template it was just generated from.
2. **`forge upgrade` blind-overwrote co-owned hook config, destroying third-party entries — the
   `todo-cli` incident.** `.claude/settings.json` and `.codex/hooks.json` are not forge's alone:
   `bd prime --hook-json` installs a `SessionStart` matcher group, and
   `.beads/hooks/agent-fitness-functions-*` installs `PreToolUse` matcher groups, both interleaved
   into the same top-level `hooks` object forge also writes into. In the `todo-cli` repo, a routine
   `forge upgrade` blind-copied the embedded template over `.claude/settings.json`, silently
   deleting bd's `SessionStart` entry and both `agent-fitness-functions-*` `PreToolUse` groups.
   Nothing detected this at write time — the file is strict JSON, so there was no comment-based
   marker convention to violate, and nothing enforced one even if there had been.
3. **`.beads` was left world-readable (0755).** `bd init` inherits the ambient umask rather than
   setting its own directory mode, so every freshly scaffolded repo tripped `bd`'s own
   "recommended: 0700" warning immediately after `forge init` finished.
4. **A red pinned-hash gate reached main from an unbumped template edit.** `templates/common/`
   changed shape when `opencode.jsonc.tmpl` was edited without incrementing `upgrade.Version` or
   regenerating `internal/upgrade/testdata/pinned-hashes.txt` — the gate test that pins a SHA-256
   hash per managed file per declared version. The mismatch should have failed CI at the point of
   the edit; it did not, because `go test ./...` was not yet wired as its own blocking CI job at
   the time.
5. **`forge upgrade` wrote raw, unrendered template bytes to `opencode.jsonc`.** Because
   `opencode.jsonc` was folded into the same blind-byte-copy managed-file list as the truly
   wholly-owned files, `forge upgrade` copied `templates/common/opencode.jsonc.tmpl` — literal
   `{{ }}` template syntax and all — straight into generated repos' `opencode.jsonc`, instead of
   rendering it. The file was never eligible for a byte-copy in the first place: it depends on
   init-time parameters (`language`, `frontend`, `includePersonal`) that exist only in memory
   during `forge init` and are discarded once the process exits.

## The architecture that closes them

**Ownership classes** (SPEC §3.1) replace the single "all managed files get overwritten" model
the original `forge upgrade` design assumed (`docs/superpowers/specs/2026-07-16-forge-upgrade-design.md`)
with three classes, declared once per file rather than inferred at write time:

- **Wholly-owned** (`.claude/hooks/guard`, `.claude/hooks/secret-scan.sh`,
  `.opencode/plugins/forge-hooks.js`) — blind byte-copy is safe because nothing else ever writes
  these files.
- **Co-owned, hook config** (`.claude/settings.json`, `.codex/hooks.json`) — reconciled by
  **owned-entry identity** via `internal/hookcfg.Reconcile` (ADR-0016): ownership is per command
  string found in the canonical embedded template's `hooks` object, not per matcher group and not
  per file. Owned entries converge to their canonical form exactly once; a historical registry
  converges old shipped command forms in place instead of leaving stale duplicates; everything
  unclaimed — third-party matcher groups, third-party entries interleaved inside partially-owned
  groups, entire events forge never declared (`PreCompact`) — passes through byte-for-byte. This
  directly closes defect 2: a second `forge upgrade` against a reconstruction of the `todo-cli`
  incident now leaves both third-party entries intact while still converging forge's own.
- **Co-owned, init-rendered** (`opencode.jsonc`) — rendered from `.tmpl` **only when the file is
  missing** on disk, using `.forge/manifest.json`'s persisted init params
  (`text/template`, `missingkey=error`). Once it exists, `forge upgrade` never touches it again —
  this closes defect 5 by construction: there is no code path left that byte-copies the `.tmpl`.
- **Delegated** (`.beads/` via `bd`, `.apm/` via `instill`) — forge does not own their contents,
  only the permission repair it is responsible for.

**The forge manifest** (ADR-0017) closes defect 1. `forge init` calls `upgrade.Stamp` immediately
after the Phase 1 scaffold writer succeeds — before Phase 3's `git add`/`commit` — writing both
`.forge/manifest.json` (`schemaVersion`, `infraVersion`, and the init params `language`,
`frontend`, `includePersonal`, `stack`) and the legacy `.forge-infra-version` marker in the same
call, so the two can never drift apart. A freshly scaffolded repo is therefore born at the current
infrastructure version, never v0, and both files ride the initial scaffold commit — so a clone of
that repo is not stale either. `.forge-infra-version` survives as the fallback for repos
scaffolded before the manifest existed and for pre-v4 `forge` binaries that only know the bare
integer format.

**`.beads` permission repair** closes defect 3. `internal/upgrade.EnsureBeadsDirPerms` is called
immediately after `bd init` inside `forge init`, and again inside `forge upgrade`'s mutating path;
`forge upgrade --check` detects the drift (`os.Stat` only, never `os.Chmod`) and exits 1 without
mutating, so the SessionStart advisory surfaces the drift without silently fixing it out from under
an agent mid-session.

**`go test ./...` as its own blocking CI job** closes defect 4 structurally: a red unit-test gate
(including the pinned-hash gate below) can no longer land on `main` unnoticed.

### The `defaultRegistry` historical-fingerprint bump ritual

`internal/upgrade/upgrade.go` declares `defaultRegistry` (a `hookcfg.Registry`) with a
`Historical` map from a current owned command string to its prior shipped forms — e.g.
`"bd prime || true"` was previously shipped as bare `"bd prime"`. This is the mechanism that lets
`Reconcile` converge an old on-disk command to the new one in place instead of leaving it as a
stale duplicate.

**When a shipped forge hook command changes** (any edit to a command string inside
`templates/common/claude/settings.json` or `templates/common/codex/hooks.json`), the ritual is:

1. Add the **old** command form to `defaultRegistry.Historical[<new command>]` in
   `internal/upgrade/upgrade.go`, so repos still carrying the old form converge cleanly.
2. Bump `upgrade.Version` in the same file.
3. Regenerate the pinned-hash fixture:
   `go test ./internal/upgrade/ -run TestVersionMatchesEmbeddedFileHashes -count=1`
   (delete `internal/upgrade/testdata/pinned-hashes.txt` first if regenerating from scratch — the
   test only writes it when it doesn't yet exist; otherwise it fails loud, by design, per
   defect 4).

Skipping step 3 is exactly how defect 4 happened. The gate test's failure message names the fix
directly; the only way to silence it *should* be doing the above, not weakening the test.

### The manifest backfill is best-effort, not fail-loud

A legacy repo with a marker but no manifest gets a best-effort backfill during `forge upgrade`:
`allowlist.InferLanguage`/`InferFrontend` read the managed allowlist block in
`.claude/settings.local.json` to reconstruct `language`/`frontend` (`includePersonal` always
defaults `false` — it is never inferable from on-disk state). When inference succeeds,
`.forge/manifest.json` is written at the current `infraVersion` with the inferred params. When it
fails — no `settings.local.json`, or an ambiguous managed block — the upgrade still succeeds (the
infra files were already reconciled); only the bare legacy marker is written, plus a one-line
stderr notice suggesting the user create the manifest by hand. Infra reconciliation must never
block on a manifest it cannot honestly construct (ADR-0017 Consequences).

## Where to look

- `docs/adr/0016-co-owned-config-reconciled-by-owned-entry-identity.md` — the reconciler engine
  decision.
- `docs/adr/0017-committed-forge-manifest-with-legacy-marker-fallback.md` — the manifest decision.
- `docs/SPEC.md` §3.1 (ownership classes), §4.2 (init ordering invariants), §19 (full upgrade-path
  behavior and Gherkin scenarios).
- `internal/hookcfg/` — the pure reconciler engine (no file I/O).
- `internal/upgrade/` — `Stamp`, `EnsureBeadsDirPerms`, `Run`'s wiring of both, and
  `defaultRegistry`.
- `test/upgrade_coownership_test.go` — the end-to-end regression test reconstructing the
  `todo-cli`-shaped incident against the released binary; wired into `make verify-fast`.

---

*Authored By Peter O'Connor with Assistance from Claude Code (claude-sonnet-5) · 2026-08-30 · forge upgrade co-ownership series*
