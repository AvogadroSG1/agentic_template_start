# OpenCode JSONC Configuration with Comments and Global LSP Support

- **Date:** 2026-08-18
- **Status:** Approved
- **Issue:** `agentic_template_start-squ`

## 1. Problem Statement

OpenCode natively supports JSON with Comments (`.jsonc`). Previously, `forge` scaffolded `opencode.json` without comments and used per-language dictionary maps for the LSP configuration (`"lsp": { "gopls": true }`, etc.).

Setting `"lsp": true` globally instructs OpenCode to enable default language server protocol tooling automatically for supported languages. Additionally, using `opencode.jsonc` allows informative comments explaining each section (schema, LSP, read deny-list, edit permissions, bash command allowlist, and built-in tools).

## 2. Goals & Non-Goals

### Goals
- Rename template `templates/common/opencode.json.tmpl` to `templates/common/opencode.jsonc.tmpl` so `forge init` produces `opencode.jsonc`.
- Update the LSP configuration in the template to `"lsp": true,`.
- Add structured JSONC comments explaining each configuration section.
- Support `opencode.jsonc` (with backwards compatibility fallback to `opencode.json`) in `forge sync-allowlist`.
- Update `internal/upgrade/upgrade.go` managed files list to track `opencode.jsonc`.
- Bump `internal/upgrade/upgrade.go` `Version` from 2 to 3 and regenerate `pinned-hashes.txt`.
- Update test fixtures, tests, and documentation.

### Non-Goals
- Altering the underlying allowlist or deny-floor permission rules themselves.

## 3. Detailed Design

### 3.1 Template (`templates/common/opencode.jsonc.tmpl`)

