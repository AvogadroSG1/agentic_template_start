package initcmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"mkproj/internal/project"
	"mkproj/internal/scaffold"
)

func TestInitializerRunsPhaseOneThenDelegatesThenRemote(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := &recordingRunner{}
	writer := scaffold.Writer{Assets: fstest.MapFS{
		"templates/common/AGENTS.md.tmpl": {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base": {Data: []byte(".DS_Store\n")},
		"templates/common/claude/skill-manifest.json.tmpl": {
			Data: []byte("{\"skills\":[\"golang/golang-cli\",\"productivity/mise\"]}\n"),
		},
		"templates/common/claude/hooks/secret-scan.sh": {Data: []byte("#!/usr/bin/env bash\n")},
		"templates/common/codex/hooks.json":            {Data: []byte("{\"hooks\":{}}\n")},
		"templates/gitignore/Go.gitignore":             {Data: []byte("bin/\n")},
		"templates/golden/go-cli-cobra/main.go.tmpl":   {Data: []byte("package main\n")},
	}}
	init := Initializer{Writer: writer, Runner: runner}

	vars, err := project.ResolveVariables(project.Input{
		ProjectName: "Sample App",
		Language:    "go",
		ProjectType: "cli",
		Stack:       "go-cli-cobra",
		AuthorName:  "Ada Lovelace",
		AuthorEmail: "ada@example.com",
		Remote:      project.RemoteNone,
	})
	if err != nil {
		t.Fatalf("ResolveVariables() error = %v", err)
	}

	if err := init.Run(context.Background(), tempDir, vars); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := []string{
		"git init",
		"git identity name",
		"git identity email",
		"bd init",
		"instill init",
		"instill check-skills",
		"lefthook install",
		"git add",
		"git commit",
	}
	if got := runner.stepNames(); !equalStrings(got, want) {
		t.Fatalf("step order = %#v, want %#v", got, want)
	}

	assertRecordedStepArgs(t, runner.steps, "instill init", "init", "--force", "--skills", "golang/golang-cli,productivity/mise")
}

func TestInitializerPassesManifestSkillsToInstillInit(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := &recordingRunner{}
	writer := scaffold.Writer{Assets: fstest.MapFS{
		"templates/common/AGENTS.md.tmpl": {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base": {Data: []byte(".DS_Store\n")},
		"templates/common/claude/skill-manifest.json.tmpl": {
			Data: []byte("{\"skills\":[\"golang/golang-cli\",\"productivity/mise\",\"superpowers/brainstorming\"]}\n"),
		},
		"templates/common/claude/hooks/secret-scan.sh": {Data: []byte("#!/usr/bin/env bash\n")},
		"templates/common/codex/hooks.json":            {Data: []byte("{\"hooks\":{}}\n")},
		"templates/gitignore/Go.gitignore":             {Data: []byte("bin/\n")},
		"templates/golden/go-cli-cobra/main.go.tmpl":   {Data: []byte("package main\n")},
	}}
	init := Initializer{Writer: writer, Runner: runner}

	vars, err := project.ResolveVariables(project.Input{
		ProjectName: "Sample App",
		Language:    "go",
		ProjectType: "cli",
		Stack:       "go-cli-cobra",
		AuthorName:  "Ada Lovelace",
		AuthorEmail: "ada@example.com",
		Remote:      project.RemoteNone,
	})
	if err != nil {
		t.Fatalf("ResolveVariables() error = %v", err)
	}

	if err := init.Run(context.Background(), tempDir, vars); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertRecordedStepArgs(t, runner.steps, "instill init",
		"init", "--force", "--skills", "golang/golang-cli,productivity/mise,superpowers/brainstorming")
}

func TestInitializerStopsAtTheFailedStepWithRecoveryText(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := &recordingRunner{failStep: "instill init"}
	writer := scaffold.Writer{Assets: fstest.MapFS{
		"templates/common/AGENTS.md.tmpl":                  {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base":                  {Data: []byte(".DS_Store\n")},
		"templates/common/claude/skill-manifest.json.tmpl": {Data: []byte("{\"skills\":[\"productivity/mise\"]}\n")},
		"templates/common/claude/hooks/secret-scan.sh":     {Data: []byte("#!/usr/bin/env bash\n")},
		"templates/common/codex/hooks.json":                {Data: []byte("{\"hooks\":{}}\n")},
		"templates/gitignore/Go.gitignore":                 {Data: []byte("bin/\n")},
		"templates/golden/go-cli-cobra/main.go":            {Data: []byte("package main\n")},
	}}
	init := Initializer{Writer: writer, Runner: runner}

	err := init.Run(context.Background(), tempDir, project.Variables{
		ProjectName: "Sample App",
		Language:    "go",
		Stack:       "go-cli-cobra",
		AuthorName:  "Ada Lovelace",
		AuthorEmail: "ada@example.com",
		Remote:      project.RemoteNone,
	})
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if !strings.Contains(err.Error(), `init failed at step "instill init"`) {
		t.Fatalf("Run() error = %v, want failed step text", err)
	}
	if !strings.Contains(err.Error(), "delete the directory recursively") {
		t.Fatalf("Run() error = %v, want recovery text", err)
	}
}

func TestInitializerRepairsBeadsHooksAfterForcedLefthookInstall(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := &recordingRunner{
		afterStep: func(dir string, step string, _ string, _ ...string) error {
			switch step {
			case "bd init":
				return seedBeadsHooks(dir)
			case "lefthook install":
				return simulateForcedLefthookInstall(dir)
			default:
				return nil
			}
		},
	}
	writer := scaffold.Writer{Assets: fstest.MapFS{
		"templates/common/AGENTS.md.tmpl":                  {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base":                  {Data: []byte(".DS_Store\n")},
		"templates/common/claude/skill-manifest.json.tmpl": {Data: []byte("{\"skills\":[\"productivity/mise\"]}\n")},
		"templates/common/claude/hooks/secret-scan.sh":     {Data: []byte("#!/usr/bin/env bash\n")},
		"templates/common/codex/hooks.json":                {Data: []byte("{\"hooks\":{}}\n")},
		"templates/gitignore/Go.gitignore":                 {Data: []byte("bin/\n")},
		"templates/golden/go-cli-cobra/main.go":            {Data: []byte("package main\n")},
	}}
	init := Initializer{Writer: writer, Runner: runner}

	err := init.Run(context.Background(), tempDir, project.Variables{
		ProjectName: "Sample App",
		Language:    "go",
		Stack:       "go-cli-cobra",
		AuthorName:  "Ada Lovelace",
		AuthorEmail: "ada@example.com",
		Remote:      project.RemoteNone,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertRecordedStepArgs(t, runner.steps, "lefthook install", "install", "--force")

	checks := []struct {
		hook      string
		wantLines []string
	}{
		{hook: "pre-commit", wantLines: []string{"beads pre-commit", "lefthook pre-commit"}},
		{hook: "pre-push", wantLines: []string{"beads pre-push", "lefthook pre-push"}},
	}

	for _, check := range checks {
		t.Run(check.hook, func(t *testing.T) {
			hookPath := filepath.Join(tempDir, ".beads", "hooks", check.hook)
			if _, err := os.Stat(hookPath + ".old"); err != nil {
				t.Fatalf("Stat(%s.old) error = %v", hookPath, err)
			}
			if _, err := os.Stat(hookPath + ".lefthook"); err != nil {
				t.Fatalf("Stat(%s.lefthook) error = %v", hookPath, err)
			}

			output := runHook(t, hookPath)
			for _, want := range check.wantLines {
				if !strings.Contains(output, want) {
					t.Fatalf("%s output = %q, want %q", hookPath, output, want)
				}
			}
			if strings.Index(output, check.wantLines[0]) > strings.Index(output, check.wantLines[1]) {
				t.Fatalf("%s output order = %q, want beads hook before lefthook", hookPath, output)
			}
		})
	}
}

func seedBeadsHooks(dir string) error {
	hooksDir := filepath.Join(dir, ".beads", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return err
	}

	for _, hook := range []string{"pre-commit", "pre-push"} {
		content := "#!/usr/bin/env bash\nset -euo pipefail\necho \"beads " + hook + "\"\n"
		if err := os.WriteFile(filepath.Join(hooksDir, hook), []byte(content), 0o755); err != nil {
			return err
		}
	}

	return nil
}

func simulateForcedLefthookInstall(dir string) error {
	hooksDir := filepath.Join(dir, ".beads", "hooks")
	for _, hook := range []string{"pre-commit", "pre-push"} {
		hookPath := filepath.Join(hooksDir, hook)
		if err := os.Rename(hookPath, hookPath+".old"); err != nil {
			return err
		}

		content := "#!/usr/bin/env bash\nset -euo pipefail\necho \"lefthook " + hook + "\"\n"
		if err := os.WriteFile(hookPath, []byte(content), 0o755); err != nil {
			return err
		}
	}

	return nil
}

func runHook(t *testing.T, hookPath string) string {
	t.Helper()

	cmd := exec.Command(hookPath)
	var output strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		t.Fatalf("Run(%s) error = %v\n%s", hookPath, err, output.String())
	}

	return output.String()
}

func assertRecordedStepArgs(t *testing.T, steps []recordedStep, name string, wantArgs ...string) {
	t.Helper()

	for _, step := range steps {
		if step.name != name {
			continue
		}
		if !equalStrings(step.args, wantArgs) {
			t.Fatalf("%s args = %#v, want %#v", name, step.args, wantArgs)
		}
		return
	}

	t.Fatalf("%s step not recorded", name)
}

type recordingRunner struct {
	failStep  string
	steps     []recordedStep
	afterStep func(dir string, step string, command string, args ...string) error
}

type recordedStep struct {
	name    string
	command string
	args    []string
}

func (r *recordingRunner) Run(_ context.Context, dir string, step string, command string, args ...string) error {
	r.steps = append(r.steps, recordedStep{name: step, command: command, args: args})
	if step == r.failStep {
		return errors.New("boom")
	}
	if r.afterStep != nil {
		return r.afterStep(dir, step, command, args...)
	}

	return nil
}

func (r *recordingRunner) stepNames() []string {
	names := make([]string, 0, len(r.steps))
	for _, step := range r.steps {
		names = append(names, step.name)
	}

	return names
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}

	return true
}
