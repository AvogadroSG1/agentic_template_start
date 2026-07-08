package allowlist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forge"
)

func TestDetectReportsStaleManagedBlocks(t *testing.T) {
	t.Parallel()

	status, err := Detect(`"// BEGIN FORGE ALLOW v:0",` + "\n" + `"Bash(old:*)",` + "\n" + `"// END FORGE ALLOW",` + "\n")
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !status.Stale {
		t.Fatalf("Detect() stale = false, want true")
	}
}

func TestDetectRejectsMalformedManagedBlockVersions(t *testing.T) {
	t.Parallel()

	_, err := Detect(`"// BEGIN FORGE ALLOW v:nope",` + "\n" + `"// END FORGE ALLOW",` + "\n")
	if err == nil || !strings.Contains(err.Error(), "invalid managed block version") {
		t.Fatalf("Detect() error = %v, want invalid managed block version", err)
	}
}

func TestSyncRewritesOnlyTheManagedBlock(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "settings.local.json")
	original := `{
  "permissions": {
    "allow": [
      "// BEGIN FORGE ALLOW v:0",
      "Bash(old:*)",
      "// END FORGE ALLOW",
      "Bash(true)"
    ]
  },
  "after": true
}
`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	status, err := Sync(path, `      "Bash(new:*)",`, false)
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
	if !strings.Contains(text, `"after": true`) {
		t.Fatalf("Sync() rewrote surrounding content:\n%s", text)
	}
	if !strings.Contains(text, `"Bash(new:*)"`) {
		t.Fatalf("Sync() missing replacement block:\n%s", text)
	}
	if strings.Contains(text, `"Bash(old:*)"`) {
		t.Fatalf("Sync() did not remove old block:\n%s", text)
	}
}

func TestInferLanguageUsesOnlyTheManagedBlock(t *testing.T) {
	t.Parallel()

	contents := `{
  "note": "Bash(python:*) belongs in docs only",
  "permissions": {
    "allow": [
      "// BEGIN FORGE ALLOW v:1",
      "Bash(go:*)",
      "// END FORGE ALLOW",
      "Bash(true)"
    ]
  }
}
`

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

	contents := `{
  "permissions": {
    "allow": [
      "// BEGIN FORGE ALLOW v:1",
      "Bash(git status:*)",
      "// END FORGE ALLOW",
      "Bash(true)"
    ]
  }
}
`

	_, err := InferLanguage(contents)
	if err == nil || !strings.Contains(err.Error(), "could not infer project language") {
		t.Fatalf("InferLanguage() error = %v, want missing language marker", err)
	}
}

func TestCanonicalBlockKeepsPersonalRulesOptIn(t *testing.T) {
	t.Parallel()

	defaultBlock, err := CanonicalBlock(forge.Assets(), "go", false)
	if err != nil {
		t.Fatalf("CanonicalBlock(default) error = %v", err)
	}
	if strings.Contains(defaultBlock, `"Bash(gw:*)"`) {
		t.Fatalf("CanonicalBlock(default) unexpectedly included personal rules:\n%s", defaultBlock)
	}

	personalBlock, err := CanonicalBlock(forge.Assets(), "go", true)
	if err != nil {
		t.Fatalf("CanonicalBlock(include personal) error = %v", err)
	}
	for _, snippet := range []string{
		`"Bash(gw:*)",`,
		`"Bash(slack-cli:*)",`,
		`"Bash(docker images:*)",`,
	} {
		if !strings.Contains(personalBlock, snippet) {
			t.Fatalf("CanonicalBlock(include personal) missing %q in:\n%s", snippet, personalBlock)
		}
	}
}