```jsonc
{
  // OpenCode Configuration (https://opencode.ai)
  "$schema": "https://opencode.ai/config.json",

  // Enable default Language Server Protocol support
  "lsp": true,

  "permission": {
    // Read permissions: allow all repository files except sensitive secrets
    "read": {
      "*": "allow",
      "// BEGIN FORGE ALLOW v:2",
      "**/.env": "deny",
      "**/.env.*": "deny",
      "**/*.pem": "deny",
      "**/*.key": "deny",
      "**/id_rsa*": "deny",
      "**/id_ed25519*": "deny",
      "**/credentials": "deny",
      "**/*secret*": "deny",
      "**/*.tfstate": "deny",
      "~/.ssh/**": "deny",
      "~/.aws/**": "deny",
      "~/.gnupg/**": "deny",
      "~/.config/gh/**": "deny",
      "~/.git-credentials": "deny",
      "~/.netrc": "deny",
      "~/.npmrc": "deny",
      "~/.pypirc": "deny"
    },

    // Edit permissions: allow general file editing while protecting git and opencode internals
    "edit": {
      "*": "allow",
      ".git/**": "deny",
      ".opencode/**": "deny"
    },

    // Bash permissions: vetted commands for git, build, test, and package tools
    "bash": {
      "*": "ask",
      "git status*": "allow",
      "git diff*": "allow",
      "git log*": "allow",
      "git add*": "allow",
      "git commit*": "allow",
      "git checkout*": "allow",
      "git switch*": "allow",
      "git branch*": "allow",
      "git stash*": "allow",
      "git show*": "allow",
      "git rev-parse*": "allow",
      "git remote*": "allow",
      "git mv*": "allow",
      "git rm*": "allow",
      "git check-ignore*": "allow",
      "git init*": "allow",
      "git pull*": "allow",
      "git fetch*": "allow",
      "git push*": "allow",
      "gh pr*": "allow",
      "gh run list*": "allow",
      "gh search*": "allow",
      "gh api*": "allow",
      "gh auth status": "allow",
      "gh repo create*": "allow",
      "bd*": "allow",
      "instill*": "allow",
      "apm*": "allow",
      "mise*": "allow",
      "lefthook*": "allow",
      "ls*": "allow",
      "cat*": "allow",
      "head*": "allow",
      "tail*": "allow",
      "mkdir*": "allow",
      "cp*": "allow",
      "mv*": "allow",
      "ln*": "allow",
      "chmod*": "allow",
      "touch*": "allow",
      "find*": "allow",
      "tree*": "allow",
      "realpath*": "allow",
      "dirname*": "allow",
      "basename*": "allow",
      "pwd*": "allow",
      "which*": "allow",
      "stat*": "allow",
      "wc*": "allow",
      "du*": "allow",
      "cd*": "allow",
      "test*": "allow",
      "grep*": "allow",
      "rg*": "allow",
      "fd*": "allow",
      "jq*": "allow",
      "sed*": "allow",
      "awk*": "allow",
      "sort*": "allow",
      "uniq*": "allow",
      "tr*": "allow",
      "cut*": "allow",
      "tee*": "allow",
      "diff*": "allow",
      "echo*": "allow",
      "printf*": "allow",
      "date*": "allow",
      "xargs*": "allow",
      "timeout*": "allow",
      "bash*": "allow",
{{- if eq .Language "go" }}
      "go*": "allow",
{{- else if eq .Language "python" }}
      "python*": "allow",
      "python3*": "allow",
      ".venv/bin/python3*": "allow",
      "uv*": "allow",
      "pip*": "allow",
      "pip3*": "allow",
      "pytest*": "allow",
      "ruff*": "allow",
      "pyright*": "allow",
      "source .venv/bin/activate*": "allow",
{{- else if eq .Language "csharp" }}
      "dotnet*": "allow",
{{- else if eq .Language "typescript" }}
      "node*": "allow",
      "npm*": "allow",
      "npx*": "allow",
      "tsc*": "allow",
      "vitest*": "allow",
      "eslint*": "allow",
      "prettier*": "allow",
{{- end }}
{{- if .Frontend }}
      "node*": "allow",
      "npm*": "allow",
      "npx*": "allow",
{{- end }}
{{- if .IncludePersonal }}
      "gw*": "allow",
      "rtk*": "allow",
      "slack-cli*": "allow",
      "gcloud auth*": "allow",
      "gcloud services list*": "allow",
      "az account*": "allow",
      "az rest*": "allow",
      "az costmanagement*": "allow",
      "brew install*": "allow",
      "brew search*": "allow",
      "brew info*": "allow",
      "docker images*": "allow",
{{- end }}
      "// END FORGE ALLOW"
    },

    // Built-in tools allowed unconditionally
    "glob": "allow",
    "grep": "allow",
    "webfetch": "allow",
    "skill": "allow",
    "todowrite": "allow"
  }
}
```

### 3.2 Allowlist Package & CLI (`internal/allowlist/sync.go` & `cmd/forge/main.go`)
- `CanonicalBlockOpenCode` in `internal/allowlist/sync.go` reads `templates/common/opencode.jsonc.tmpl`.
- In `cmd/forge/main.go`, `runSyncAllowlist` first checks for `filepath.Join(cwd, "opencode.jsonc")`. If not found, it checks `filepath.Join(cwd, "opencode.json")`.

### 3.3 Upgrade Package (`internal/upgrade/upgrade.go`)
- Change managed entry from `templates/common/opencode.json.tmpl` -> `opencode.json` to:
  `{src: "templates/common/opencode.jsonc.tmpl", dest: "opencode.jsonc", mode: 0o644}`
- Increment `Version = 3`.
- Regenerate `pinned-hashes.txt`.

### 3.4 Verification Plan
- Unit tests: `internal/allowlist`, `internal/upgrade`, `internal/scaffold`, `cmd/forge`.
- Full test suite: `go test ./... -count=1`.
- Binary compilation: `go build ./cmd/forge`.

---

*Authored By Peter O'Connor with Assistance from OpenCode (google-vertex/gemini-3.7-flash) · 2026-08-18 · forge opencode.jsonc with comments and global LSP*
