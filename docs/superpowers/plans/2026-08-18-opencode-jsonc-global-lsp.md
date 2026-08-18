# OpenCode JSONC Configuration with Comments and Global LSP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Adopt `opencode.jsonc` with structured section comments and `"lsp": true,` for OpenCode configuration across templates, sync-allowlist reconciler, upgrade propagation, and tests.

**Architecture:** Replace the legacy `templates/common/opencode.json.tmpl` with `templates/common/opencode.jsonc.tmpl` containing JSONC comments and global LSP configuration. Update `internal/allowlist/sync.go` and `cmd/forge/main.go` to read and sync `opencode.jsonc` (with legacy fallback). Bump `upgrade.Version` to 3 in `internal/upgrade/upgrade.go` and regenerate pinned hashes.

**Tech Stack:** Go 1.24+, `embed.FS`, OpenCode JSONC configuration.

## Global Constraints

- Must follow Go 12-factor and engineering standards.
- Must keep `go test ./... -count=1` passing at every step.
- Must preserve backward compatibility in `forge sync-allowlist` for existing repos with `opencode.json`.

---

### Task 1: Template Update & Allowlist Canonical Block

**Files:**
- Create: `templates/common/opencode.jsonc.tmpl`
- Delete: `templates/common/opencode.json.tmpl`
- Modify: `internal/allowlist/sync.go:169-175`
- Test: `internal/allowlist/sync_test.go`

**Interfaces:**
- Consumes: `forge.Assets()`
- Produces: `allowlist.CanonicalBlockOpenCode(assets fs.FS, language string, frontend bool, includePersonal bool) (string, error)`

- [ ] **Step 1: Write the failing test for CanonicalBlockOpenCode reading jsonc template**

Add test in `internal/allowlist/sync_test.go`:
```go
func TestCanonicalBlockOpenCodeRendersFromJSONCTemplate(t *testing.T) {
	t.Parallel()

	block, err := CanonicalBlockOpenCode(forge.Assets(), "go", false, false)
	if err != nil {
		t.Fatalf("CanonicalBlockOpenCode() error = %v", err)
	}
	if !strings.Contains(block, `"go*": "allow"`) {
		t.Fatalf("CanonicalBlockOpenCode() missing go rule in:\n%s", block)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/allowlist -run TestCanonicalBlockOpenCodeRendersFromJSONCTemplate -count=1`
Expected: FAIL (cannot read `templates/common/opencode.jsonc.tmpl` or file missing)

- [ ] **Step 3: Create `templates/common/opencode.jsonc.tmpl`, delete `templates/common/opencode.json.tmpl`, and update `internal/allowlist/sync.go`**

Create `templates/common/opencode.jsonc.tmpl`:
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

Update `internal/allowlist/sync.go`:
```go
func CanonicalBlockOpenCode(assets fs.FS, language string, frontend bool, includePersonal bool) (string, error) {
	data, err := fs.ReadFile(assets, "templates/common/opencode.jsonc.tmpl")
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("opencode.jsonc.tmpl").Option("missingkey=error").Parse(string(data))
	if err != nil {
		return "", err
	}
...
```

- [ ] **Step 4: Run test to verify it passes**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/allowlist -run TestCanonicalBlockOpenCodeRendersFromJSONCTemplate -count=1`
Expected: PASS

---

### Task 2: CLI Allowlist Reconciler Integration

**Files:**
- Modify: `cmd/forge/main.go:188-203`

**Interfaces:**
- Consumes: `allowlist.Sync`, `allowlist.CanonicalBlockOpenCode`
- Produces: `runSyncAllowlist(args []string, assets fs.FS) error`

- [ ] **Step 1: Update `cmd/forge/main.go` to look for `opencode.jsonc` with fallback to `opencode.json`**

In `cmd/forge/main.go`:
```go
	opencodePath := filepath.Join(cwd, "opencode.jsonc")
	if _, err := os.Stat(opencodePath); os.IsNotExist(err) {
		opencodePath = filepath.Join(cwd, "opencode.json")
	}
	var opencodeStale bool
	if opencodeData, opencodeErr := os.ReadFile(opencodePath); opencodeErr == nil {
		opencodeLang, opencodeLangErr := allowlist.InferLanguage(string(opencodeData))
		if opencodeLangErr != nil {
			opencodeLang = language
		}
		opencodeBlock, opencodeBlockErr := allowlist.CanonicalBlockOpenCode(assets, opencodeLang, allowlist.InferFrontend(string(opencodeData)), includePersonal)
		if opencodeBlockErr == nil {
			ocStatus, ocErr := allowlist.Sync(opencodePath, opencodeBlock, checkOnly)
			if ocErr == nil {
				opencodeStale = ocStatus.Stale
			}
		}
	}
```

- [ ] **Step 2: Run tests in `cmd/forge`**

Run: `GOCACHE=$PWD/.cache/go-build go test ./cmd/forge -count=1`
Expected: PASS

---

### Task 3: Upgrade Package Migration & Hash Pinning

**Files:**
- Modify: `internal/upgrade/upgrade.go:14,29`
- Modify: `internal/upgrade/upgrade_test.go`
- Modify: `internal/upgrade/testdata/pinned-hashes.txt`

**Interfaces:**
- Consumes: `managedFiles`, `upgrade.Version`
- Produces: `upgrade.Run(assets fs.FS, targetDir string, checkOnly bool) (Status, error)`

- [ ] **Step 1: Update `internal/upgrade/upgrade.go` and `internal/upgrade/upgrade_test.go`**

In `internal/upgrade/upgrade.go`:
- Set `const Version = 3`
- In `managedFiles`:
  `{src: "templates/common/opencode.jsonc.tmpl", dest: "opencode.jsonc", mode: 0o644},`

In `internal/upgrade/upgrade_test.go`:
- Update mock fs entries from `"templates/common/opencode.json.tmpl"` to `"templates/common/opencode.jsonc.tmpl"`.
- Update assertions checking `opencode.json` to check `opencode.jsonc`.

- [ ] **Step 2: Regenerate pinned hashes for Version 3**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/upgrade/ -run TestVersionMatchesEmbeddedFileHashes -count=1`
Verify `internal/upgrade/testdata/pinned-hashes.txt` contains version: 3 and updated hashes.

- [ ] **Step 3: Run all upgrade package tests**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/upgrade/... -count=1`
Expected: PASS

---

### Task 4: Context / Docs Update and Full Verification

**Files:**
- Modify: `CONTEXT.md:132`

- [ ] **Step 1: Update `CONTEXT.md` to reference `opencode.jsonc`**

Replace `opencode.json` with `opencode.jsonc` in `CONTEXT.md`.

- [ ] **Step 2: Run the full test suite**

Run: `GOCACHE=$PWD/.cache/go-build go test ./... -count=1`
Expected: PASS across all packages.

- [ ] **Step 3: Build forge binary**

Run: `go build ./cmd/forge`
Expected: Success.

---

*Authored By Peter O'Connor with Assistance from OpenCode (google-vertex/gemini-3.7-flash) · 2026-08-18 · forge opencode.jsonc implementation plan*
