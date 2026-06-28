package delegate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandEnvUsesRepoLocalToolStateForMise(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sample-project")
	runtimeRoot := t.TempDir()
	t.Setenv("MKPROJ_RUNTIME_ROOT", runtimeRoot)
	env := commandEnv(dir, "mise install", "mise")

	assertEnvContains(t, env, "HOME="+filepath.Join(runtimeRoot, "tool-home"))
	assertEnvContains(t, env, "MISE_STATE_DIR="+filepath.Join(runtimeRoot, "mise-state"))
	assertEnvContains(t, env, "MISE_DATA_DIR="+filepath.Join(runtimeRoot, "mise-data"))
	assertEnvContains(t, env, "MISE_CACHE_DIR="+filepath.Join(runtimeRoot, "mise-cache"))
	assertEnvContains(t, env, "GOCACHE="+filepath.Join(runtimeRoot, "go-build"))
	assertEnvContains(t, env, "GOMODCACHE="+filepath.Join(runtimeRoot, "go-mod"))
	assertEnvContains(t, env, "GOPATH="+filepath.Join(runtimeRoot, "go"))
	assertEnvContains(t, env, "TOKF_HOME="+filepath.Join(runtimeRoot, "tokf"))
	assertEnvContains(t, env, "TOKF_DB_PATH="+filepath.Join(runtimeRoot, "tokf", "tracking.db"))
	assertEnvContains(t, env, "XDG_CACHE_HOME="+filepath.Join(runtimeRoot, "xdg-cache"))
	assertEnvContains(t, env, "XDG_CONFIG_HOME="+filepath.Join(runtimeRoot, "tool-home", ".config"))
	assertEnvContains(t, env, "XDG_DATA_HOME="+filepath.Join(runtimeRoot, "tool-home", ".local", "share"))
	assertEnvContains(t, env, "PIP_CACHE_DIR="+filepath.Join(runtimeRoot, "pip-cache"))
	assertEnvContains(t, env, "UV_CACHE_DIR="+filepath.Join(runtimeRoot, "uv-cache"))
	assertEnvContains(t, env, "DOTNET_CLI_HOME="+filepath.Join(runtimeRoot, "dotnet-home"))
	assertEnvContains(t, env, "NUGET_PACKAGES="+filepath.Join(runtimeRoot, "nuget-packages"))
}

