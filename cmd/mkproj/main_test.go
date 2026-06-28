package main

import (
	"io"
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
	original := "{\n  \"permissions\": {\n    \"allow\": [\n      \"// BEGIN MKPROJ ALLOW v:0\",\n      \"Bash(go:*)\",\n      \"// END MKPROJ ALLOW\",\n      \"Bash(true)\"\n    ]\n  }\n}\n"
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

func TestRunSyncAllowlistCheckNotifiesWithoutMutatingSettings(t *testing.T) {
	tempDir := t.TempDir()
	settingsPath := filepath.Join(tempDir, "settings.local.json")
	original := "{\n  \"permissions\": {\n    \"allow\": [\n      \"// BEGIN MKPROJ ALLOW v:0\",\n      \"Bash(go:*)\",\n      \"// END MKPROJ ALLOW\",\n      \"Bash(true)\"\n    ]\n  }\n}\n"
	if err := os.WriteFile(settingsPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	output, err := captureStdout(t, func() error {
		return runSyncAllowlist([]string{"--check", "--path", settingsPath}, mkproj.Assets())
	})
	if err != nil {
		t.Fatalf("runSyncAllowlist(--check) error = %v", err)
	}
	if !strings.Contains(output, "allowlist is 1 version(s) behind; run mkproj sync-allowlist") {
		t.Fatalf("runSyncAllowlist(--check) output = %q, want stale notice", output)
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != original {
		t.Fatalf("runSyncAllowlist(--check) mutated settings.local.json:\n%s", string(data))
	}
}

func TestRunSyncAllowlistIncludesPersonalRulesOnlyWhenRequested(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	settingsPath := filepath.Join(tempDir, "settings.local.json")
	original := "{\n  \"permissions\": {\n    \"allow\": [\n      \"// BEGIN MKPROJ ALLOW v:0\",\n      \"Bash(go:*)\",\n      \"// END MKPROJ ALLOW\",\n      \"Bash(true)\"\n    ]\n  }\n}\n"
	if err := os.WriteFile(settingsPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := runSyncAllowlist([]string{"--path", settingsPath}, mkproj.Assets()); err != nil {
		t.Fatalf("runSyncAllowlist(default) error = %v", err)
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(default) error = %v", err)
	}
	if strings.Contains(string(data), `"Bash(gw:*)",`) {
		t.Fatalf("default sync unexpectedly included personal rules:\n%s", string(data))
	}

	if err := runSyncAllowlist([]string{"--include-personal", "--path", settingsPath}, mkproj.Assets()); err != nil {
		t.Fatalf("runSyncAllowlist(--include-personal) error = %v", err)
	}
	data, err = os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(include personal) error = %v", err)
	}
	for _, snippet := range []string{`"Bash(gw:*)",`, `"Bash(slack-cli:*)",`} {
		if !strings.Contains(string(data), snippet) {
			t.Fatalf("sync with personal rules missing %q in:\n%s", snippet, string(data))
		}
	}
}

func TestRunSyncAllowlistRejectsConflictingManagedBlockLanguageMarkers(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	settingsPath := filepath.Join(tempDir, "settings.local.json")
	original := "{\n  \"permissions\": {\n    \"allow\": [\n      \"// BEGIN MKPROJ ALLOW v:0\",\n      \"Bash(go:*)\",\n      \"Bash(python:*)\",\n      \"// END MKPROJ ALLOW\",\n      \"Bash(true)\"\n    ]\n  }\n}\n"
	if err := os.WriteFile(settingsPath, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := runSyncAllowlist([]string{"--path", settingsPath}, mkproj.Assets())
	if err == nil || !strings.Contains(err.Error(), "conflicting language markers") {
		t.Fatalf("runSyncAllowlist() error = %v, want conflicting language markers", err)
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

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	runErr := fn()
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("reader.Close() error = %v", err)
	}

	return string(output), runErr
}
