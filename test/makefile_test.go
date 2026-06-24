package test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var makefileTestMu sync.Mutex

func TestMakefileDefinesCoreTargets(t *testing.T) {
	t.Parallel()

	lockMakefileTest(t)

	repoRoot := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(repoRoot, "Makefile"))
	if err != nil {
		t.Fatalf("ReadFile(Makefile) error = %v", err)
	}

	text := string(data)
	for _, snippet := range []string{
		".DEFAULT_GOAL := help",
		".PHONY: help build test install uninstall clean",
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

	lockMakefileTest(t)

	repoRoot := repoRoot(t)
	output, err := runMake(t, repoRoot)
	if err != nil {
		t.Fatalf("make error = %v\n%s", err, output)
	}

	for _, snippet := range []string{"help", "build", "test", "install", "uninstall", "clean"} {
		if !strings.Contains(output, snippet) {
			t.Fatalf("make output missing %q\n%s", snippet, output)
		}
	}
}

func TestMakeBuildProducesMkprojBinary(t *testing.T) {
	t.Parallel()

	lockMakefileTest(t)

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

func TestMakefileDefinesInstallLifecycleTargets(t *testing.T) {
	t.Parallel()

	lockMakefileTest(t)

	repoRoot := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(repoRoot, "Makefile"))
	if err != nil {
		t.Fatalf("ReadFile(Makefile) error = %v", err)
	}

	text := string(data)
	for _, snippet := range []string{
		"BINDIR ?= $(HOME)/.local/bin",
		".PHONY: help build test install uninstall clean",
		"install: build ## Install mkproj into BINDIR",
		"\t@mkdir -p $(BINDIR)",
		"\tinstall -m 0755 $(BIN_PATH) $(BINDIR)/mkproj",
		"uninstall: ## Remove installed mkproj from BINDIR",
		"\trm -f $(BINDIR)/mkproj",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("Makefile missing %q\n%s", snippet, text)
		}
	}
}

func TestMakeInstallAndUninstallRoundTrip(t *testing.T) {
	t.Parallel()

	lockMakefileTest(t)

	repoRoot := repoRoot(t)
	bindir := filepath.Join(repoRoot, ".cache", "mkproj-bin")
	installedBinary := filepath.Join(bindir, "mkproj")

	if output, err := runMake(t, repoRoot, "clean"); err != nil {
		t.Fatalf("make clean error = %v\n%s", err, output)
	}
	defer func() {
		if output, err := runMake(t, repoRoot, "clean"); err != nil {
			t.Fatalf("deferred make clean error = %v\n%s", err, output)
		}
	}()

	if output, err := runMake(t, repoRoot, "install", "BINDIR="+bindir); err != nil {
		t.Fatalf("make install error = %v\n%s", err, output)
	}

	info, err := os.Stat(installedBinary)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", installedBinary, err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("%s mode = %v, want executable bit", installedBinary, info.Mode())
	}

	cmd := exec.Command(installedBinary, "update")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("%s update error = nil, want missing flag\n%s", installedBinary, output)
	}
	if !strings.Contains(string(output), "missing required flag: --stack") {
		t.Fatalf("%s update output = %q, want missing stack flag", installedBinary, string(output))
	}

	if output, err := runMake(t, repoRoot, "uninstall", "BINDIR="+bindir); err != nil {
		t.Fatalf("make uninstall error = %v\n%s", err, output)
	}
	if _, err := os.Stat(installedBinary); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(%s) error = %v, want not exists", installedBinary, err)
	}
}

func TestGitIgnoreIgnoresBinDirectory(t *testing.T) {
	t.Parallel()

	lockMakefileTest(t)

	repoRoot := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile(.gitignore) error = %v", err)
	}

	text := string(data)
	if !strings.Contains(text, "\nbin/\n") && !strings.HasPrefix(text, "bin/\n") {
		t.Fatalf(".gitignore missing bin/ entry\n%s", text)
	}
}

func TestReadmeDocumentsMakeWorkflow(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(repoRoot, "README.md"))
	if err != nil {
		t.Fatalf("ReadFile(README.md) error = %v", err)
	}

	text := string(data)
	for _, snippet := range []string{
		"make help",
		"make install",
		"make install BINDIR=/custom/bin",
		"make build",
		"make test",
		"make clean",
		"make uninstall",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("README.md missing %q\n%s", snippet, text)
		}
	}

	for _, snippet := range []string{
		"go build ./cmd/mkproj",
		"go build -o $(BIN_PATH) ./cmd/mkproj",
	} {
		if strings.Contains(text, snippet) {
			t.Fatalf("README.md still contains stale guidance %q\n%s", snippet, text)
		}
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

func lockMakefileTest(t *testing.T) {
	t.Helper()

	makefileTestMu.Lock()
	t.Cleanup(makefileTestMu.Unlock)
}
