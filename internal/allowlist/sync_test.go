package allowlist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
