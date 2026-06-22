package initcmd

import (
	"context"
	"errors"
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
		"templates/common/AGENTS.md.tmpl":              {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base":              {Data: []byte(".DS_Store\n")},
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
		"instill pick-skills",
		"instill check-skills",
		"lefthook install",
		"git add",
		"git commit",
	}
	if got := runner.stepNames(); !equalStrings(got, want) {
		t.Fatalf("step order = %#v, want %#v", got, want)
	}
}

func TestInitializerStopsAtTheFailedStepWithRecoveryText(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := &recordingRunner{failStep: "instill init"}
	writer := scaffold.Writer{Assets: fstest.MapFS{
		"templates/common/AGENTS.md.tmpl":              {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base":              {Data: []byte(".DS_Store\n")},
		"templates/common/claude/hooks/secret-scan.sh": {Data: []byte("#!/usr/bin/env bash\n")},
		"templates/common/codex/hooks.json":            {Data: []byte("{\"hooks\":{}}\n")},
		"templates/gitignore/Go.gitignore":             {Data: []byte("bin/\n")},
		"templates/golden/go-cli-cobra/main.go":        {Data: []byte("package main\n")},
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

type recordingRunner struct {
	failStep string
	steps    []recordedStep
}

type recordedStep struct {
	name    string
	command string
	args    []string
}

func (r *recordingRunner) Run(_ context.Context, _ string, step string, command string, args ...string) error {
	r.steps = append(r.steps, recordedStep{name: step, command: command, args: args})
	if step == r.failStep {
		return errors.New("boom")
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
