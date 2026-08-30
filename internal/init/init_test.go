package initcmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"forge"
	"forge/internal/project"
	"forge/internal/scaffold"
	"forge/internal/upgrade"
)

func TestInitializerRunsPhaseOneThenDelegatesThenRemote(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := &recordingRunner{}
	writer := scaffold.Writer{Assets: fstest.MapFS{
		"templates/common/AGENTS.md.tmpl": {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base": {Data: []byte(".DS_Store\n")},
		"templates/seed/skills.json.tmpl": {
			Data: []byte("{\"skills\":[\"golang-cli\",\"mise\"]}\n"),
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
		"instill bootstrap",
		"instill init",
		"instill sync",
		"mise trust",
		"mise install",
		"go mod tidy",
		"lefthook install",
		"git add",
		"git commit",
	}
	if got := runner.stepNames(); !equalStrings(got, want) {
		t.Fatalf("step order = %#v, want %#v", got, want)
	}

	assertRecordedStepArgs(t, runner.steps, "instill init", "init", "--force", "--skills", "golang-cli,mise", "--targets", "claude,codex,opencode")
	assertRecordedStepArgs(t, runner.steps, "mise trust", "trust", "--all")
}

func TestInitializerStampsForgeManifest(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := &recordingRunner{}
	writer := scaffold.Writer{Assets: fstest.MapFS{
		"templates/common/AGENTS.md.tmpl": {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base": {Data: []byte(".DS_Store\n")},
		"templates/seed/skills.json.tmpl": {
			Data: []byte("{\"skills\":[\"golang-cli\",\"mise\"]}\n"),
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

	// The manifest and legacy marker are stamped during Phase 1, before the
	// remote publish step's git add/commit, so both ride the scaffold commit.
	m, err := upgrade.ReadManifest(tempDir)
	if err != nil {
		t.Fatalf("ReadManifest() error = %v; init must stamp .forge/manifest.json", err)
	}
	if m.InfraVersion != upgrade.Version {
		t.Fatalf("manifest infraVersion = %d, want %d", m.InfraVersion, upgrade.Version)
	}
	if m.Language != "go" || m.Stack != "go-cli-cobra" {
		t.Fatalf("manifest params = %+v, want language go / stack go-cli-cobra", m)
	}

	legacy, err := os.ReadFile(filepath.Join(tempDir, ".forge-infra-version"))
	if err != nil {
		t.Fatalf("legacy marker not written: %v", err)
	}
	if strings.TrimSpace(string(legacy)) != strconv.Itoa(upgrade.Version) {
		t.Fatalf("legacy marker = %q, want %d", string(legacy), upgrade.Version)
	}
}

func TestInitializerRunsPipInstallForPythonProjects(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := &recordingRunner{}
	writer := scaffold.Writer{Assets: fstest.MapFS{
		"templates/common/AGENTS.md.tmpl": {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base": {Data: []byte(".DS_Store\n")},
		"templates/seed/skills.json.tmpl": {
			Data: []byte("{\"skills\":[\"mise\"]}\n"),
		},
		"templates/common/claude/hooks/secret-scan.sh":          {Data: []byte("#!/usr/bin/env bash\n")},
		"templates/common/codex/hooks.json":                     {Data: []byte("{\"hooks\":{}}\n")},
		"templates/gitignore/Python.gitignore":                  {Data: []byte("__pycache__/\n")},
		"templates/golden/python-cli-typer/pyproject.toml.tmpl": {Data: []byte("[project]\nname = \"app\"\n")},
	}}
	init := Initializer{Writer: writer, Runner: runner}

	vars, err := project.ResolveVariables(project.Input{
		ProjectName: "Snake App",
		Language:    "python",
		ProjectType: "cli",
		Stack:       "python-cli-typer",
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
		"instill bootstrap",
		"instill init",
		"instill sync",
		"mise trust",
		"mise install",
		"pip install",
		"lefthook install",
		"git add",
		"git commit",
	}
	if got := runner.stepNames(); !equalStrings(got, want) {
		t.Fatalf("step order = %#v, want %#v", got, want)
	}

	assertRecordedStepArgs(t, runner.steps, "pip install", "exec", "--", "pip", "install", "-e", ".[dev]")
}

func TestInitializerPassesSeedSkillsToInstillInit(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := &recordingRunner{}
	writer := scaffold.Writer{Assets: fstest.MapFS{
		"templates/common/AGENTS.md.tmpl": {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base": {Data: []byte(".DS_Store\n")},
		"templates/seed/skills.json.tmpl": {
			Data: []byte("{\"skills\":[\"golang-cli\",\"mise\",\"brainstorming\"]}\n"),
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
		"init", "--force", "--skills", "golang-cli,mise,brainstorming", "--targets", "claude,codex,opencode")
}

func TestInitializerFallsBackToPerSkillInstillInitWhenCombinedInitFails(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := &recordingRunner{
		failRun: func(step string, command string, args []string) error {
			if command != "instill" || len(args) == 0 || args[0] != "init" {
				return nil
			}

			switch {
			case step == "instill init":
				return errors.New("combined init failed")
			case skillArg(args) == "writing-rules":
				return errors.New("permission denied")
			default:
				return nil
			}
		},
	}
	writer := scaffold.Writer{Assets: fstest.MapFS{
		"templates/common/AGENTS.md.tmpl": {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base": {Data: []byte(".DS_Store\n")},
		"templates/seed/skills.json.tmpl": {
			Data: []byte("{\"skills\":[\"golang-cli\",\"writing-rules\",\"mise\"]}\n"),
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
		"init", "--force", "--skills", "golang-cli,writing-rules,mise", "--targets", "claude,codex,opencode")
	assertRecordedStepArgs(t, runner.steps, "instill init (golang-cli)",
		"init", "--force", "--skills", "golang-cli", "--targets", "claude,codex,opencode")
	assertRecordedStepArgs(t, runner.steps, "instill init (writing-rules)",
		"init", "--force", "--skills", "writing-rules", "--targets", "claude,codex,opencode")
	assertRecordedStepArgs(t, runner.steps, "instill init (mise)",
		"init", "--force", "--skills", "mise", "--targets", "claude,codex,opencode")
	assertRecordedStepArgs(t, runner.steps, "instill init (retry)",
		"init", "--force", "--skills", "golang-cli,mise", "--targets", "claude,codex,opencode")

	if !hasStep(runner.steps, "instill sync") {
		t.Fatalf("instill sync should run after successful fallback init: %#v", runner.stepNames())
	}
}

func TestInitializerContinuesWhenNoSeedSkillsCanBeInitialized(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := &recordingRunner{
		failRun: func(_ string, command string, args []string) error {
			if command != "instill" || len(args) == 0 || args[0] != "init" {
				return nil
			}
			return errors.New("permission denied")
		},
	}
	writer := scaffold.Writer{Assets: fstest.MapFS{
		"templates/common/AGENTS.md.tmpl": {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base": {Data: []byte(".DS_Store\n")},
		"templates/seed/skills.json.tmpl": {
			Data: []byte("{\"skills\":[\"executing-plans\",\"mise\"]}\n"),
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

	if hasStep(runner.steps, "instill sync") {
		t.Fatalf("instill sync should be skipped when no skills can be initialized: %#v", runner.stepNames())
	}
}

func TestInitializerSkipsSkillSetupWhenInstillBootstrapFails(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := &recordingRunner{
		failRun: func(step string, _ string, _ []string) error {
			if step == "instill bootstrap" {
				return errors.New("brew not found")
			}
			return nil
		},
	}
	writer := scaffold.Writer{Assets: fstest.MapFS{
		"templates/common/AGENTS.md.tmpl": {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base": {Data: []byte(".DS_Store\n")},
		"templates/seed/skills.json.tmpl": {
			Data: []byte("{\"skills\":[\"mise\"]}\n"),
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

	for _, step := range []string{"instill init", "instill sync"} {
		if hasStep(runner.steps, step) {
			t.Fatalf("%s should be skipped when instill bootstrap fails: %#v", step, runner.stepNames())
		}
	}
	if !hasStep(runner.steps, "git commit") {
		t.Fatalf("init should continue past a failed instill bootstrap: %#v", runner.stepNames())
	}
}

func TestReadSeedSkills(t *testing.T) {
	t.Parallel()

	goSkills, err := readSeedSkills(forge.Assets(), "go")
	if err != nil {
		t.Fatalf("readSeedSkills(go) error = %v", err)
	}
	for _, want := range []string{"coding/golang/golang-cli", "productivity/mise", "superpowers/brainstorming"} {
		if !slices.Contains(goSkills, want) {
			t.Fatalf("go seed skills = %#v, want to contain %q", goSkills, want)
		}
	}

	pythonSkills, err := readSeedSkills(forge.Assets(), "python")
	if err != nil {
		t.Fatalf("readSeedSkills(python) error = %v", err)
	}
	if slices.Contains(pythonSkills, "coding/golang/golang-cli") {
		t.Fatalf("python seed skills = %#v, want no golang-cli", pythonSkills)
	}
	if !slices.Contains(pythonSkills, "productivity/mise") {
		t.Fatalf("python seed skills = %#v, want to contain mise", pythonSkills)
	}
	if !slices.Contains(pythonSkills, "coding/python/python-code-style") {
		t.Fatalf("python seed skills = %#v, want to contain python-code-style", pythonSkills)
	}
}

func TestInitializerStopsAtTheFailedStepWithRecoveryText(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := &recordingRunner{failStep: "instill init"}
	writer := scaffold.Writer{Assets: fstest.MapFS{
		"templates/common/AGENTS.md.tmpl":              {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base":              {Data: []byte(".DS_Store\n")},
		"templates/seed/skills.json.tmpl":              {Data: []byte("{\"skills\":[\"mise\"]}\n")},
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
		"templates/common/AGENTS.md.tmpl":              {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base":              {Data: []byte(".DS_Store\n")},
		"templates/seed/skills.json.tmpl":              {Data: []byte("{\"skills\":[\"mise\"]}\n")},
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
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertRecordedStepArgs(t, runner.steps, "lefthook install", "exec", "--", "lefthook", "install", "--force")

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

func TestInitializerSecuresBeadsDirPermissions(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := &recordingRunner{
		afterStep: func(dir string, step string, _ string, _ ...string) error {
			if step == "bd init" {
				// bd init creates .beads under the ambient umask → 0755.
				if err := seedBeadsHooks(dir); err != nil {
					return err
				}
				return os.Chmod(filepath.Join(dir, ".beads"), 0o755)
			}
			return nil
		},
	}
	writer := scaffold.Writer{Assets: fstest.MapFS{
		"templates/common/AGENTS.md.tmpl":              {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base":              {Data: []byte(".DS_Store\n")},
		"templates/seed/skills.json.tmpl":              {Data: []byte("{\"skills\":[\"mise\"]}\n")},
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
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(tempDir, ".beads"))
	if err != nil {
		t.Fatalf("Stat(.beads) error = %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf(".beads mode after init = %o, want 700 (bd requires owner-only)", info.Mode().Perm())
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
	failRun   func(step string, command string, args []string) error
}

type recordedStep struct {
	name    string
	command string
	args    []string
}

func (r *recordingRunner) Run(_ context.Context, dir string, step string, command string, args ...string) error {
	r.steps = append(r.steps, recordedStep{name: step, command: command, args: args})
	if r.failRun != nil {
		if err := r.failRun(step, command, args); err != nil {
			return err
		}
	}
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

func hasStep(steps []recordedStep, name string) bool {
	for _, step := range steps {
		if step.name == name {
			return true
		}
	}

	return false
}

func lastArg(args []string) string {
	if len(args) == 0 {
		return ""
	}

	return args[len(args)-1]
}

func skillArg(args []string) string {
	for i, arg := range args {
		if arg == "--skills" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func TestInitializerRunsNpmInstallForTypescriptProjects(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := &recordingRunner{}
	writer := scaffold.Writer{Assets: fstest.MapFS{
		"templates/common/AGENTS.md.tmpl": {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base": {Data: []byte(".DS_Store\n")},
		"templates/seed/skills.json.tmpl": {
			Data: []byte("{\"skills\":[\"mise\"]}\n"),
		},
		"templates/common/claude/hooks/secret-scan.sh": {Data: []byte("#!/usr/bin/env bash\n")},
		"templates/common/codex/hooks.json":            {Data: []byte("{\"hooks\":{}}\n")},
		"templates/gitignore/Node.gitignore":           {Data: []byte("node_modules/\n")},
		"templates/golden/vite-ts/package.json.tmpl":   {Data: []byte("{\"name\": \"{{.NpmPackage}}\"}\n")},
	}}
	init := Initializer{Writer: writer, Runner: runner}

	vars, err := project.ResolveVariables(project.Input{
		ProjectName: "Web App",
		Language:    "typescript",
		ProjectType: "frontend",
		Stack:       "vite-ts",
		APIBaseURL:  "https://api.example.com",
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
		"instill bootstrap",
		"instill init",
		"instill sync",
		"mise trust",
		"mise install",
		"npm install",
		"lefthook install",
		"git add",
		"git commit",
	}
	if got := runner.stepNames(); !equalStrings(got, want) {
		t.Fatalf("step order = %#v, want %#v", got, want)
	}

	assertRecordedStepArgs(t, runner.steps, "npm install", "exec", "--", "npm", "install")
}

func TestInitializerRunsWebNpmInstallForFullstackProjects(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	runner := &recordingRunner{}
	writer := scaffold.Writer{Assets: fstest.MapFS{
		"templates/common/AGENTS.md.tmpl": {Data: []byte("Project {{.ProjectName}}\n")},
		"templates/common/gitignore.base": {Data: []byte(".DS_Store\n")},
		"templates/seed/skills.json.tmpl": {
			Data: []byte("{\"skills\":[\"mise\"]}\n"),
		},
		"templates/common/claude/hooks/secret-scan.sh": {Data: []byte("#!/usr/bin/env bash\n")},
		"templates/common/codex/hooks.json":            {Data: []byte("{\"hooks\":{}}\n")},
		"templates/gitignore/Go.gitignore":             {Data: []byte("bin/\n")},
		"templates/gitignore/Node.gitignore":           {Data: []byte("node_modules/\n")},
		"templates/golden/go-api-chi/go.mod.tmpl":      {Data: []byte("module {{.ModulePath}}\n")},
		"templates/golden/vite-ts/package.json.tmpl":   {Data: []byte("{\"name\": \"{{.NpmPackage}}\"}\n")},
	}}
	init := Initializer{Writer: writer, Runner: runner}

	vars, err := project.ResolveVariables(project.Input{
		ProjectName: "Full App",
		Language:    "go",
		ProjectType: "fullstack",
		Stack:       "go-api-chi",
		Frontend:    "vite-ts",
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
		"instill bootstrap",
		"instill init",
		"instill sync",
		"mise trust",
		"mise install",
		"go mod tidy",
		"npm install (web)",
		"lefthook install",
		"git add",
		"git commit",
	}
	if got := runner.stepNames(); !equalStrings(got, want) {
		t.Fatalf("step order = %#v, want %#v", got, want)
	}

	assertRecordedStepArgs(t, runner.steps, "npm install (web)", "exec", "--", "npm", "--prefix", "web", "install")
}
