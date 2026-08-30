package upgrade

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureBeadsDirPermsTightensTo0700(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".beads")
	if err := os.Mkdir(beadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// bd init runs under the parent umask and leaves .beads world-readable;
	// bd itself wants 0700.
	if err := os.Chmod(beadsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	repaired, err := EnsureBeadsDirPerms(dir)
	if err != nil {
		t.Fatalf("EnsureBeadsDirPerms() error = %v", err)
	}
	if !repaired {
		t.Fatal("EnsureBeadsDirPerms() repaired = false, want true for 0755 dir")
	}

	info, err := os.Stat(beadsDir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf(".beads mode = %o, want 700", info.Mode().Perm())
	}
}

func TestEnsureBeadsDirPermsIsIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	beadsDir := filepath.Join(dir, ".beads")
	if err := os.Mkdir(beadsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(beadsDir, 0o700); err != nil {
		t.Fatal(err)
	}

	repaired, err := EnsureBeadsDirPerms(dir)
	if err != nil {
		t.Fatalf("EnsureBeadsDirPerms() error = %v", err)
	}
	if repaired {
		t.Fatal("EnsureBeadsDirPerms() repaired = true for already-0700 dir, want false")
	}

	info, err := os.Stat(beadsDir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf(".beads mode = %o, want 700", info.Mode().Perm())
	}
}

func TestEnsureBeadsDirPermsNoOpWhenAbsent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	repaired, err := EnsureBeadsDirPerms(dir)
	if err != nil {
		t.Fatalf("EnsureBeadsDirPerms() error = %v, want nil when .beads is absent", err)
	}
	if repaired {
		t.Fatal("EnsureBeadsDirPerms() repaired = true when .beads is absent, want false")
	}
}
