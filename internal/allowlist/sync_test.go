package allowlist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mkproj"
)

func TestDetectReportsStaleManagedBlocks(t *testing.T) {
	t.Parallel()

	status, err := Detect("// BEGIN MKPROJ ALLOW v:0\nold\n// END MKPROJ ALLOW\n")
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !status.Stale {
		t.Fatalf("Detect() stale = false, want true")
	}
}

func TestDetectRejectsMalformedManagedBlockVersions(t *testing.T) {
	t.Parallel()

	_, err := Detect("// BEGIN MKPROJ ALLOW v:nope\nold\n// END MKPROJ ALLOW\n")
	if err == nil || !strings.Contains(err.Error(), "invalid managed block version") {
		t.Fatalf("Detect() error = %v, want invalid managed block version", err)
	}
}

func TestSyncRewritesOnlyTheManagedBlock(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "settings.local.json")
	original := "{\n  \"before\": true,\n  // BEGIN MKPROJ ALLOW v:0\n  old rules\n  // END MKPROJ ALLOW\n  \"after\": true\n}\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	status, err := Sync(path, "  new rules", false)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if status.CurrentVersion != Version {
		t.Fatalf("Sync() version = %d, want %d", status.CurrentVersion, Version)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "\"before\": true") || !strings.Contains(text, "\"after\": true") {
		t.Fatalf("Sync() rewrote surrounding content:\n%s", text)
	}
	if !strings.Contains(text, "new rules") {
		t.Fatalf("Sync() missing replacement block:\n%s", text)
	}
}

func TestInferLanguageUsesOnlyTheManagedBlock(t *testing.T) {
	t.Parallel()

	contents := "{\n  \"note\": \"Bash(python:*) belongs in docs only\",\n  \"permissions\": {\n    \"allow\": [\n      // BEGIN MKPROJ ALLOW v:1\n      \"Bash(go:*)\",\n      // END MKPROJ ALLOW\n      \"Bash(true)\"\n    ]\n  }\n}\n"

	language, err := InferLanguage(contents)
	if err != nil {
		t.Fatalf("InferLanguage() error = %v", err)
	}
	if language != "go" {
		t.Fatalf("InferLanguage() = %q, want go", language)
	}
}

func TestInferLanguageRejectsMissingManagedBlockLanguageMarkers(t *testing.T) {
	t.Parallel()

	contents := "{\n  \"permissions\": {\n    \"allow\": [\n      // BEGIN MKPROJ ALLOW v:1\n      \"Bash(git status:*)\",\n      // END MKPROJ ALLOW\n      \"Bash(true)\"\n    ]\n  }\n}\n"

	_, err := InferLanguage(contents)
	if err == nil || !strings.Contains(err.Error(), "could not infer project language") {
		t.Fatalf("InferLanguage() error = %v, want missing language marker", err)
	}
}

func TestCanonicalBlockKeepsPersonalRulesOptIn(t *testing.T) {
	t.Parallel()

	defaultBlock, err := CanonicalBlock(mkproj.Assets(), "go", false)
	if err != nil {
		t.Fatalf("CanonicalBlock(default) error = %v", err)
	}
	if strings.Contains(defaultBlock, `"Bash(gw:*)"`) {
		t.Fatalf("CanonicalBlock(default) unexpectedly included personal rules:\n%s", defaultBlock)
	}

	personalBlock, err := CanonicalBlock(mkproj.Assets(), "go", true)
	if err != nil {
		t.Fatalf("CanonicalBlock(include personal) error = %v", err)
	}
	for _, snippet := range []string{
		`// BEGIN MKPROJ PERSONAL`,
		`"Bash(gw:*)",`,
		`"Bash(slack-cli:*)",`,
		`"Bash(docker images:*)",`,
		`// END MKPROJ PERSONAL`,
	} {
		if !strings.Contains(personalBlock, snippet) {
			t.Fatalf("CanonicalBlock(include personal) missing %q in:\n%s", snippet, personalBlock)
		}
	}
}
