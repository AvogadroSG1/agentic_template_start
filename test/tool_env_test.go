package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandEnvAddsFallbackMiseDirectoryToPATHForMkproj(t *testing.T) {
	envRoot := t.TempDir()
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

	env := commandEnv(envRoot, "mkproj")
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

func TestPrepareCommandEnvCreatesRuntimeToolDirectories(t *testing.T) {
	envRoot := t.TempDir()
	if err := prepareCommandEnv(envRoot, "mise"); err != nil {
		t.Fatalf("prepareCommandEnv() error = %v", err)
	}

	for _, dir := range []string{
		filepath.Join(envRoot, "home"),
		filepath.Join(envRoot, "home", ".config"),
		filepath.Join(envRoot, "home", ".local", "share"),
		filepath.Join(envRoot, "mise-state"),
		filepath.Join(envRoot, "mise-data"),
		filepath.Join(envRoot, "mise-cache"),
		filepath.Join(envRoot, "xdg-cache"),
		filepath.Join(envRoot, "pip-cache"),
		filepath.Join(envRoot, "uv-cache"),
		filepath.Join(envRoot, "dotnet-home"),
		filepath.Join(envRoot, "nuget-packages"),
	} {
		if _, err := os.Stat(dir); err != nil {
			t.Fatalf("Stat(%s) error = %v", dir, err)
		}
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

func TestEnsureProcessToolPathPrependsFallbackDirectory(t *testing.T) {
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

	ensureProcessToolPath("mise")

	pathValue := os.Getenv("PATH")
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

func TestStageToolBinaryCopiesReadableSourceAsExecutable(t *testing.T) {
	sourceRoot := t.TempDir()
	sourceMise := filepath.Join(sourceRoot, "mise")
	if err := os.WriteFile(sourceMise, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", sourceMise, err)
	}

	previous := fallbackExecutableCandidates
	fallbackExecutableCandidates = map[string][]string{
		"mise": {sourceMise},
	}
	t.Cleanup(func() {
		fallbackExecutableCandidates = previous
	})

	stagedDir := t.TempDir()
	stagedPath, err := stageToolBinary(stagedDir, "mise")
	if err != nil {
		t.Fatalf("stageToolBinary() error = %v", err)
	}

	if stagedPath != filepath.Join(stagedDir, "mise") {
		t.Fatalf("staged path = %q, want %q", stagedPath, filepath.Join(stagedDir, "mise"))
	}

	info, err := os.Stat(stagedPath)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", stagedPath, err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("staged mode = %v, want executable", info.Mode())
	}
}
