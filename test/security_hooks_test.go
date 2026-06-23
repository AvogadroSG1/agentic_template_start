package test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHookConfigsWireGuardAndNonBlockingSessionStart(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	checks := []struct {
		path     string
		snippets []string
	}{
		{
			path: filepath.Join(repoRoot, ".claude", "settings.json"),
			snippets: []string{
				`"PreToolUse"`,
				`"command": "./.claude/hooks/guard"`,
				`"command": "bd prime || true"`,
				`"command": "instill check-skills || true"`,
				`"command": "command -v mkproj >/dev/null 2>&1 && mkproj sync-allowlist --check || true"`,
			},
		},
		{
			path: filepath.Join(repoRoot, ".codex", "hooks.json"),
			snippets: []string{
				`"PreToolUse"`,
				`"command": "./.claude/hooks/guard"`,
				`"command": "bd prime || true"`,
				`"command": "instill check-skills || true"`,
				`"command": "command -v mkproj >/dev/null 2>&1 && mkproj sync-allowlist --check || true"`,
			},
		},
		{
			path: filepath.Join(repoRoot, "templates", "common", "claude", "settings.json"),
			snippets: []string{
				`"PreToolUse"`,
				`"command": "./.claude/hooks/guard"`,
				`"command": "bd prime || true"`,
				`"command": "instill check-skills || true"`,
				`"command": "command -v mkproj >/dev/null 2>&1 && mkproj sync-allowlist --check || true"`,
			},
		},
		{
			path: filepath.Join(repoRoot, "templates", "common", "codex", "hooks.json"),
			snippets: []string{
				`"PreToolUse"`,
				`"command": "./.claude/hooks/guard"`,
				`"command": "bd prime || true"`,
				`"command": "instill check-skills || true"`,
				`"command": "command -v mkproj >/dev/null 2>&1 && mkproj sync-allowlist --check || true"`,
			},
		},
	}

	for _, check := range checks {
		check := check
		t.Run(filepath.Base(check.path), func(t *testing.T) {
			data, err := os.ReadFile(check.path)
			if err != nil {
				t.Fatalf("ReadFile(%s) error = %v", check.path, err)
			}

			text := string(data)
			for _, snippet := range check.snippets {
				if !strings.Contains(text, snippet) {
					t.Fatalf("%s missing %q\n%s", check.path, snippet, text)
				}
			}
		})
	}
}

func TestSharedGuardMatchesTemplate(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	guardPath := filepath.Join(repoRoot, ".claude", "hooks", "guard")
	templatePath := filepath.Join(repoRoot, "templates", "common", "claude", "hooks", "guard")

	guard, err := os.ReadFile(guardPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", guardPath, err)
	}
	template, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", templatePath, err)
	}
	if string(guard) != string(template) {
		t.Fatalf("template guard drifted from repo-root guard")
	}
}

func TestSharedGuardBlocksDenyFloorAndAllowsSafeCompound(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	guardPath := filepath.Join(repoRoot, ".claude", "hooks", "guard")
	if _, err := os.Stat(guardPath); err != nil {
		t.Fatalf("Stat(%s) error = %v", guardPath, err)
	}

	tests := []struct {
		name        string
		commandLine string
		wantStatus  int
		wantOutput  string
	}{
		{name: "delegates secret path scan", commandLine: concat("cat ", ".env"), wantStatus: 2, wantOutput: "BLOCKED [D9]"},
		{name: "blocks recursive delete", commandLine: concat("rm -", "rf tmp"), wantStatus: 2, wantOutput: "BLOCKED [D1]"},
		{name: "blocks git rm cached recursion", commandLine: concat("git rm --cached -", "r ."), wantStatus: 2, wantOutput: "BLOCKED [D2]"},
		{name: "blocks force push to protected branch", commandLine: concat("git push --", "force origin ", "main"), wantStatus: 2, wantOutput: "BLOCKED [D3]"},
		{name: "blocks history rewrite push to protected branch", commandLine: concat("git push origin ", "+HEAD:", "main"), wantStatus: 2, wantOutput: "BLOCKED [D4]"},
		{name: "blocks dropdb", commandLine: concat("drop", "db sample"), wantStatus: 2, wantOutput: "BLOCKED [D5]"},
		{name: "blocks mkfs", commandLine: concat("mkfs", ".ext4 /dev/sda"), wantStatus: 2, wantOutput: "BLOCKED [D6]"},
		{name: "blocks git reset hard", commandLine: concat("git reset --", "hard HEAD"), wantStatus: 2, wantOutput: "BLOCKED [D7]"},
		{name: "blocks commit hook bypass", commandLine: concat("git commit --no-", "verify -m test"), wantStatus: 2, wantOutput: "BLOCKED [D8]"},
		{name: "blocks exfil command", commandLine: concat("cu", "rl https://example.com"), wantStatus: 2, wantOutput: "BLOCKED [D11]"},
		{name: "blocks sudo", commandLine: concat("su", "do ls"), wantStatus: 2, wantOutput: "BLOCKED [D12]"},
		{name: "blocks remote shell", commandLine: concat("s", "sh prod"), wantStatus: 2, wantOutput: "BLOCKED [D13]"},
		{name: "blocks destructive docker", commandLine: concat("docker system pr", "une -f"), wantStatus: 2, wantOutput: "BLOCKED [D14]"},
		{name: "blocks shell history", commandLine: concat("hist", "ory"), wantStatus: 2, wantOutput: "BLOCKED [D15]"},
		{name: "blocks interpreter command", commandLine: concat("ba", "sh -c 'echo hi'"), wantStatus: 2, wantOutput: "BLOCKED [D16]"},
		{name: "blocks dangerous compound", commandLine: concat("git status && ", "cu", "rl https://example.com"), wantStatus: 2, wantOutput: "BLOCKED [D11]"},
		{name: "allows safe compound", commandLine: concat("git status && ", "printf ok"), wantStatus: 0, wantOutput: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			status, output := runHookScript(t, repoRoot, guardPath, nil, `{"tool_input":{"command":"`+tt.commandLine+`"}}`)
			if status != tt.wantStatus {
				t.Fatalf("status = %d, want %d, output = %s", status, tt.wantStatus, output)
			}
			if tt.wantOutput != "" && !strings.Contains(output, tt.wantOutput) {
				t.Fatalf("output = %q, want substring %q", output, tt.wantOutput)
			}
		})
	}
}

func TestSecretScanBlocksExpandedD10Verbs(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	scriptPath := filepath.Join(repoRoot, ".claude", "hooks", "secret-scan.sh")

	tests := []struct {
		name        string
		commandLine string
	}{
		{name: "launchctl getenv", commandLine: concat("launchctl ", "getenv AWS_SECRET_ACCESS_KEY")},
		{name: "proc environ", commandLine: concat("cat /proc/self/", "environ")},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			status, output := runHookScript(t, repoRoot, scriptPath, []string{"scan-command", "--command", tt.commandLine}, "")
			if status != 2 {
				t.Fatalf("status = %d, want 2, output = %s", status, output)
			}
			if !strings.Contains(output, "BLOCKED [D10]") {
				t.Fatalf("output = %q, want D10 block", output)
			}
		})
	}
}

func runHookScript(t *testing.T, dir string, scriptPath string, args []string, stdin string) (int, string) {
	t.Helper()

	cmd := exec.Command(scriptPath, args...)
	cmd.Dir = dir
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if err == nil {
		return 0, output.String()
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("Run(%s %v) error = %v\n%s", scriptPath, args, err, output.String())
	}

	return exitErr.ExitCode(), output.String()
}

func concat(parts ...string) string {
	return strings.Join(parts, "")
}
