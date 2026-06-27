package test

import (
	"os"
	"path/filepath"
)

func newWorkspaceRuntimeDir(repoRoot string, prefix string) (string, error) {
	baseDir := filepath.Join(repoRoot, ".cache", "local-release")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", err
	}

	return os.MkdirTemp(baseDir, prefix+"-*")
}
