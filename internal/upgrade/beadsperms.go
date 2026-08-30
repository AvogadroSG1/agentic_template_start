package upgrade

import (
	"os"
	"path/filepath"
)

// beadsDirPerm is the permission bd itself expects .beads to carry. bd init
// inherits the ambient umask (typically 0o755), which trips bd's own
// "recommended: 0700" warning on every scaffold until repaired.
const beadsDirPerm = 0o700

// EnsureBeadsDirPerms tightens <targetDir>/.beads to 0700 if it exists and
// does not already carry that mode. It is idempotent and a no-op when
// .beads is absent (e.g. bd init has not run yet, or failed).
func EnsureBeadsDirPerms(targetDir string) (repaired bool, err error) {
	beadsDir := filepath.Join(targetDir, ".beads")

	info, err := os.Stat(beadsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if info.Mode().Perm() == beadsDirPerm {
		return false, nil
	}

	if err := os.Chmod(beadsDir, beadsDirPerm); err != nil {
		return false, err
	}

	return true, nil
}