func TestCommandEnvAddsFallbackMiseDirectoryToPATH(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sample-project")
	runtimeRoot := t.TempDir()
	fallbackRoot := t.TempDir()
	fallbackMise := filepath.Join(fallbackRoot, "mise")
	if err := os.WriteFile(fallbackMise, []byte("#!/bin/sh\n"), 0o755); err != nil {
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
	t.Setenv("MKPROJ_RUNTIME_ROOT", runtimeRoot)

	env := commandEnv(dir, "mise install", "mise")
	pathValue := envValue(env, "PATH")
	if pathValue == "" {
		t.Fatalf("PATH not found in env: %#v", env)
	}

	entries := strings.Split(pathValue, string(os.PathListSeparator))
	if len(entries) < 2 {
		t.Fatalf("PATH entries = %#v, want fallback dir plus existing PATH", entries)
	}
	if entries[0] != fallbackRoot {
		t.Fatalf("PATH first entry = %q, want %q in %#v", entries[0], fallbackRoot, entries)
	}
	if entries[1] != "/usr/bin" {
		t.Fatalf("PATH second entry = %q, want /usr/bin in %#v", entries[1], entries)
	}
}

func TestResolveCommandPathUsesFallbackExecutableWhenPATHMissesMise(t *testing.T) {
	fallbackRoot := t.TempDir()
	fallbackMise := filepath.Join(fallbackRoot, "mise")
	if err := os.WriteFile(fallbackMise, []byte("#!/bin/sh\n"), 0o755); err != nil {
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

	got := resolveCommandPath("mise")
	if got != fallbackMise {
		t.Fatalf("resolveCommandPath(mise) = %q, want %q", got, fallbackMise)
	}
}

func TestResolveCommandPathForEnvUsesProvidedPATH(t *testing.T) {
	fallbackRoot := t.TempDir()
	fallbackMise := filepath.Join(fallbackRoot, "mise")
	if err := os.WriteFile(fallbackMise, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", fallbackMise, err)
	}

	t.Setenv("PATH", "/usr/bin")

	got := resolveCommandPathForEnv("mise", []string{"PATH=" + fallbackRoot})
	if got != fallbackMise {
		t.Fatalf("resolveCommandPathForEnv(mise) = %q, want %q", got, fallbackMise)
	}
}

func TestCommandEnvLeavesHomeUntouchedForNonMiseCommands(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sample-project")
	env := commandEnv(dir, "bd init", "bd")

	assertEnvMissing(t, env, "HOME="+filepath.Join(dir, ".cache", "tool-home"))
	assertEnvMissing(t, env, "MISE_STATE_DIR="+filepath.Join(dir, ".cache", "mise-state"))
	assertEnvMissing(t, env, "MISE_DATA_DIR="+filepath.Join(dir, ".cache", "mise-data"))
	assertEnvMissing(t, env, "MISE_CACHE_DIR="+filepath.Join(dir, ".cache", "mise-cache"))
	assertEnvMissing(t, env, "XDG_CONFIG_HOME="+filepath.Join(dir, ".cache", "tool-home", ".config"))
	assertEnvMissing(t, env, "XDG_DATA_HOME="+filepath.Join(dir, ".cache", "tool-home", ".local", "share"))
}

func TestCommandEnvUsesRepoLocalMiseStateForGitCommit(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sample-project")
	runtimeRoot := t.TempDir()
	t.Setenv("MKPROJ_RUNTIME_ROOT", runtimeRoot)
	env := commandEnv(dir, "git commit", "git")

	assertEnvContains(t, env, "HOME="+filepath.Join(runtimeRoot, "tool-home"))
	assertEnvContains(t, env, "MISE_STATE_DIR="+filepath.Join(runtimeRoot, "mise-state"))
	assertEnvContains(t, env, "MISE_DATA_DIR="+filepath.Join(runtimeRoot, "mise-data"))
	assertEnvContains(t, env, "MISE_CACHE_DIR="+filepath.Join(runtimeRoot, "mise-cache"))
	assertEnvContains(t, env, "GOCACHE="+filepath.Join(runtimeRoot, "go-build"))
	assertEnvContains(t, env, "GOMODCACHE="+filepath.Join(runtimeRoot, "go-mod"))
	assertEnvContains(t, env, "GOPATH="+filepath.Join(runtimeRoot, "go"))
	assertEnvContains(t, env, "TOKF_HOME="+filepath.Join(runtimeRoot, "tokf"))
	assertEnvContains(t, env, "TOKF_DB_PATH="+filepath.Join(runtimeRoot, "tokf", "tracking.db"))
	assertEnvContains(t, env, "XDG_CACHE_HOME="+filepath.Join(runtimeRoot, "xdg-cache"))
	assertEnvContains(t, env, "XDG_CONFIG_HOME="+filepath.Join(runtimeRoot, "tool-home", ".config"))
	assertEnvContains(t, env, "XDG_DATA_HOME="+filepath.Join(runtimeRoot, "tool-home", ".local", "share"))
}

func TestCommandEnvLeavesGitInitUntouched(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sample-project")
	env := commandEnv(dir, "git init", "git")

	assertEnvMissing(t, env, "MISE_STATE_DIR="+filepath.Join(dir, ".cache", "mise-state"))
	assertEnvMissing(t, env, "GOCACHE="+filepath.Join(dir, ".cache", "go-build"))
	assertEnvMissing(t, env, "TOKF_HOME="+filepath.Join(dir, ".cache", "tokf"))
}

func TestCommandEnvLeavesToolStateUntouchedWithoutRuntimeRoot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sample-project")
	env := commandEnv(dir, "mise install", "mise")

	assertEnvMissing(t, env, "HOME="+filepath.Join(dir, ".cache", "tool-home"))
	assertEnvMissing(t, env, "MISE_STATE_DIR="+filepath.Join(dir, ".cache", "mise-state"))
	assertEnvMissing(t, env, "GOCACHE="+filepath.Join(dir, ".cache", "go-build"))
}

func assertEnvContains(t *testing.T, env []string, want string) {
	t.Helper()

	for _, entry := range env {
		if entry == want {
			return
		}
	}

	t.Fatalf("env missing %q in %#v", want, env)
}

func assertEnvMissing(t *testing.T, env []string, want string) {
	t.Helper()

	for _, entry := range env {
		if entry == want {
			t.Fatalf("env unexpectedly contains %q in %#v", want, env)
		}
	}
}
