package test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var vendoredGuidelineFiles = []string{
	"golang.md", "python.md", "csharp.md", "typescript.md", "rust.md", "bash.md",
}

// Contract 1: conformance inputs are vendored in-repo so `go test ./...`
// passes on any machine, including CI runners without the maintainer's
// home directory layout.
func TestVendoredGuidelineSnapshotsExistInRepo(t *testing.T) {
	root := repoRoot(t)
	for _, name := range vendoredGuidelineFiles {
		path := filepath.Join(root, "test", "testdata", "guidelines", name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("vendored guideline snapshot missing: %s", path)
		}
		if info.Size() < 100 {
			t.Fatalf("vendored guideline snapshot %s is suspiciously small (%d bytes)", path, info.Size())
		}
	}
}

// Contract 2: the conformance checker resolves guideline files inside the
// repository, never under the user's home directory.
func TestGuidelineCheckerResolvesInsideRepo(t *testing.T) {
	checker := newGuidelineChecker(t)
	root := repoRoot(t)

	for _, language := range []string{"golang", "python", "csharp", "typescript"} {
		path, err := checker.guidelinePath(language)
		if err != nil {
			t.Fatalf("guideline path for %s: %v", language, err)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			t.Fatalf("%s guideline resolves outside the repo: %s", language, path)
		}
	}
}

// Contract 3: on machines that hold the canonical guideline source, the
// vendored snapshots must match it byte-for-byte; elsewhere the check skips.
func TestVendoredGuidelinesMatchCanonicalSource(t *testing.T) {
	canonicalDir := os.Getenv("FORGE_CANONICAL_GUIDELINES_DIR")
	if canonicalDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("no home dir: %v", err)
		}
		canonicalDir = filepath.Join(home, "peter_code", "ai_support", "guidelines")
	}
	if _, err := os.Stat(canonicalDir); err != nil {
		t.Skipf("canonical guideline source not present: %s", canonicalDir)
	}

	root := repoRoot(t)
	for _, name := range vendoredGuidelineFiles {
		canonicalPath := filepath.Join(canonicalDir, name)
		vendoredPath := filepath.Join(root, "test", "testdata", "guidelines", name)

		canonical, err := os.ReadFile(canonicalPath)
		if err != nil {
			t.Fatalf("read canonical %s: %v", name, err)
		}
		vendored, err := os.ReadFile(vendoredPath)
		if err != nil {
			t.Fatalf("read vendored %s: %v", name, err)
		}
		if !bytes.Equal(canonical, vendored) {
			t.Fatalf("vendored guideline %s drifted from canonical source; refresh with: cp -f %s %s", name, canonicalPath, vendoredPath)
		}
	}
}
