package hookcfg

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"strings"
	"testing"

	"forge"
)

// canonicalFixture mirrors templates/common/claude/settings.json: forge's
// owned entries are exactly the entries present in the canonical template.
const canonicalFixture = `{
  "enabledPlugins": {},
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {"type": "command", "command": "./.claude/hooks/guard"}
        ]
      }
    ],
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {"type": "command", "command": "bd prime || true"},
          {"type": "command", "command": "command -v forge >/dev/null 2>&1 && forge upgrade --check || true"}
        ]
      }
    ]
  }
}
`

// existingWithThirdParty mirrors the real todo-cli file at its last good
// commit: bd appended its own matcher groups (duplicate "Bash" and ""
// matchers, machine-absolute paths) alongside forge's entries.
const existingWithThirdParty = `{
  "enabledPlugins": {},
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {"type": "command", "command": "./.claude/hooks/guard"}
        ]
      },
      {
        "matcher": "Bash",
        "hooks": [
          {"type": "command", "command": "/Users/avogadro/peter_code/todo-cli/.beads/hooks/agent-fitness-functions-git-guard"}
        ]
      },
      {
        "matcher": "Edit|Write",
        "hooks": [
          {"type": "command", "command": "/Users/avogadro/peter_code/todo-cli/.beads/hooks/agent-fitness-functions-pre-tool-use"}
        ]
      }
    ],
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {"type": "command", "command": "bd prime || true"},
          {"type": "command", "command": "command -v forge >/dev/null 2>&1 && forge upgrade --check || true"}
        ]
      },
      {
        "matcher": "",
        "hooks": [
          {"type": "command", "command": "bd prime --hook-json"}
        ]
      }
    ],
    "PreCompact": [
      {
        "matcher": "",
        "hooks": [
          {"type": "command", "command": "bd prime"}
        ]
      }
    ]
  }
}
`

func countOccurrences(t *testing.T, data []byte, needle string) int {
	t.Helper()
	return strings.Count(string(data), needle)
}

func TestReconcilePreservesThirdPartyHookEntries(t *testing.T) {
	t.Parallel()

	out, _, err := Reconcile([]byte(existingWithThirdParty), []byte(canonicalFixture), Registry{})
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	for _, thirdParty := range []string{
		"/Users/avogadro/peter_code/todo-cli/.beads/hooks/agent-fitness-functions-git-guard",
		"/Users/avogadro/peter_code/todo-cli/.beads/hooks/agent-fitness-functions-pre-tool-use",
		"bd prime --hook-json",
	} {
		if got := countOccurrences(t, out, thirdParty); got != 1 {
			t.Fatalf("third-party entry %q occurs %d times, want 1\n%s", thirdParty, got, out)
		}
	}
	// The whole PreCompact event exists only in the existing file — it must
	// survive verbatim.
	if got := countOccurrences(t, out, "PreCompact"); got != 1 {
		t.Fatalf("PreCompact event occurs %d times, want 1\n%s", got, out)
	}
	for _, owned := range []string{
		"./.claude/hooks/guard",
		"bd prime || true",
		"forge upgrade --check",
	} {
		if got := countOccurrences(t, out, owned); got != 1 {
			t.Fatalf("owned entry %q occurs %d times, want 1\n%s", owned, got, out)
		}
	}
}

