package test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWorkspaceRuntimeDirCreatesCacheScopedDirectory(t *testing.T) {
	repoRoot := t.TempDir()

	dir, err := newWorkspaceRuntimeDir(repoRoot, "mkproj-runtime")
	if err != nil {
		t.Fatalf("newWorkspaceRuntimeDir returned error: %v", err)
	}

	expectedPrefix := filepath.Join(repoRoot, ".cache", "local-release") + string(filepath.Separator)
	if !strings.HasPrefix(dir, expectedPrefix) {
		t.Fatalf("runtime dir = %q, want prefix %q", dir, expectedPrefix)
	}
}
