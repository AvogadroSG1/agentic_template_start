# Allowlist + Deny-Floor Seed Contents (bd: agentic_template_start-aoc)

**Status:** decided 2026-06-18 (grill-with-docs) · **Author:** Peter O'Connor with Claude Code (databricks-claude-opus-4-8)

The canonical contents `mkproj` embeds and the reconciler writes into each project's
managed block. Sourced from the author's real, battle-tested global config
(`~/.claude/settings.json`) plus two research passes into Claude Code / Codex permission
internals and secret-read bypass vectors.

> **Glob format:** `Bash(cmd:*)` (colon form). The param-style `Bash(command:rm *)` is
> **silently ignored** by Claude Code (compound-bypassable) — never use it. `*` matches
> across spaces, so `Bash(tool:*)` covers all subcommands. **Deny beats allow** always.
> **Compounds auto-approve natively** when every subcommand matches an allow rule — the
> matcher is shell-operator-aware (separators `&& || ; | |& &` + newlines, and `$()`).

---

## 1. Allow — Universal dev baseline (portable; every project)

```
# version control
Bash(git status:*) Bash(git diff:*) Bash(git log:*) Bash(git add:*)
Bash(git commit:*) Bash(git checkout:*) Bash(git switch:*) Bash(git branch:*)
Bash(git stash:*) Bash(git show:*) Bash(git rev-parse:*) Bash(git remote:*)
Bash(git mv:*) Bash(git rm:*) Bash(git check-ignore:*) Bash(git init:*)
Bash(git pull:*) Bash(git fetch:*) Bash(git push:*)
Bash(gh pr:*) Bash(gh run list:*) Bash(gh search:*) Bash(gh api:*)
Bash(gh auth status) Bash(gh repo create:*)

# beads / instill / toolchain
Bash(bd:*) Bash(instill:*) Bash(mise:*) Bash(lefthook:*)

# file ops (read/navigate/create)
Bash(ls:*) Bash(cat:*) Bash(head:*) Bash(tail:*) Bash(mkdir:*) Bash(cp:*)
Bash(mv:*) Bash(ln:*) Bash(chmod:*) Bash(touch:*) Bash(find:*) Bash(tree:*)
Bash(realpath:*) Bash(dirname:*) Bash(basename:*) Bash(pwd:*) Bash(which:*)
Bash(stat:*) Bash(wc:*) Bash(du:*) Bash(cd:*) Bash(test:*)

# text processing
Bash(grep:*) Bash(rg:*) Bash(fd:*) Bash(jq:*) Bash(sed:*) Bash(awk:*)
Bash(sort:*) Bash(uniq:*) Bash(tr:*) Bash(cut:*) Bash(tee:*) Bash(diff:*)

# misc shell
Bash(echo:*) Bash(printf:*) Bash(date:*) Bash(xargs:*) Bash(timeout:*) Bash(bash:*)

# Claude tool-layer (native)
Read Glob Grep Edit Write WebSearch
```

**Deliberately decided IN:** `git rm`/`git mv` (guard still blocks `git rm -rf`),
`bash:*` (run project scripts — accepted despite `bash -c` being an interpreter; the
interpreter-class deny + sandbox FS isolation are the mitigations), `sed`/`awk` (essential
daily; the secret path-token scan catches `sed -n p .env` / `awk '1' .env`).

**Deliberately decided OUT** (stay prompting/denied): `curl`, `wget` — your only remaining
command-layer exfil control after "network on, FS-isolation-only." Use the `WebFetch`
tool for fetching. These also sit on the deny floor (D-exfil below).

## 2. Allow — Per-stack language slice (only the project's stack)

| Stack | Adds to allow |
|---|---|
| Go | `Bash(go:*)` |
| Python | `Bash(python:*) Bash(python3:*) Bash(.venv/bin/python3:*) Bash(uv:*) Bash(pip:*) Bash(pip3:*) Bash(pytest:*) Bash(ruff:*) Bash(mypy:*) Bash(source .venv/bin/activate:*)` |
| C# | `Bash(dotnet:*)` |

A Go project does **not** get `pip`; a Python project does **not** get `dotnet`. The
golden snapshot is already stack-specific, so the allow slice is too.

## 3. Allow — Personal section (tagged, optional; reconciler `--include-personal`)

Wrapped in a clearly marked block; **stripped by default** for portability, included on
the author's own machines via flag.

```
# --- BEGIN PERSONAL (Peter env) — stripped unless --include-personal ---
Bash(gw:*) Bash(rtk:*) Bash(slack-cli:*)
Bash(gcloud auth:*) Bash(gcloud services list:*)
Bash(az account:*) Bash(az rest:*) Bash(az costmanagement:*)
Bash(brew install:*) Bash(brew search:*) Bash(brew info:*)
Bash(docker images:*) Bash(docker-compose up:*) Bash(docker-compose down:*)
# --- END PERSONAL ---
```

---

## 4. Deny floor (stable; safety)

