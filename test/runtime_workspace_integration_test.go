//go:build integration

package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTempRuntimeDirUsesWorkspaceRuntimeRootWhenConfigured(t *testing.T) {
	repoRoot := t.TempDir()
	previousRuntimeRoot := runtimeWorkspaceRoot
	runtimeWorkspaceRoot = repoRoot
	t.Cleanup(func() {
		runtimeWorkspaceRoot = previousRuntimeRoot
	})

	dir := tempRuntimeDir(t)

	expectedPrefix := filepath.Join(repoRoot, ".cache", "local-release") + string(filepath.Separator)
	if !strings.HasPrefix(dir, expectedPrefix) {
		t.Fatalf("tempRuntimeDir() = %q, want prefix %q", dir, expectedPrefix)
	}

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("Stat(%s) error = %v", dir, err)
	}
}
