package test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var fallbackExecutableCandidates = map[string][]string{
	"mise": {
		"/opt/homebrew/bin/mise",
		"/usr/local/bin/mise",
	},
	"dotnet": {
		"/opt/homebrew/bin/dotnet",
		"/usr/local/share/dotnet/dotnet",
		"/usr/local/bin/dotnet",
	},
}

func commandEnv(envRoot string, command string) []string {
	switch filepath.Base(command) {
	case "mkproj":
		return ensureFallbackPATH(append(os.Environ(), "MKPROJ_RUNTIME_ROOT="+envRoot), "mise")
	case "mise", "dotnet":
		homeDir := filepath.Join(envRoot, "home")
		env := append(os.Environ(),
			"MKPROJ_RUNTIME_ROOT="+envRoot,
			"HOME="+homeDir,
			"GOCACHE="+filepath.Join(envRoot, "go-build"),
			"GOMODCACHE="+filepath.Join(envRoot, "go-mod"),
			"GOPATH="+filepath.Join(envRoot, "go"),
			"TOKF_HOME="+filepath.Join(envRoot, "tokf"),
			"TOKF_DB_PATH="+filepath.Join(envRoot, "tokf", "tracking.db"),
			"MISE_STATE_DIR="+filepath.Join(envRoot, "mise-state"),
			"MISE_DATA_DIR="+filepath.Join(envRoot, "mise-data"),
			"MISE_CACHE_DIR="+filepath.Join(envRoot, "mise-cache"),
			"XDG_CACHE_HOME="+filepath.Join(envRoot, "xdg-cache"),
			"XDG_CONFIG_HOME="+filepath.Join(homeDir, ".config"),
			"XDG_DATA_HOME="+filepath.Join(homeDir, ".local", "share"),
			"PIP_CACHE_DIR="+filepath.Join(envRoot, "pip-cache"),
			"UV_CACHE_DIR="+filepath.Join(envRoot, "uv-cache"),
			"DOTNET_CLI_HOME="+filepath.Join(envRoot, "dotnet-home"),
			"NUGET_PACKAGES="+filepath.Join(envRoot, "nuget-packages"),
		)

		return ensureFallbackPATH(env, filepath.Base(command))
	default:
		return os.Environ()
	}
}

func prepareCommandEnv(envRoot string, command string) error {
	switch filepath.Base(command) {
	case "mkproj":
		return nil
	case "mise", "dotnet":
		homeDir := filepath.Join(envRoot, "home")
		dirs := []string{
			homeDir,
			filepath.Join(homeDir, ".config"),
			filepath.Join(homeDir, ".local", "share"),
			filepath.Join(envRoot, "mise-state"),
			filepath.Join(envRoot, "mise-data"),
			filepath.Join(envRoot, "mise-cache"),
			filepath.Join(envRoot, "xdg-cache"),
			filepath.Join(envRoot, "pip-cache"),
			filepath.Join(envRoot, "uv-cache"),
			filepath.Join(envRoot, "dotnet-home"),
			filepath.Join(envRoot, "nuget-packages"),
		}

		for _, dir := range dirs {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}

		return nil
	default:
		return nil
	}
}

func ensureFallbackPATH(env []string, command string) []string {
	if command == "" || commandExistsOnPATH(command) {
		return env
	}

	pathValue := envValue(env, "PATH")
	fallbackDir := fallbackExecutableDir(command)
	if fallbackDir == "" || pathListContains(pathValue, fallbackDir) {
		return env
	}

	if pathValue == "" {
		return setEnvValue(env, "PATH", fallbackDir)
	}

	return setEnvValue(env, "PATH", fallbackDir+string(os.PathListSeparator)+pathValue)
}

func ensureProcessToolPath(commands ...string) {
	pathValue := os.Getenv("PATH")
	updated := pathValue
	for _, command := range commands {
		fallbackDir := fallbackExecutableDir(command)
		if fallbackDir == "" || pathListContains(updated, fallbackDir) {
			continue
		}
		if updated == "" {
			updated = fallbackDir
			continue
		}
		updated = fallbackDir + string(os.PathListSeparator) + updated
	}
	if updated != pathValue {
		_ = os.Setenv("PATH", updated)
	}
}

func stageToolBinary(binDir string, command string) (string, error) {
	sourcePath := readableCommandPath(command)
	if sourcePath == "" {
		return "", os.ErrNotExist
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", err
	}

	stagedPath := filepath.Join(binDir, filepath.Base(command))
	wrapper := fmt.Sprintf("#!/usr/bin/env bash\nchmod 755 %s >/dev/null 2>&1 || true\nexec %s \"$@\"\n", strconv.Quote(sourcePath), strconv.Quote(sourcePath))
	if err := os.WriteFile(stagedPath, []byte(wrapper), 0o755); err != nil {
		return "", err
	}

	return stagedPath, nil
}

func commandExistsOnPATH(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

func resolveCommandPath(command string) string {
	return resolveCommandPathForEnv(command, os.Environ())
}

func resolveCommandPathForEnv(command string, env []string) string {
	if path, ok := lookPathFromEnv(command, env); ok {
		return path
	}
	if path, err := exec.LookPath(command); err == nil {
		return path
	}

	for _, candidate := range fallbackExecutableCandidates[command] {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
			continue
		}
		return candidate
	}

	return command
}

func readableCommandPath(command string) string {
	if path, ok := lookReadablePathFromEnv(command, os.Environ()); ok {
		return path
	}

	for _, candidate := range fallbackExecutableCandidates[command] {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		return candidate
	}

	return ""
}

func lookPathFromEnv(command string, env []string) (string, bool) {
	if strings.ContainsRune(command, os.PathSeparator) {
		info, err := os.Stat(command)
		if err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return command, true
		}
		return "", false
	}

	pathValue := envValue(env, "PATH")
	for _, entry := range filepath.SplitList(pathValue) {
		if entry == "" {
			continue
		}
		candidate := filepath.Join(entry, command)
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
			continue
		}
		return candidate, true
	}

	return "", false
}

func lookReadablePathFromEnv(command string, env []string) (string, bool) {
	if strings.ContainsRune(command, os.PathSeparator) {
		info, err := os.Stat(command)
		if err == nil && !info.IsDir() {
			return command, true
		}
		return "", false
	}

	pathValue := envValue(env, "PATH")
	for _, entry := range filepath.SplitList(pathValue) {
		if entry == "" {
			continue
		}
		candidate := filepath.Join(entry, command)
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		return candidate, true
	}

	return "", false
}

func fallbackExecutableDir(command string) string {
	path := resolveCommandPath(command)
	if path == command {
		return ""
	}

	return filepath.Dir(path)
}

func pathListContains(pathValue string, want string) bool {
	for _, entry := range filepath.SplitList(pathValue) {
		if entry == want {
			return true
		}
	}

	return false
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}

	return ""
}

func setEnvValue(env []string, key string, value string) []string {
	prefix := key + "="
	updated := make([]string, 0, len(env)+1)
	replaced := false
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			updated = append(updated, prefix+value)
			replaced = true
			continue
		}
		updated = append(updated, entry)
	}
	if !replaced {
		updated = append(updated, prefix+value)
	}

	return updated
}
