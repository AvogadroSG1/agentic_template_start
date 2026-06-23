package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const remoteSmokeEnvVar = "MKPROJ_REMOTE_SMOKE"

type remoteSmokeHarness struct {
	binaryPath string
	buildDir   string
	env        []string
	owner      string
}

type remoteSmokeRun struct {
	Status       string `json:"status"`
	Conclusion   string `json:"conclusion"`
	WorkflowName string `json:"workflowName"`
	HeadBranch   string `json:"headBranch"`
	URL          string `json:"url"`
}

type remoteSmokeResult struct {
	repoName string
	run      remoteSmokeRun
}

func TestRemoteSmokeGHGoldenPathCreatesRepoAndSeesGreenCI(t *testing.T) {
	harness := requireRemoteSmokeHarness(t)

	result, err := runRemoteSmokeScenario(t, harness, false)
	if err != nil {
		t.Fatalf("runRemoteSmokeScenario() error = %v", err)
	}
	if result.run.Status != "completed" {
		t.Fatalf("workflow status = %q, want completed", result.run.Status)
	}
	if result.run.Conclusion != "success" {
		t.Fatalf("workflow conclusion = %q, want success (url: %s)", result.run.Conclusion, result.run.URL)
	}
	if result.run.URL == "" {
		t.Fatal("workflow url = empty, want run url")
	}
}

func TestRemoteSmokeGHDeletesRepoOnInjectedFailure(t *testing.T) {
	harness := requireRemoteSmokeHarness(t)

	if _, err := runRemoteSmokeScenario(t, harness, true); err == nil {
		t.Fatal("runRemoteSmokeScenario() error = nil, want injected failure")
	} else if !strings.Contains(err.Error(), "injected assertion failure") {
		t.Fatalf("runRemoteSmokeScenario() error = %v, want injected failure", err)
	}
}

func requireRemoteSmokeHarness(t *testing.T) remoteSmokeHarness {
	t.Helper()

	if os.Getenv(remoteSmokeEnvVar) != "1" {
		t.Skipf("set %s=1 to enable the opt-in remote smoke tier", remoteSmokeEnvVar)
	}

	buildDir, binaryPath := buildWalkingSkeletonBinary(t)
	t.Cleanup(func() {
		_ = makeTreeWritable(buildDir)
	})
	envRoot := filepath.Join(buildDir, "remote-smoke-env")
	binDir := filepath.Join(envRoot, "bin")
	for _, dir := range []string{
		binDir,
		filepath.Join(envRoot, "mise-data"),
		filepath.Join(envRoot, "mise-cache"),
		filepath.Join(envRoot, "mise-state"),
		filepath.Join(envRoot, "gopath"),
		filepath.Join(envRoot, "gomodcache"),
		filepath.Join(envRoot, "go-cache"),
		filepath.Join(envRoot, "golangci-cache"),
		filepath.Join(envRoot, "tokf"),
		filepath.Join(envRoot, "xdg-cache"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", dir, err)
		}
	}

	writeExecutable(t, filepath.Join(binDir, "lefthook"), `#!/usr/bin/env bash
set -euo pipefail
mise trust -a -y >/dev/null 2>&1 || true
exec mise x 'ubi:evilmartians/lefthook@latest' -- lefthook "$@"
`)

	authStatus := runRemoteSmokeCommand(t, nil, ".", "gh", "auth", "status")
	if !strings.Contains(authStatus, "delete_repo") {
		t.Skip("remote smoke teardown requires a GitHub token with delete_repo scope")
	}

	owner := strings.TrimSpace(runRemoteSmokeCommand(t, nil, ".", "gh", "api", "user", "--jq", ".login"))
	if owner == "" {
		t.Fatal("gh api user --jq .login returned an empty login")
	}

	pathEnv := binDir + string(os.PathListSeparator) + os.Getenv("PATH")
	env := append(os.Environ(),
		"PATH="+pathEnv,
		"TOKF_HOME="+filepath.Join(envRoot, "tokf"),
		"XDG_CACHE_HOME="+filepath.Join(envRoot, "xdg-cache"),
		"MISE_DATA_DIR="+filepath.Join(envRoot, "mise-data"),
		"MISE_CACHE_DIR="+filepath.Join(envRoot, "mise-cache"),
		"MISE_STATE_DIR="+filepath.Join(envRoot, "mise-state"),
		"GOPATH="+filepath.Join(envRoot, "gopath"),
		"GOMODCACHE="+filepath.Join(envRoot, "gomodcache"),
		"GOCACHE="+filepath.Join(envRoot, "go-cache"),
		"GOLANGCI_LINT_CACHE="+filepath.Join(envRoot, "golangci-cache"),
	)

	if _, err := exec.LookPath("gh"); err != nil {
		t.Fatalf("LookPath(gh) error = %v", err)
	}
	if _, err := exec.LookPath("mise"); err != nil {
		t.Fatalf("LookPath(mise) error = %v", err)
	}

	return remoteSmokeHarness{
		binaryPath: binaryPath,
		buildDir:   buildDir,
		env:        env,
		owner:      owner,
	}
}

