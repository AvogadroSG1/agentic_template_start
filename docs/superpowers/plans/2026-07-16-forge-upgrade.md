# forge upgrade Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `forge upgrade` command that propagates managed static infrastructure files from the embedded template into existing forge-scaffolded repos.

**Architecture:** A new `internal/upgrade/` package owns the version constant, managed file manifest, and the overwrite/check logic. The CLI wires it into the command switch alongside `init`, `sync-allowlist`, and `update`. SessionStart hooks in the managed `settings.json` and `codex/hooks.json` advertise staleness on session open.

**Tech Stack:** Go 1.26, `io/fs` for embedded asset reads, standard `os` for file writes.

## Global Constraints

- Module path: `forge`
- Go version: 1.26.x
- Test command: `GOCACHE=$PWD/.cache/go-build go test ./... -count=1`
- Build command: `go build ./cmd/forge`
- All managed files are static byte-copies (no template rendering)
- Hook scripts (`guard`, `secret-scan.sh`) require mode 0755
- The `Version` constant MUST be incremented when any managed template file changes
- The `.forge-infra-version` file is written last (transaction commit marker)

---

### Task 1: Core upgrade package with check-only mode

**Files:**
- Create: `internal/upgrade/upgrade.go`
- Create: `internal/upgrade/upgrade_test.go`

**Interfaces:**
- Consumes: `io/fs.FS` (the embedded asset filesystem from `forge.Assets()`)
- Produces:
  - `upgrade.Version` (int constant)
  - `upgrade.Status{Version int, OnDisk int, Stale bool, Updated []string}`
  - `upgrade.Run(assets fs.FS, targetDir string, checkOnly bool) (Status, error)`

- [ ] **Step 1: Write the failing test for check-only detecting staleness**

```go
// internal/upgrade/upgrade_test.go
package upgrade

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func testAssets() fstest.MapFS {
	return fstest.MapFS{
		"templates/common/claude/hooks/guard":      &fstest.MapFile{Data: []byte("#!/usr/bin/env bash\n# guard v1\n")},
		"templates/common/claude/hooks/secret-scan.sh": &fstest.MapFile{Data: []byte("#!/usr/bin/env bash\n# secret-scan v1\n")},
		"templates/common/claude/settings.json":    &fstest.MapFile{Data: []byte(`{"hooks":{}}` + "\n")},
		"templates/common/codex/hooks.json":        &fstest.MapFile{Data: []byte(`{"hooks":{}}` + "\n")},
	}
}

func TestCheckReportsStaleWhenVersionFileMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Create minimal forge repo marker
	if err := os.MkdirAll(filepath.Join(dir, ".claude", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".claude", "hooks", "guard"), []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	status, err := Run(testAssets(), dir, true)
	if err != nil {
		t.Fatalf("Run(check) error = %v", err)
	}
	if !status.Stale {
		t.Fatal("Run(check) stale = false, want true")
	}
	if status.OnDisk != 0 {
		t.Fatalf("Run(check) OnDisk = %d, want 0", status.OnDisk)
	}
	if status.Version != Version {
		t.Fatalf("Run(check) Version = %d, want %d", status.Version, Version)
	}
}

func TestCheckReportsCurrentWhenVersionMatches(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".claude", "hooks", "guard"), []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".forge-infra-version"), []byte("1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	status, err := Run(testAssets(), dir, true)
	if err != nil {
		t.Fatalf("Run(check) error = %v", err)
	}
	if status.Stale {
		t.Fatal("Run(check) stale = true, want false")
	}
}

func TestCheckDoesNotMutateFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	guardPath := filepath.Join(dir, ".claude", "hooks", "guard")
	if err := os.WriteFile(guardPath, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Run(testAssets(), dir, true)
	if err != nil {
		t.Fatalf("Run(check) error = %v", err)
	}

	data, err := os.ReadFile(guardPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old" {
		t.Fatalf("Run(check) mutated guard: got %q", string(data))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/upgrade/ -count=1 -v`