> Two enforcement points seeded together: **native `Read()/Edit()/Write()` matchers**
> (tool layer) **and** the **PreToolUse guard hook** (shell layer). Plus OS sandbox (§5).

### 4a. Native tool-layer secret matchers (`settings.json`)

```
Read(**/.env) Read(**/.env.*) Read(**/*.pem) Read(**/*.key) Read(**/id_rsa*)
Read(**/id_ed25519*) Read(**/credentials) Read(**/*secret*) Read(**/*.tfstate)
Read(~/.ssh/**) Read(~/.aws/**) Read(~/.gnupg/**) Read(~/.config/gh/**)
Read(~/.git-credentials) Read(~/.netrc) Read(~/.npmrc) Read(~/.pypirc)
Write(**/.env*) Edit(.git/**) Edit(.claude/**)
```

### 4b. Bash guard-hook rules (whole-command-line, ported from `guardrails-bin`)

| # | Rule | Decision |
|---|---|---|
| D1 | `rm -rf` / recursive-force delete (any flag order) | block |
| D2 | `git rm -rf` / mass cached removal | block |
| D3 | `git push --force`/`-f` to protected branch (`main master develop release/* production`) | block |
| D4 | history-rewriting push to protected branch | block |
| D5 | `DROP DATABASE`/`DROP SCHEMA`/`dropdb`/`TRUNCATE` (no WHERE) | block |
| D6 | `mkfs*`, `dd of=/dev/*`, `> /dev/sd*`, fork bombs, `chmod -R 777 /`, `chmod 777` | block |
| D7 | `git reset --hard` / `git clean -fdx` / `git checkout .` discarding work | block |
| D8 | `git commit --no-verify`/`-n`, `--no-gpg-sign` | block |
| **D9** | **Secret path-token scan:** any command line referencing a secret path/dir token (`.env`, `.env.*`, `*.pem`, `*.key`, `id_rsa*`, `credentials`, `*secret*`, `*.tfstate`, `.ssh/`, `.aws/`, `.gnupg/`, git-ref forms `HEAD:.env`) — regardless of binary (`cat`/`awk`/`python -c`/`git show`/`tar -O`/`base64`/`pbcopy`/`$(<…)`) | block |
| **D10** | **Env-dump verbs:** bare `env`, `printenv`, `set`, `export -p`, `declare -p/-x`, `typeset -p`, `compgen -v`, `/proc/*/environ`, `launchctl getenv` | block |
| **D11** | **Exfil channels:** `curl`, `wget`, `nc`, `ncat`, `socat`, `scp`, `sftp`, `rsync`, clipboard (`pbcopy`/`xclip`/`xsel`/`wl-copy`), `/dev/tcp/`+`/dev/udp/`, DNS-exfil (`dig`/`host`/`nslookup` w/ encoded labels) | block |
| **D12** | **System/process control:** `sudo`, `kill`/`killall`/`pkill`, `shutdown`/`reboot` | block |
| **D13** | **Remote shells:** `ssh` (covered w/ scp/rsync above) | block |
| **D14** | **Destructive docker:** `docker rm`/`rmi`/`system prune` | block |
| **D15** | **Shell history:** `history`, `fc -l` (can leak previously-typed secrets) | block |
| **D16** | **Interpreter-class (belt-and-suspenders):** `bash -c`, `sh -c`, `zsh -c`, `eval`, `python -c`, `python3 -c`, `node -e`, `perl -e`, `ruby -e` | block |

**Documented irreducible gaps** (a string hook cannot close these — the boundary is the
OS sandbox + not committing secrets): obfuscated paths inside interpreter one-liners
(`python3 -c 'open("."+"env")'`), raw `git cat-file -p <sha>` / `git stash show -p` (no
path token), hard-link/inode reuse created outside the guard's view, secrets already
exported into the environment before the session.

---

## 5. OS-level sandbox (both agents; FS isolation, network ON)

The only OS-enforced layer — survives interpreter abuse that defeats every string rule.

**Claude Code** (`.claude/settings.json`):
```json
{ "sandbox": { "enabled": true,
    "filesystem": { "denyRead": ["~/.ssh", "~/.aws", "~/.gnupg"] } } }
```
**Codex** (`~/.codex/config.toml` — on by default, set explicitly):
```toml
sandbox_mode = "workspace-write"
[sandbox_workspace_write]
network_access = true   # network ON to keep push/installs working
```

**Decision: network ON, FS-isolation-only.** Network-off breaks day-one `git push`,
`gh repo create`, `bd dolt push`, and dependency installs (`uv`/`pip`/`npm`/`dotnet
restore`/`mise install`). Egress is instead controlled by the guard's D11 exfil-channel
deny (command layer), accepting that OS-level exfil protection is traded for zero friction.
Asymmetry: Codex lacks per-path `denyRead` and a domain allowlist; Claude has both.

---

*Authored By Peter O'Connor with Assistance from Claude Code (databricks-claude-opus-4-8) · 2026-06-18 · mkproj allowlist + deny-floor seed*
