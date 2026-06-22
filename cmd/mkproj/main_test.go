package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mkproj"
)

func TestSelectCommandDefaultsToInit(t *testing.T) {
	t.Parallel()

	command, remaining := selectCommand(nil)
	if command != "init" {
		t.Fatalf("command = %q, want init", command)
	}
	if len(remaining) != 0 {
		t.Fatalf("remaining args = %#v, want none", remaining)
	}
}

func TestSelectCommandRecognizesExplicitSubcommands(t *testing.T) {
	t.Parallel()

	command, remaining := selectCommand([]string{"sync-allowlist", "--check"})
	if command != "sync-allowlist" {
		t.Fatalf("command = %q, want sync-allowlist", command)
	}
	if len(remaining) != 1 || remaining[0] != "--check" {
		t.Fatalf("remaining args = %#v, want [--check]", remaining)
	}
}

func TestSyncAllowlistUsesCanonicalEmbeddedEntriesForTheProjectLanguage(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	settingsPath := filepath.Join(tempDir, "settings.local.json")
	original := "{\n  \"permissions\": {\n    \"allow\": [\n      // BEGIN MKPROJ ALLOW v:0\n      \"Bash(go:*)\",\n      // END MKPROJ ALLOW\n      \"Bash(true)\"\n    ]\n  }\n}\n"
	if err := os.WriteFile(settingsPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := runSyncAllowlist([]string{"--path", settingsPath}, mkproj.Assets()); err != nil {
		t.Fatalf("runSyncAllowlist() error = %v", err)
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	block := string(data)
	for _, snippet := range []string{`"Bash(git status:*)",`, `"Bash(instill:*)",`, `"Bash(lefthook:*)",`, `"Bash(go:*)",`} {
		if !strings.Contains(block, snippet) {
			t.Fatalf("synced allowlist missing %q in %s", snippet, block)
		}
	}
	if strings.Contains(block, `"Bash(python:*)",`) {
		t.Fatalf("synced allowlist should not inject python rules into a go project:\n%s", block)
	}
}

func TestSelectCommandRecognizesUpdate(t *testing.T) {
	t.Parallel()

	command, remaining := selectCommand([]string{"update", "--stack", "go-cli-cobra"})
	if command != "update" {
		t.Fatalf("command = %q, want update", command)
	}
	if got := strings.Join(remaining, " "); got != "--stack go-cli-cobra" {
		t.Fatalf("remaining args = %q, want %q", got, "--stack go-cli-cobra")
	}
}

func TestRunUpdateRequiresStackFlag(t *testing.T) {
	t.Parallel()

	err := runUpdate(nil, mkproj.Assets())
	if err == nil {
		t.Fatal("runUpdate() error = nil, want missing stack flag")
	}
	if !strings.Contains(err.Error(), "missing required flag: --stack") {
		t.Fatalf("runUpdate() error = %q, want missing stack flag", err)
	}
}
