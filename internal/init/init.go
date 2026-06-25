package initcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	skills, err := readManifestSkills(filepath.Join(targetDir, ".claude", "skill-manifest.json"))
	if err != nil {
		return failWithRecovery(targetDir, "skill manifest read", err)
	}

	steps := []struct {
		name    string
		command string
		args    []string
	}{
		{name: "bd init", command: "bd", args: []string{"init"}},
		{name: "instill init", command: "instill", args: []string{"init", "--force", "--skills", strings.Join(skills, ",")}},
		{name: "instill check-skills", command: "instill", args: []string{"check-skills"}},
		{name: "mise trust", command: "mise", args: []string{"trust"}},
		{name: "mise install", command: "mise", args: []string{"install"}},
		{name: "lefthook install", command: "mise", args: []string{"exec", "--", "lefthook", "install", "--force"}},
	}

	for _, step := range steps {
		if err := i.Runner.Run(ctx, targetDir, step.name, step.command, step.args...); err != nil {
			return failWithRecovery(targetDir, step.name, err)
		}
		if step.name == "lefthook install" {
			if err := repairBeadsHookChain(targetDir); err != nil {
				return failWithRecovery(targetDir, "lefthook chain repair", err)
			}
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

type skillManifest struct {
	Skills []string `json:"skills"`
}

func readManifestSkills(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest skillManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse skill manifest: %w", err)
	}

	if len(manifest.Skills) == 0 {
		return nil, fmt.Errorf("skill manifest is empty")
	}

	return manifest.Skills, nil
}

func repairBeadsHookChain(targetDir string) error {
	hooksDir := filepath.Join(targetDir, ".beads", "hooks")
	entries, err := os.ReadDir(hooksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".old") {
			continue
		}
		if err := repairBeadsHook(hooksDir, entry.Name()); err != nil {
			return err
		}
	}

	return nil
}

func repairBeadsHook(hooksDir string, oldName string) error {
	hookName := strings.TrimSuffix(oldName, ".old")
	hookPath := filepath.Join(hooksDir, hookName)
	lefthookPath := hookPath + ".lefthook"

	if _, err := os.Stat(hookPath); err != nil {
		return err
	}
	if _, err := os.Stat(lefthookPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(hookPath, lefthookPath); err != nil {
		return err
	}

	wrapper := chainedHookWrapper(hookName)
	return os.WriteFile(hookPath, []byte(wrapper), 0o755)
}

func chainedHookWrapper(hookName string) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
"$script_dir/%[1]s.old" "$@"
exec "$script_dir/%[1]s.lefthook" "$@"
`, hookName)
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