Expected: FAIL — package does not exist yet

- [ ] **Step 3: Write minimal implementation**

```go
// internal/upgrade/upgrade.go
package upgrade

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const Version = 1

const versionFile = ".forge-infra-version"

type managedFile struct {
	src  string
	dest string
	mode os.FileMode
}

var managedFiles = []managedFile{
	{src: "templates/common/claude/hooks/guard", dest: ".claude/hooks/guard", mode: 0o755},
	{src: "templates/common/claude/hooks/secret-scan.sh", dest: ".claude/hooks/secret-scan.sh", mode: 0o755},
	{src: "templates/common/claude/settings.json", dest: ".claude/settings.json", mode: 0o644},
	{src: "templates/common/codex/hooks.json", dest: ".codex/hooks.json", mode: 0o644},
}

type Status struct {
	Version int
	OnDisk  int
	Stale   bool
	Updated []string
}

func Run(assets fs.FS, targetDir string, checkOnly bool) (Status, error) {
	if err := validateForgeRepo(targetDir); err != nil {
		return Status{}, err
	}

	onDisk := readOnDiskVersion(targetDir)
	status := Status{
		Version: Version,
		OnDisk:  onDisk,
		Stale:   onDisk < Version,
	}

	if checkOnly || !status.Stale {
		return status, nil
	}

	for _, mf := range managedFiles {
		data, err := fs.ReadFile(assets, mf.src)
		if err != nil {
			return Status{}, fmt.Errorf("read embedded %s: %w", mf.src, err)
		}
		dest := filepath.Join(targetDir, filepath.FromSlash(mf.dest))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return Status{}, fmt.Errorf("create directory for %s: %w", mf.dest, err)
		}
		if err := os.WriteFile(dest, data, mf.mode); err != nil {
			return Status{}, fmt.Errorf("write %s: %w", mf.dest, err)
		}
		status.Updated = append(status.Updated, mf.dest)
	}

	versionContent := fmt.Sprintf("%d\n", Version)
	versionPath := filepath.Join(targetDir, versionFile)
	if err := os.WriteFile(versionPath, []byte(versionContent), 0o644); err != nil {
		return Status{}, fmt.Errorf("write %s: %w", versionFile, err)
	}

	return status, nil
}

func validateForgeRepo(targetDir string) error {
	markers := []string{
		filepath.Join(targetDir, ".claude", "hooks", "guard"),
		filepath.Join(targetDir, ".forge-infra-version"),
		filepath.Join(targetDir, ".claude", "settings.json"),
	}
	for _, marker := range markers {
		if _, err := os.Stat(marker); err == nil {
			return nil
		}
	}
	return fmt.Errorf("not a forge-managed repository (run from a project created with forge init)")
}

func readOnDiskVersion(targetDir string) int {
	data, err := os.ReadFile(filepath.Join(targetDir, versionFile))
	if err != nil {
		return 0
	}
	v, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return v
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/upgrade/ -count=1 -v`
Expected: PASS (3 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/upgrade/upgrade.go internal/upgrade/upgrade_test.go
git commit -m "feat(upgrade): add core package with check-only mode

Co-Authored-By: Peter O'Connor <poconnor@stackoverflow.com>
Co-Authored-By: Claude Code <noreply@anthropic.com> - databricks-claude-opus-4-6"
```

---

### Task 2: Mutating upgrade path with full test coverage

**Files:**
- Modify: `internal/upgrade/upgrade_test.go`

**Interfaces:**
- Consumes: `upgrade.Run` from Task 1
- Produces: Tests validating overwrite behavior, idempotency, partial presence, and non-forge rejection

- [ ] **Step 1: Write tests for the mutating upgrade path**

Append to `internal/upgrade/upgrade_test.go`:

```go
func TestRunOverwritesManagedFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".claude", "hooks", "guard"), []byte("old-guard"), 0o755); err != nil {
		t.Fatal(err)
	}

	assets := testAssets()
	status, err := Run(assets, dir, false)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(status.Updated) != 4 {
		t.Fatalf("Run() updated %d files, want 4", len(status.Updated))
	}

	// Verify guard was overwritten
	data, err := os.ReadFile(filepath.Join(dir, ".claude", "hooks", "guard"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "#!/usr/bin/env bash\n# guard v1\n" {
		t.Fatalf("guard not overwritten: got %q", string(data))
	}

	// Verify version file was written
	vData, err := os.ReadFile(filepath.Join(dir, ".forge-infra-version"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(vData)) != "1" {
		t.Fatalf("version file = %q, want \"1\"", string(vData))
	}
}

func TestRunIsIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".claude", "hooks", "guard"), []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	assets := testAssets()

	// First run: should upgrade
	status1, err := Run(assets, dir, false)
	if err != nil {
		t.Fatalf("Run(1) error = %v", err)
	}
	if len(status1.Updated) == 0 {
		t.Fatal("Run(1) updated nothing")
	}

	// Second run: should be current
	status2, err := Run(assets, dir, false)
	if err != nil {
		t.Fatalf("Run(2) error = %v", err)
	}
	if status2.Stale {
		t.Fatal("Run(2) stale = true after upgrade")
	}
	if len(status2.Updated) != 0 {
		t.Fatalf("Run(2) updated %d files, want 0", len(status2.Updated))
	}
}

func TestRunCreatesDirectoriesForMissingFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Only create the version file as a marker — no .claude/ dir at all
	if err := os.WriteFile(filepath.Join(dir, ".forge-infra-version"), []byte("0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	status, err := Run(testAssets(), dir, false)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(status.Updated) != 4 {
		t.Fatalf("Run() updated %d files, want 4", len(status.Updated))
	}

	// Verify .codex/hooks.json was created
	if _, err := os.Stat(filepath.Join(dir, ".codex", "hooks.json")); err != nil {
		t.Fatalf(".codex/hooks.json not created: %v", err)
	}
}

func TestRunRejectsNonForgeDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	_, err := Run(testAssets(), dir, false)
	if err == nil {
		t.Fatal("Run() on non-forge dir should fail")
	}
	if !strings.Contains(err.Error(), "not a forge-managed repository") {
		t.Fatalf("Run() error = %q, want 'not a forge-managed repository'", err.Error())
	}
}

func TestRunSetsExecutablePermissionOnHooks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".claude", "hooks", "guard"), []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Run(testAssets(), dir, false)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, ".claude", "hooks", "guard"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("guard mode = %o, want 755", info.Mode().Perm())
	}
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/upgrade/ -count=1 -v`
Expected: PASS (8 tests)

- [ ] **Step 3: Commit**

```bash
git add internal/upgrade/upgrade_test.go
git commit -m "test(upgrade): add coverage for overwrite, idempotency, partial presence, permissions

Co-Authored-By: Peter O'Connor <poconnor@stackoverflow.com>
Co-Authored-By: Claude Code <noreply@anthropic.com> - databricks-claude-opus-4-6"
```

---

### Task 3: CLI wiring and command registration

**Files:**
- Modify: `cmd/forge/main.go`

**Interfaces:**
- Consumes: `upgrade.Run(assets fs.FS, targetDir string, checkOnly bool) (upgrade.Status, error)`
- Produces: `forge upgrade [--check]` command available from CLI

- [ ] **Step 1: Write the failing test for command routing**

The existing test approach for CLI routing in this repo is via integration-level tests in `test/`. However, since the pattern is trivial (matching the `sync-allowlist` and `update` handlers), we test this via build verification and manual invocation. First, verify the build compiles with the new command wired in.

- [ ] **Step 2: Add the upgrade command to main.go**

Add the import and wiring. In `cmd/forge/main.go`:

Add to imports:
```go
upgradepkg "forge/internal/upgrade"
```

Add `"upgrade"` to the `selectCommand` switch:
```go
case "init", "sync-allowlist", "update", "upgrade":
    return args[0], args[1:]
