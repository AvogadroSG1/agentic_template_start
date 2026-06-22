package update

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

type commandCall struct {
	Dir     string
	Step    string
	Command string
	Args    []string
}

type recordingCommandRunner struct {
	calls   []commandCall
	runFunc func(dir string, step string, command string, args ...string) error
}

func (r *recordingCommandRunner) Run(_ context.Context, dir string, step string, command string, args ...string) error {
	r.calls = append(r.calls, commandCall{Dir: dir, Step: step, Command: command, Args: append([]string(nil), args...)})
	if r.runFunc != nil {
		return r.runFunc(dir, step, command, args...)
	}
	return nil
}

type recordingGitRunner struct {
	operations []string
	cloneFunc  func(dir string)
}

func (r *recordingGitRunner) Clone(_ context.Context, repo string, dir string) error {
	r.operations = append(r.operations, "clone:"+repo)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if r.cloneFunc != nil {
		r.cloneFunc(dir)
	}
	return nil
}

func (r *recordingGitRunner) Checkout(_ context.Context, dir string, ref string) error {
	r.operations = append(r.operations, "checkout:"+ref)
	return nil
}

func TestRunExecutesSingleStepRecipeInOrder(t *testing.T) {
	t.Parallel()

	assets := fstest.MapFS{
		"sources.yaml": {Data: []byte(`single-step:
  kind: scaffolder
  steps:
    - run: "cobra-cli init --pkg-name example.com/sample"
  gitignore: Go
  normalize: []
  resolved:
    ref: "v1.3.0"
    captured: "2026-06-20"
`)},
	}
	runner := &recordingCommandRunner{}
	git := &recordingGitRunner{}

	if err := Run(context.Background(), assets, "single-step", runner, git); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(git.operations) != 0 {
		t.Fatalf("git operations = %#v, want none", git.operations)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("runner calls = %#v, want 1 call", runner.calls)
	}
	call := runner.calls[0]
	if call.Command != "cobra-cli" {
		t.Fatalf("command = %q, want cobra-cli", call.Command)
	}
	if got := strings.Join(call.Args, " "); got != "init --pkg-name example.com/sample" {
		t.Fatalf("args = %q, want %q", got, "init --pkg-name example.com/sample")
	}
	if call.Dir == "" {
		t.Fatal("call dir is empty")
	}
}

func TestRunExecutesCheckoutStripAndRunRecipe(t *testing.T) {
	t.Parallel()

	assets := fstest.MapFS{
		"sources.yaml": {Data: []byte(`recipe-stack:
  kind: recipe
  steps:
    - checkout: "github.com/example/project-layout"
      ref: "abc123"
      strip: [".git", "README.md", "docs"]
    - run: "go mod init example.com/sample"
  gitignore: Go
  normalize: []
  resolved:
    ref: "abc123"
    captured: "2026-06-20"
`)},
	}
	git := &recordingGitRunner{cloneFunc: func(dir string) {
		mustWriteFile(t, filepath.Join(dir, "README.md"), "remove me")
		mustWriteFile(t, filepath.Join(dir, "docs", "guide.md"), "remove me")
		mustWriteFile(t, filepath.Join(dir, ".git", "HEAD"), "ref: refs/heads/main")
		mustWriteFile(t, filepath.Join(dir, "keep.txt"), "keep me")
	}}
	runner := &recordingCommandRunner{runFunc: func(dir string, _ string, command string, args ...string) error {
		if command != "go" {
			return errors.New("unexpected command")
		}
		assertMissingPath(t, filepath.Join(dir, "README.md"))
		assertMissingPath(t, filepath.Join(dir, "docs"))
		assertMissingPath(t, filepath.Join(dir, ".git"))
		if _, err := os.Stat(filepath.Join(dir, "keep.txt")); err != nil {
			t.Fatalf("keep.txt missing before run: %v", err)
		}
		return nil
	}}

	if err := Run(context.Background(), assets, "recipe-stack", runner, git); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := strings.Join(git.operations, ","); got != "clone:github.com/example/project-layout,checkout:abc123" {
		t.Fatalf("git operations = %q", got)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("runner calls = %#v, want 1 call", runner.calls)
	}
	if runner.calls[0].Command != "go" {
		t.Fatalf("command = %q, want go", runner.calls[0].Command)
	}
	if got := strings.Join(runner.calls[0].Args, " "); got != "mod init example.com/sample" {
		t.Fatalf("args = %q, want %q", got, "mod init example.com/sample")
	}
}

func TestRunReturnsHelpfulErrorWhenToolIsMissing(t *testing.T) {
	t.Parallel()

	assets := fstest.MapFS{
		"sources.yaml": {Data: []byte(`single-step:
  kind: scaffolder
  steps:
    - run: "cobra-cli init --pkg-name example.com/sample"
  gitignore: Go
  normalize: []
  resolved:
    ref: "v1.3.0"
    captured: "2026-06-20"
`)},
	}
	runner := &recordingCommandRunner{runFunc: func(_ string, _ string, _ string, _ ...string) error {
		return exec.ErrNotFound
	}}

	err := Run(context.Background(), assets, "single-step", runner, &recordingGitRunner{})
	if err == nil {
		t.Fatal("Run() error = nil, want missing tool error")
	}
	for _, snippet := range []string{"single-step", "cobra-cli"} {
		if !strings.Contains(err.Error(), snippet) {
			t.Fatalf("Run() error = %q, want snippet %q", err, snippet)
		}
	}
}

func mustWriteFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func assertMissingPath(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Stat(%s) error = %v, want not exists", path, err)
	}
}

func TestRunPreservesQuotedArguments(t *testing.T) {
	t.Parallel()

	assets := fstest.MapFS{
		"sources.yaml": {Data: []byte(`quoted-step:
  kind: scaffolder
  steps:
    - run: "uv add fastapi 'uvicorn[standard]'"
  gitignore: Python
  normalize: []
  resolved:
    ref: "uv 0.11.8"
    captured: "2026-06-20"
`)},
	}
	runner := &recordingCommandRunner{}

	if err := Run(context.Background(), assets, "quoted-step", runner, &recordingGitRunner{}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("runner calls = %#v, want 1 call", runner.calls)
	}
	if got := strings.Join(runner.calls[0].Args, "|"); got != "add|fastapi|uvicorn[standard]" {
		t.Fatalf("args = %q, want %q", got, "add|fastapi|uvicorn[standard]")
	}
}
