// Package upgrade propagates managed static infrastructure files from the
// embedded template into existing forge-scaffolded repositories.
package upgrade

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const Version = 1

const versionFile = ".forge-infra-version"

type managedFile struct {
	src  string
	dest string
	mode os.FileMode
}

var managedFiles = []managedFile{
	{src: "templates/common/claude/hooks/guard", dest: ".claude/hooks/guard", mode: 0o755},
	{src: "templates/common/claude/hooks/secret-scan.sh", dest: ".claude/hooks/secret-scan.sh", mode: 0o755},
	{src: "templates/common/claude/settings.json", dest: ".claude/settings.json", mode: 0o644},
	{src: "templates/common/codex/hooks.json", dest: ".codex/hooks.json", mode: 0o644},
}

type Status struct {
	Version int
	OnDisk  int
	Stale   bool
	Updated []string
}

func Run(assets fs.FS, targetDir string, checkOnly bool) (Status, error) {
	if err := validateForgeRepo(targetDir); err != nil {
		return Status{}, err
	}

	onDisk := readOnDiskVersion(targetDir)
	status := Status{
		Version: Version,
		OnDisk:  onDisk,
		Stale:   onDisk < Version,
	}

	if checkOnly || !status.Stale {
		return status, nil
	}

	for _, mf := range managedFiles {
		data, err := fs.ReadFile(assets, mf.src)
		if err != nil {
			return Status{}, fmt.Errorf("read embedded %s: %w", mf.src, err)
		}
		dest := filepath.Join(targetDir, filepath.FromSlash(mf.dest))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return Status{}, fmt.Errorf("create directory for %s: %w", mf.dest, err)
		}
		if err := os.WriteFile(dest, data, mf.mode); err != nil {
			return Status{}, fmt.Errorf("write %s: %w", mf.dest, err)
		}
		if err := os.Chmod(dest, mf.mode); err != nil {
			return Status{}, fmt.Errorf("chmod %s: %w", mf.dest, err)
		}
		status.Updated = append(status.Updated, mf.dest)
	}

	versionContent := fmt.Sprintf("%d\n", Version)
	versionPath := filepath.Join(targetDir, versionFile)
	if err := os.WriteFile(versionPath, []byte(versionContent), 0o644); err != nil {
		return Status{}, fmt.Errorf("write %s: %w", versionFile, err)
	}

	return status, nil
}

func validateForgeRepo(targetDir string) error {
	markers := []string{
		filepath.Join(targetDir, ".claude", "hooks", "guard"),
		filepath.Join(targetDir, ".forge-infra-version"),
		filepath.Join(targetDir, ".claude", "settings.json"),
	}
	for _, marker := range markers {
		if _, err := os.Stat(marker); err == nil {
			return nil
		}
	}
	return fmt.Errorf("not a forge-managed repository (run from a project created with forge init)")
}

func readOnDiskVersion(targetDir string) int {
	data, err := os.ReadFile(filepath.Join(targetDir, versionFile))
	if err != nil {
		return 0
	}
	v, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return v
}
