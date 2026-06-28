package delegate

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestVerboseRunnerPrintsStepSummaryOnSuccess(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	runner := &VerboseRunner{out: &buf, isTTY: false}

	err := runner.Run(context.Background(), t.TempDir(), "echo test", "echo", "hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "→ echo test") {
		t.Fatalf("output missing step header, got: %q", output)
	}
	if !strings.Contains(output, "→ echo test ✓") {
		t.Fatalf("output missing success summary, got: %q", output)
	}
	if !strings.Contains(output, "  hello\n") {
		t.Fatalf("output missing indented command output, got: %q", output)
	}
}

func TestVerboseRunnerPrintsFailureSummaryAndKeepsOutput(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("test uses sh -c")
	}

	var buf bytes.Buffer
	runner := &VerboseRunner{out: &buf, isTTY: false}

	err := runner.Run(context.Background(), t.TempDir(), "failing cmd", "sh", "-c", "echo oops && exit 1")
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}

	output := buf.String()
	if !strings.Contains(output, "  oops\n") {
		t.Fatalf("output missing command output on failure, got: %q", output)
	}
	if !strings.Contains(output, "→ failing cmd ✗") {
		t.Fatalf("output missing failure summary, got: %q", output)
	}
}

func TestVerboseRunnerEmitsANSIEraseInTTYMode(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	runner := &VerboseRunner{out: &buf, isTTY: true}

	err := runner.Run(context.Background(), t.TempDir(), "echo test", "echo", "hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\033[") {
		t.Fatalf("output missing ANSI escape sequence in TTY mode, got: %q", output)
	}
	if !strings.Contains(output, "→ echo test ✓") {
		t.Fatalf("output missing success summary after ANSI erase, got: %q", output)
	}
}

func TestVerboseRunnerNoANSIInNonTTYMode(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	runner := &VerboseRunner{out: &buf, isTTY: false}

	err := runner.Run(context.Background(), t.TempDir(), "echo test", "echo", "hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "\033[") {
		t.Fatalf("output contains ANSI escape in non-TTY mode, got: %q", output)
	}
}

func TestVerboseRunnerTTYEraseWithNoOutput(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	runner := &VerboseRunner{out: &buf, isTTY: true}

	err := runner.Run(context.Background(), t.TempDir(), "silent step", "true")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\033[1A\033[J") {
		t.Fatalf("expected single-line erase for step with no output, got: %q", output)
	}
	if !strings.Contains(output, "→ silent step ✓") {
		t.Fatalf("output missing success summary, got: %q", output)
	}
}

func TestVerboseRunnerUsesFallbackExecutableWhenPATHMissesCommand(t *testing.T) {
	fallbackRoot := t.TempDir()
	fallbackMise := filepath.Join(fallbackRoot, "mise")
	if err := os.WriteFile(fallbackMise, []byte("#!/bin/sh\necho fallback-mise\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", fallbackMise, err)
	}

	previous := fallbackExecutableCandidates
	fallbackExecutableCandidates = map[string][]string{
		"mise": {fallbackMise},
	}
	t.Cleanup(func() {
		fallbackExecutableCandidates = previous
	})

	t.Setenv("PATH", "/usr/bin")

	var buf bytes.Buffer
	runner := &VerboseRunner{out: &buf, isTTY: false}

	if err := runner.Run(context.Background(), t.TempDir(), "mise version", "mise"); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "fallback-mise") {
		t.Fatalf("output missing fallback executable output, got: %q", output)
	}
}
