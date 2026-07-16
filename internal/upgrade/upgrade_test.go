package upgrade

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func testAssets() fstest.MapFS {
	return fstest.MapFS{
		"templates/common/claude/hooks/guard":          &fstest.MapFile{Data: []byte("#!/usr/bin/env bash\n# guard v1\n")},
		"templates/common/claude/hooks/secret-scan.sh": &fstest.MapFile{Data: []byte("#!/usr/bin/env bash\n# secret-scan v1\n")},
		"templates/common/claude/settings.json":        &fstest.MapFile{Data: []byte(`{"hooks":{}}` + "\n")},
		"templates/common/codex/hooks.json":            &fstest.MapFile{Data: []byte(`{"hooks":{}}` + "\n")},
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