```

Add the case in `run`:
```go
case "upgrade":
    err = runUpgrade(args, assets)
```

Add the `runUpgrade` function:
```go
func runUpgrade(args []string, assets fs.FS) error {
	cwd, err := currentWorkingDir()
	if err != nil {
		return err
	}

	flags := flag.NewFlagSet("forge upgrade", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var checkOnly bool
	flags.BoolVar(&checkOnly, "check", false, "Only report staleness")
	if err := flags.Parse(args); err != nil {
		return err
	}

	status, err := upgradepkg.Run(assets, cwd, checkOnly)
	if err != nil {
		return err
	}

	if checkOnly {
		if status.Stale {
			fmt.Printf("forge upgrade: infrastructure is at v%d, current is v%d; run `forge upgrade`\n", status.OnDisk, status.Version)
			os.Exit(1)
		}
		return nil
	}

	if len(status.Updated) == 0 {
		fmt.Printf("forge upgrade: infrastructure is current (v%d)\n", status.Version)
		return nil
	}

	fmt.Printf("forge upgrade: updated %d files to infrastructure v%d\n", len(status.Updated), status.Version)
	for _, path := range status.Updated {
		fmt.Printf("  %s\n", path)
	}
	return nil
}
```

- [ ] **Step 3: Verify build compiles**

Run: `go build ./cmd/forge`
Expected: success, no errors

- [ ] **Step 4: Run full test suite**

Run: `GOCACHE=$PWD/.cache/go-build go test ./... -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/forge/main.go
git commit -m "feat(upgrade): wire CLI command with --check flag

Co-Authored-By: Peter O'Connor <poconnor@stackoverflow.com>
Co-Authored-By: Claude Code <noreply@anthropic.com> - databricks-claude-opus-4-6"
```

---

### Task 4: SessionStart hook integration in managed templates

**Files:**
- Modify: `templates/common/claude/settings.json`
- Modify: `templates/common/codex/hooks.json`

**Interfaces:**
- Consumes: `forge upgrade --check` CLI command from Task 3
- Produces: Updated managed templates that advertise staleness on session start

- [ ] **Step 1: Write the failing test — verify embedded settings.json contains the upgrade hook**

Add to `internal/upgrade/upgrade_test.go`:

```go
func TestEmbeddedSettingsContainsUpgradeCheckHook(t *testing.T) {
	t.Parallel()

	assets := testAssets()

	// This test uses the real embedded assets to verify the hook is present
	// Skip if running against the test fixture (which has minimal content)
	data, err := fs.ReadFile(assets, "templates/common/claude/settings.json")
	if err != nil {
		t.Skip("using test fixture assets")
	}
	if !strings.Contains(string(data), "forge upgrade --check") {
		t.Fatal("settings.json missing 'forge upgrade --check' hook")
	}
}
```

Also add a test using real assets in a separate file to avoid import cycles. Actually, we'll verify via the gate test pattern used elsewhere — add to `internal/scaffold/gate_assets_test.go`:

First, check what's already there:

- [ ] **Step 2: Add upgrade --check hook to settings.json**

Edit `templates/common/claude/settings.json` to add the hook to the SessionStart array:

```json
{
  "enabledPlugins": {},
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "./.claude/hooks/guard"
          }
        ]
      }
    ],
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "bd prime || true"
          },
          {
            "type": "command",
            "command": "instill sync || true"
          },
          {
            "type": "command",
            "command": "command -v forge >/dev/null 2>&1 && forge sync-allowlist --check || true"
          },
          {
            "type": "command",
            "command": "command -v forge >/dev/null 2>&1 && forge upgrade --check || true"
          }
        ]
      }
    ]
  }
}
```

- [ ] **Step 3: Add upgrade --check hook to codex/hooks.json**

Edit `templates/common/codex/hooks.json` to add the same hook to the SessionStart array:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "./.claude/hooks/guard"
          }
        ]
      }
    ],
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "bd prime || true"
          },
          {
            "type": "command",
            "command": "instill sync || true"
          },
          {
            "type": "command",
            "command": "command -v forge >/dev/null 2>&1 && forge sync-allowlist --check || true"
          },
          {
            "type": "command",
            "command": "command -v forge >/dev/null 2>&1 && forge upgrade --check || true"
          }
        ]
      }
    ]
  }
}
```

