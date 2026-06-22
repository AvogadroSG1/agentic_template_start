package main

import (
	"strings"
	"testing"
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

func TestDefaultAllowlistBlockProducesJSONCEntries(t *testing.T) {
	t.Parallel()

	block := defaultAllowlistBlock()
	for _, snippet := range []string{`"Bash(git:*)",`, `"Bash(instill:*)",`, `"Bash(lefthook:*)",`} {
		if !strings.Contains(block, snippet) {
			t.Fatalf("defaultAllowlistBlock() missing %q in %q", snippet, block)
		}
	}
}