func runRemoteSmokeScenario(t *testing.T, harness remoteSmokeHarness, injectFailure bool) (result remoteSmokeResult, err error) {
	t.Helper()

	repoName := fmt.Sprintf("mkproj-smoke-%d", time.Now().UTC().UnixNano())
	result.repoName = repoName
	targetDir := filepath.Join(harness.buildDir, repoName)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return result, fmt.Errorf("MkdirAll(%s): %w", targetDir, err)
	}

	defer func() {
		if cleanupErr := deleteRemoteRepo(t, harness.env, harness.owner, repoName); cleanupErr != nil {
			if err == nil {
				err = cleanupErr
			} else {
				err = fmt.Errorf("%w; cleanup: %v", err, cleanupErr)
			}
			return
		}
		if assertErr := assertRemoteRepoDeleted(t, harness.env, harness.owner, repoName); assertErr != nil {
			if err == nil {
				err = assertErr
			} else {
				err = fmt.Errorf("%w; cleanup verify: %v", err, assertErr)
			}
		}
	}()

	initCmd := exec.Command(harness.binaryPath,
		"init",
		"--project-name", "Sample App",
		"--language", "go",
		"--project-type", "cli",
		"--stack", "go-cli-cobra",
		"--author-name", "Ada Lovelace",
		"--author-email", "ada@example.com",
		"--github-user", harness.owner,
		"--remote", "gh",
	)
	initCmd.Dir = targetDir
	initCmd.Env = harness.env
	if output, runErr := initCmd.CombinedOutput(); runErr != nil {
		return result, fmt.Errorf("mkproj init --remote gh: %w\n%s", runErr, output)
	}

	if exists, existsErr := remoteRepoExists(t, harness.env, harness.owner, repoName); existsErr != nil {
		return result, existsErr
	} else if !exists {
		return result, fmt.Errorf("remote repo %s/%s was not created", harness.owner, repoName)
	}

	if injectFailure {
		return result, fmt.Errorf("injected assertion failure after remote creation")
	}

	run, waitErr := waitForRemoteWorkflow(t, harness.env, harness.owner, repoName)
	if waitErr != nil {
		return result, waitErr
	}
	result.run = run

	return result, nil
}

func waitForRemoteWorkflow(t *testing.T, env []string, owner string, repoName string) (remoteSmokeRun, error) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Minute)
	repo := owner + "/" + repoName
	for time.Now().Before(deadline) {
		output := runRemoteSmokeCommand(t, env, ".", "gh", "run", "list", "-R", repo, "--limit", "10", "--json", "status,conclusion,workflowName,headBranch,url")
		var runs []remoteSmokeRun
		if err := json.Unmarshal([]byte(output), &runs); err != nil {
			return remoteSmokeRun{}, fmt.Errorf("parse gh run list output: %w", err)
		}
		for _, run := range runs {
			if run.HeadBranch != "main" {
				continue
			}
			if run.Status == "completed" {
				return run, nil
			}
			time.Sleep(10 * time.Second)
			goto nextPoll
		}
		time.Sleep(5 * time.Second)

	nextPoll:
	}

	return remoteSmokeRun{}, fmt.Errorf("timed out waiting for remote workflow for %s/%s", owner, repoName)
}

func deleteRemoteRepo(t *testing.T, env []string, owner string, repoName string) error {
	t.Helper()

	repo := owner + "/" + repoName
	cmd := exec.Command("gh", "repo", "delete", repo, "--yes")
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.ToLower(string(output))
		if strings.Contains(text, "not found") || strings.Contains(text, "could not resolve") {
			return nil
		}
		return fmt.Errorf("gh repo delete %s: %w\n%s", repo, err, output)
	}

	return nil
}

func assertRemoteRepoDeleted(t *testing.T, env []string, owner string, repoName string) error {
	t.Helper()

	exists, err := remoteRepoExists(t, env, owner, repoName)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("remote repo %s/%s still exists after cleanup", owner, repoName)
	}

	return nil
}

func remoteRepoExists(t *testing.T, env []string, owner string, repoName string) (bool, error) {
	t.Helper()

	repo := owner + "/" + repoName
	cmd := exec.Command("gh", "repo", "view", repo, "--json", "name")
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err == nil {
		return true, nil
	}

	text := strings.ToLower(string(output))
	if strings.Contains(text, "not found") || strings.Contains(text, "could not resolve") {
		return false, nil
	}

	return false, fmt.Errorf("gh repo view %s: %w\n%s", repo, err, output)
}

func runRemoteSmokeCommand(t *testing.T, env []string, dir string, command string, args ...string) string {
	t.Helper()

	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = env
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s error = %v\n%s", command, strings.Join(args, " "), err, output)
	}

	return strings.TrimSpace(string(output))
}

func makeTreeWritable(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil {
			return nil
		}
		mode := info.Mode()
		if mode&0o200 != 0 {
			return nil
		}
		return os.Chmod(path, mode|0o200)
	})
}
