package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakefileDefinesCoreTargets(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(repoRoot, "Makefile"))
	if err != nil {
		t.Fatalf("ReadFile(Makefile) error = %v", err)
	}

	text := string(data)
	for _, snippet := range []string{
		".DEFAULT_GOAL := help",
		".PHONY: help build test clean",
		"help: ## Show available targets",
		"build: ## Build the mkproj binary into bin/",
		"\tgo build -o $(BIN_PATH) ./cmd/mkproj",
		"test: ## Run the full Go test suite",
		"\tGOCACHE=$(PWD)/.cache/go-build go test ./... -count=1",
		"clean: ## Remove local build outputs",
		"\trm -rf $(BIN_DIR)",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("Makefile missing %q\n%s", snippet, text)
		}
	}
}

func TestMakeDefaultTargetShowsHelp(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	output, err := runMake(t, repoRoot)
	if err != nil {
		t.Fatalf("make error = %v\n%s", err, output)
	}

	for _, snippet := range []string{"help", "build", "test", "clean"} {
		if !strings.Contains(output, snippet) {
			t.Fatalf("make output missing %q\n%s", snippet, output)
		}
	}
}

func TestMakeBuildProducesMkprojBinary(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	if output, err := runMake(t, repoRoot, "clean"); err != nil {
		t.Fatalf("make clean error = %v\n%s", err, output)
	}
	if output, err := runMake(t, repoRoot, "build"); err != nil {
		t.Fatalf("make build error = %v\n%s", err, output)
	}
	defer func() {
		if output, err := runMake(t, repoRoot, "clean"); err != nil {
			t.Fatalf("deferred make clean error = %v\n%s", err, output)
		}
	}()

	info, err := os.Stat(filepath.Join(repoRoot, "bin", "mkproj"))
	if err != nil {
		t.Fatalf("Stat(bin/mkproj) error = %v", err)
	}
	if info.IsDir() {
		t.Fatal("bin/mkproj is a directory, want file")
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("bin/mkproj mode = %v, want executable bit", info.Mode())
	}
}

func runMake(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()

	if _, err := exec.LookPath("make"); err != nil {
		t.Skipf("make not available: %v", err)
	}

	cmd := exec.Command("make", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return string(output), err
}
