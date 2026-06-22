package initcmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"mkproj/internal/delegate"
	"mkproj/internal/project"
	"mkproj/internal/remote"
	"mkproj/internal/scaffold"
)

type Initializer struct {
	Writer scaffold.Writer
	Runner delegate.Runner
}

func (i Initializer) Run(ctx context.Context, targetDir string, vars project.Variables) error {
	if err := scaffoldEnsureEmptyDir(targetDir); err != nil {
		return failWithRecovery(targetDir, "empty-directory precondition", err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return failWithRecovery(targetDir, "target directory setup", err)
	}

	if err := i.Runner.Run(ctx, targetDir, "git init", "git", "init", "-b", "main"); err != nil {
		return failWithRecovery(targetDir, "git init", err)
	}
	if err := i.Runner.Run(ctx, targetDir, "git identity name", "git", "config", "user.name", vars.AuthorName); err != nil {
		return failWithRecovery(targetDir, "git identity name", err)
	}
	if err := i.Runner.Run(ctx, targetDir, "git identity email", "git", "config", "user.email", vars.AuthorEmail); err != nil {
		return failWithRecovery(targetDir, "git identity email", err)
	}

	if err := i.Writer.Write(targetDir, vars); err != nil {
		return failWithRecovery(targetDir, "phase 1 scaffold writer", err)
	}

	steps := []struct {
		name    string
		command string
		args    []string
	}{
		{name: "bd init", command: "bd", args: []string{"init"}},
		{name: "instill init", command: "instill", args: []string{"init"}},
		{name: "instill pick-skills", command: "instill", args: []string{"pick-skills"}},
		{name: "instill check-skills", command: "instill", args: []string{"check-skills"}},
		{name: "lefthook install", command: "lefthook", args: []string{"install", "--force"}},
	}

	for _, step := range steps {
		if err := i.Runner.Run(ctx, targetDir, step.name, step.command, step.args...); err != nil {
			return failWithRecovery(targetDir, step.name, err)
		}
	}

	if err := remote.Publish(ctx, i.Runner, targetDir, remote.PublishOptions{
		RepoName: filepath.Base(targetDir),
		Remote:   vars.Remote,
		URL:      vars.RemoteURL,
	}); err != nil {
		return failWithRecovery(targetDir, "phase 3 remote publish", err)
	}

	return nil
}

func failWithRecovery(targetDir string, step string, err error) error {
	return fmt.Errorf("init failed at step %q: %w\nRecovery: delete the directory recursively and retry: %s", step, err, targetDir)
}

func scaffoldEnsureEmptyDir(targetDir string) error {
	info, err := os.Stat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", targetDir)
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == ".DS_Store" {
			continue
		}
		return fmt.Errorf("directory not empty: %s", targetDir)
	}

	return nil
}