func TestReconcileConvergesHistoricalForgeEntryForms(t *testing.T) {
	t.Parallel()

	// A v1-era repo shipped "bd prime" bare; the current canonical form is
	// "bd prime || true". The historical fingerprint claims the old entry so
	// it is replaced, not duplicated.
	existing := strings.Replace(existingWithThirdParty, `"bd prime || true"`, `"bd prime"`, 1)
	reg := Registry{Historical: map[string][]string{
		"bd prime || true": {"bd prime"},
	}}

	out, changed, err := Reconcile([]byte(existing), []byte(canonicalFixture), reg)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if !changed {
		t.Fatal("Reconcile() changed = false, want true (historical form must converge)")
	}
	if got := countOccurrences(t, out, `"bd prime || true"`); got != 1 {
		t.Fatalf("canonical form occurs %d times, want 1\n%s", got, out)
	}
	// The bare historical form must be gone from the SessionStart entry it
	// claimed. (It legitimately survives inside the third-party PreCompact
	// event, which forge does not own.)
	var doc struct {
		Hooks map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	for _, group := range doc.Hooks["SessionStart"] {
		for _, h := range group.Hooks {
			if h.Command == "bd prime" {
				t.Fatalf("stale historical form still present in SessionStart:\n%s", out)
			}
		}
	}
}

func TestReconcileAppendsMissingOwnedEntries(t *testing.T) {
	t.Parallel()

	// A repo where the upgrade --check entry was lost entirely.
	existing := strings.Replace(existingWithThirdParty,
		`,
          {"type": "command", "command": "command -v forge >/dev/null 2>&1 && forge upgrade --check || true"}`, "", 1)
	if strings.Contains(existing, "forge upgrade --check") {
		t.Fatal("fixture setup failed: owned entry still present")
	}

	out, changed, err := Reconcile([]byte(existing), []byte(canonicalFixture), Registry{})
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if !changed {
		t.Fatal("Reconcile() changed = false, want true (missing owned entry must be appended)")
	}
	if got := countOccurrences(t, out, "forge upgrade --check"); got != 1 {
		t.Fatalf("restored owned entry occurs %d times, want 1\n%s", got, out)
	}
}

func TestReconcileIsAFixedPoint(t *testing.T) {
	t.Parallel()

	first, _, err := Reconcile([]byte(existingWithThirdParty), []byte(canonicalFixture), Registry{})
	if err != nil {
		t.Fatalf("Reconcile(1) error = %v", err)
	}
	second, changed, err := Reconcile(first, []byte(canonicalFixture), Registry{})
	if err != nil {
		t.Fatalf("Reconcile(2) error = %v", err)
	}
	if changed {
		t.Fatal("Reconcile(2) changed = true, want false (fixed point)")
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("second pass not byte-identical:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if !bytes.HasSuffix(first, []byte("\n")) {
		t.Fatal("output must end with a trailing newline")
	}
}

func TestReconcilePreservesUnknownTopLevelKeys(t *testing.T) {
	t.Parallel()

	existing := strings.Replace(existingWithThirdParty,
		`"enabledPlugins": {},`,
		`"enabledPlugins": {"acme@1": true}, "permissions": {"allow": ["Bash(ls:*)"]},`, 1)

	out, _, err := Reconcile([]byte(existing), []byte(canonicalFixture), Registry{})
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if !strings.Contains(string(out), `"acme@1"`) {
		t.Fatalf("populated enabledPlugins lost:\n%s", out)
	}
	if !strings.Contains(string(out), `"Bash(ls:*)"`) {
		t.Fatalf("unknown top-level permissions key lost:\n%s", out)
	}
}

func TestReconcileFailsLoudOnMalformedInput(t *testing.T) {
	t.Parallel()

	out, changed, err := Reconcile([]byte("{not json"), []byte(canonicalFixture), Registry{})
	if err == nil {
		t.Fatal("Reconcile() error = nil for malformed input, want failure")
	}
	if out != nil || changed {
		t.Fatalf("Reconcile() must write nothing on malformed input: out=%q changed=%v", out, changed)
	}
}

func TestReconcileAcceptsRealEmbeddedTemplates(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"templates/common/claude/settings.json",
		"templates/common/codex/hooks.json",
	} {
		canonical, err := fs.ReadFile(forge.Assets(), path)
		if err != nil {
			t.Fatalf("read embedded %s: %v", path, err)
		}

		out, _, err := Reconcile([]byte(existingWithThirdParty), canonical, Registry{})
		if err != nil {
			t.Fatalf("Reconcile with real %s failed: %v", path, err)
		}
		if !json.Valid(out) {
			t.Fatalf("output for %s is not valid JSON:\n%s", path, out)
		}
		if got := countOccurrences(t, out, "bd prime --hook-json"); got != 1 {
			t.Fatalf("%s: third-party entry lost (occurs %d times)\n%s", path, got, out)
		}
	}
}
