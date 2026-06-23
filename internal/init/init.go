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
	Writer      scaffold.Writer
	Runner      delegate.Runner
	Interactive bool
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

	manifestSnapshot, err := captureFile(filepath.Join(targetDir, ".claude", "skill-manifest.json"))
	if err != nil {
		return failWithRecovery(targetDir, "skill manifest snapshot", err)
	}
	settingsSnapshot, err := captureFile(filepath.Join(targetDir, ".claude", "settings.local.json"))
	if err != nil {
		return failWithRecovery(targetDir, "settings.local.json snapshot", err)
	}

	instillInitArgs := []string{"init", "--force"}
	if !i.Interactive {
		skills, err := readManifestSkills(filepath.Join(targetDir, ".claude", "skill-manifest.json"))
		if err != nil {
			return failWithRecovery(targetDir, "skill manifest read", err)
		}
		instillInitArgs = append(instillInitArgs, "--skills", strings.Join(skills, ","))
	}

	steps := []struct {
		name    string
		command string
		args    []string
	}{
		{name: "bd init", command: "bd", args: []string{"init"}},
		{name: "instill init", command: "instill", args: instillInitArgs},
	}
	if i.Interactive {
		steps = append(steps, struct {
			name    string
			command string
			args    []string
		}{name: "instill pick-skills", command: "instill", args: []string{"pick-skills"}})
	}
	steps = append(steps, struct {
		name    string
		command string
		args    []string
	}{name: "instill check-skills", command: "instill", args: []string{"check-skills"}})
	steps = append(steps, struct {
		name    string
		command string
		args    []string
	}{name: "lefthook install", command: "lefthook", args: []string{"install", "--force"}})

	settingsHidden := false
	defer func() {
		if settingsHidden {
			_ = settingsSnapshot.Restore()
		}
	}()

	for _, step := range steps {
		if step.name == "instill init" {
			if err := settingsSnapshot.Hide(); err != nil {
				return failWithRecovery(targetDir, "settings.local.json hide", err)
			}
			settingsHidden = true
		}
		if err := i.Runner.Run(ctx, targetDir, step.name, step.command, step.args...); err != nil {
			return failWithRecovery(targetDir, step.name, err)
		}
		if step.name == "instill init" {
			if err := manifestSnapshot.Restore(); err != nil {
				return failWithRecovery(targetDir, "skill manifest restore", err)
			}
		}
		if step.name == "instill check-skills" && settingsHidden {
			if err := settingsSnapshot.Restore(); err != nil {
				return failWithRecovery(targetDir, "settings.local.json restore", err)
			}
			settingsHidden = false
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

type fileSnapshot struct {
	path string
	mode os.FileMode
	data []byte
}

func captureFile(path string) (fileSnapshot, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fileSnapshot{}, nil
		}
		return fileSnapshot{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fileSnapshot{}, err
	}

	return fileSnapshot{
		path: path,
		mode: info.Mode().Perm(),
		data: data,
	}, nil
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
		return nil, err
	}

	skills := make([]string, 0, len(manifest.Skills))
	for _, skill := range manifest.Skills {
		skill = strings.TrimSpace(skill)
		if skill == "" {
			continue
		}
		skills = append(skills, skill)
	}
	if len(skills) == 0 {
		return nil, fmt.Errorf("skill manifest %s did not contain any skills", path)
	}

	return skills, nil
}

func (s fileSnapshot) Restore() error {
	if s.path == "" {
		return nil
	}

	return os.WriteFile(s.path, s.data, s.mode)
}

func (s fileSnapshot) Hide() error {
	if s.path == "" {
		return nil
	}
	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
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