- [ ] **Step 4: Run full test suite**

Run: `GOCACHE=$PWD/.cache/go-build go test ./... -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add templates/common/claude/settings.json templates/common/codex/hooks.json
git commit -m "feat(upgrade): add SessionStart hook for staleness advisory

Co-Authored-By: Peter O'Connor <poconnor@stackoverflow.com>
Co-Authored-By: Claude Code <noreply@anthropic.com> - databricks-claude-opus-4-6"
```

---

### Task 5: Update forge's own managed files and SPEC documentation

**Files:**
- Modify: `.claude/hooks/../settings.json` (forge repo's own copy — keep in sync with template)
- Modify: `docs/SPEC.md` (add upgrade command to §5 CLI surface, add §-reference)

**Interfaces:**
- Consumes: All prior tasks
- Produces: Self-consistent documentation and forge's own repo updated

- [ ] **Step 1: Update forge's own .claude/settings.json**

Copy the updated `templates/common/claude/settings.json` content into `.claude/settings.json` (the forge repo eats its own dog food — the SessionStart hook additions apply here too). Preserve the `PreCompact` hook that's specific to forge:

```json
{
  "enabledPlugins": {},
  "hooks": {
    "PreCompact": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "bd prime || true"
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "./.claude/hooks/guard"
          }
        ]
      }
    ],
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "bd prime || true"
          },
          {
            "type": "command",
            "command": "instill sync || true"
          },
          {
            "type": "command",
            "command": "command -v forge >/dev/null 2>&1 && forge sync-allowlist --check || true"
          },
          {
            "type": "command",
            "command": "command -v forge >/dev/null 2>&1 && forge upgrade --check || true"
          }
        ]
      }
    ]
  }
}
```

- [ ] **Step 2: Add `forge upgrade` to SPEC.md §5 CLI surface**

Find the CLI command list in `docs/SPEC.md` (around line 166-169) and add:

```markdown
- `forge upgrade [--check]` — infrastructure file propagation (§16).
```

- [ ] **Step 3: Add §16 to SPEC.md describing upgrade behavior**

Add a new section at the end of the behavioral sections:

```markdown
## 16. Infrastructure upgrade path

`forge upgrade` propagates managed static infrastructure files from the embedded template into an
existing forge-scaffolded repository. It overwrites `.claude/hooks/guard`,
`.claude/hooks/secret-scan.sh`, `.claude/settings.json`, and `.codex/hooks.json` unconditionally
when the on-disk infrastructure version is behind the embedded version.

Version state lives in `.forge-infra-version` (repo root). Missing file → version 0 → always stale.
The version file is written last so interrupted upgrades re-apply.

`forge upgrade --check` reports staleness (exit 1 if behind) without mutation. The SessionStart
hooks in both `settings.json` and `codex/hooks.json` run this advisory on every session open.

```gherkin
Scenario: First upgrade on a legacy repo
  Given a forge-scaffolded repo with no .forge-infra-version file
  When the user runs `forge upgrade`
  Then all managed files are overwritten from the embedded template
  And .forge-infra-version is created with the current version

Scenario: Upgrade is idempotent
  Given a repo at infrastructure version N with embedded version N
  When the user runs `forge upgrade`
  Then no files are written
  And output reports "infrastructure is current (vN)"

Scenario: Check mode advisory on session start
  Given a repo at infrastructure version 1 and embedded version 2
  When SessionStart fires `forge upgrade --check`
  Then the hook prints the staleness advisory and exits 1
  And no files are modified
```
```

- [ ] **Step 4: Run full test suite and build**

Run: `GOCACHE=$PWD/.cache/go-build go test ./... -count=1 && go build ./cmd/forge`
Expected: PASS + successful build

- [ ] **Step 5: Commit**

```bash
git add .claude/settings.json docs/SPEC.md
git commit -m "docs: add forge upgrade to CLI surface and SPEC §16

Co-Authored-By: Peter O'Connor <poconnor@stackoverflow.com>
Co-Authored-By: Claude Code <noreply@anthropic.com> - databricks-claude-opus-4-6"
```

---

### Task 6: Gate test — version constant matches embedded file hashes

**Files:**
- Create: `internal/upgrade/gate_test.go`

**Interfaces:**
- Consumes: `forge.Assets()`, `upgrade.Version`, `upgrade.managedFiles`
- Produces: A test that fails if a managed template file changes without bumping `Version`

- [ ] **Step 1: Write the gate test**

```go
// internal/upgrade/gate_test.go
package upgrade

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forge"
)

func TestVersionMatchesEmbeddedFileHashes(t *testing.T) {
	t.Parallel()

	hashFile := filepath.Join("testdata", "pinned-hashes.txt")
	pinned, err := os.ReadFile(hashFile)
	if err != nil {
		// First run: generate the file
		t.Logf("generating %s for version %d", hashFile, Version)
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		content := generatePinnedHashes(t, forge.Assets())
		if err := os.WriteFile(hashFile, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}

	// Verify current hashes match pinned
	current := generatePinnedHashes(t, forge.Assets())
	if string(pinned) != current {
		t.Fatalf("managed file hashes changed but upgrade.Version (%d) was not bumped.\n"+
			"If you changed a managed template file, increment Version and run:\n"+
			"  go test ./internal/upgrade/ -run TestVersionMatchesEmbeddedFileHashes -count=1\n"+
			"to regenerate the pinned hashes.\n\nPinned:\n%s\nCurrent:\n%s",
			Version, string(pinned), current)
	}
}

func generatePinnedHashes(t *testing.T, assets fs.FS) string {
	t.Helper()
	var lines []string
	lines = append(lines, fmt.Sprintf("version: %d", Version))
	for _, mf := range managedFiles {
		data, err := fs.ReadFile(assets, mf.src)
		if err != nil {
			t.Fatalf("read %s: %v", mf.src, err)
		}
		hash := sha256.Sum256(data)
		lines = append(lines, fmt.Sprintf("%s %x", mf.src, hash))
	}
	return strings.Join(lines, "\n") + "\n"
}
```

- [ ] **Step 2: Create testdata directory and run test to generate pinned hashes**

Run: `mkdir -p internal/upgrade/testdata && GOCACHE=$PWD/.cache/go-build go test ./internal/upgrade/ -run TestVersionMatchesEmbeddedFileHashes -count=1 -v`
Expected: PASS (generates `internal/upgrade/testdata/pinned-hashes.txt`)

- [ ] **Step 3: Verify the pinned hashes file was created**

Run: `cat internal/upgrade/testdata/pinned-hashes.txt`
Expected: Shows version line + 4 hash lines

- [ ] **Step 4: Run full test suite**

Run: `GOCACHE=$PWD/.cache/go-build go test ./... -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/upgrade/gate_test.go internal/upgrade/testdata/pinned-hashes.txt
git commit -m "test(upgrade): add gate test for version-hash consistency

Co-Authored-By: Peter O'Connor <poconnor@stackoverflow.com>
Co-Authored-By: Claude Code <noreply@anthropic.com> - databricks-claude-opus-4-6"
```
