package delegate

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var fallbackExecutableCandidates = map[string][]string{
	"mise": {
		"/opt/homebrew/bin/mise",
		"/usr/local/bin/mise",
	},
}

func commandEnv(dir string, step string, command string) []string {
	runtimeRoot := repoLocalRuntimeRoot()
	if runtimeRoot == "" || !needsRepoLocalToolState(step, command) {
		return os.Environ()
	}

	env := append(os.Environ(), repoLocalRuntimeEnv(runtimeRoot)...)
	env = ensureFallbackPATH(env, filepath.Base(command))

	if !needsRepoLocalHome(step, command) {
		return env
	}

	homeDir := filepath.Join(runtimeRoot, "tool-home")
	return append(env, miseHomeEnv(homeDir)...)
}

func prepareCommandEnv(dir string, step string, command string) error {
	runtimeRoot := repoLocalRuntimeRoot()
	if runtimeRoot == "" || !needsRepoLocalToolState(step, command) {
		return nil
	}

	dirs := repoLocalRuntimeDirs(runtimeRoot)

	if needsRepoLocalHome(step, command) {
		homeDir := filepath.Join(runtimeRoot, "tool-home")
		dirs = append(dirs,
			homeDir,
			filepath.Join(homeDir, ".config"),
			filepath.Join(homeDir, ".local", "share"),
		)
	}

	for _, path := range dirs {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}

	return nil
}

func needsRepoLocalToolState(step string, command string) bool {
	switch command {
	case "go", "mise":
		return true
	case "git":
		return step == "git commit" || step == "git push"
	default:
		return false
	}
}

func needsRepoLocalHome(step string, command string) bool {
	return command == "mise" || command == "git" && (step == "git commit" || step == "git push")
}

func repoLocalRuntimeRoot() string {
	return os.Getenv("MKPROJ_RUNTIME_ROOT")
}

func repoLocalRuntimeEnv(runtimeRoot string) []string {
	return []string{
		"GOCACHE=" + filepath.Join(runtimeRoot, "go-build"),
		"GOMODCACHE=" + filepath.Join(runtimeRoot, "go-mod"),
		"GOPATH=" + filepath.Join(runtimeRoot, "go"),
		"TOKF_HOME=" + filepath.Join(runtimeRoot, "tokf"),
		"TOKF_DB_PATH=" + filepath.Join(runtimeRoot, "tokf", "tracking.db"),
		"MISE_STATE_DIR=" + filepath.Join(runtimeRoot, "mise-state"),
		"MISE_DATA_DIR=" + filepath.Join(runtimeRoot, "mise-data"),
		"MISE_CACHE_DIR=" + filepath.Join(runtimeRoot, "mise-cache"),
		"XDG_CACHE_HOME=" + filepath.Join(runtimeRoot, "xdg-cache"),
		"PIP_CACHE_DIR=" + filepath.Join(runtimeRoot, "pip-cache"),
		"UV_CACHE_DIR=" + filepath.Join(runtimeRoot, "uv-cache"),
		"DOTNET_CLI_HOME=" + filepath.Join(runtimeRoot, "dotnet-home"),
		"NUGET_PACKAGES=" + filepath.Join(runtimeRoot, "nuget-packages"),
	}
}

func repoLocalRuntimeDirs(runtimeRoot string) []string {
	return []string{
		filepath.Join(runtimeRoot, "go-build"),
		filepath.Join(runtimeRoot, "go-mod"),
		filepath.Join(runtimeRoot, "go"),
		filepath.Join(runtimeRoot, "tokf"),
		filepath.Join(runtimeRoot, "mise-state"),
		filepath.Join(runtimeRoot, "mise-data"),
		filepath.Join(runtimeRoot, "mise-cache"),
		filepath.Join(runtimeRoot, "xdg-cache"),
		filepath.Join(runtimeRoot, "pip-cache"),
		filepath.Join(runtimeRoot, "uv-cache"),
		filepath.Join(runtimeRoot, "dotnet-home"),
		filepath.Join(runtimeRoot, "nuget-packages"),
	}
}

func miseHomeEnv(homeDir string) []string {
	return []string{
		"HOME=" + homeDir,
		"XDG_CONFIG_HOME=" + filepath.Join(homeDir, ".config"),
		"XDG_DATA_HOME=" + filepath.Join(homeDir, ".local", "share"),
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
