package scaffold

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"mkproj/internal/project"
)

type Writer struct {
	Assets fs.FS
}

func (w Writer) Write(targetDir string, vars project.Variables) error {
	if err := ensureWritableDir(targetDir); err != nil {
		return err
	}

	if err := w.copyTree("templates/common", targetDir, vars); err != nil {
		return err
	}

	stackRoot := filepath.ToSlash(filepath.Join("templates/golden", vars.Stack))
	if err := w.copyTree(stackRoot, targetDir, vars); err != nil {
		return err
	}

	overlayRoot := filepath.ToSlash(filepath.Join(stackRoot, ".mkproj-overlay"))
	if err := w.copyTree(overlayRoot, targetDir, vars); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	if err := w.writeGitIgnore(targetDir, vars.Language); err != nil {
		return err
	}

	claudePath := filepath.Join(targetDir, "CLAUDE.md")
	if err := os.RemoveAll(claudePath); err != nil {
		return err
	}
	if err := os.Symlink("AGENTS.md", claudePath); err != nil {
		return err
	}

	return nil
}

func ensureEmptyDir(targetDir string) error {
	info, err := os.Stat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(targetDir, 0o755)
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

func ensureWritableDir(targetDir string) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == ".DS_Store" || entry.Name() == ".git" {
			continue
		}
		return fmt.Errorf("directory not empty: %s", targetDir)
	}

	return nil
}

func (w Writer) copyTree(root string, targetDir string, vars project.Variables) error {
	sub, err := fs.Sub(w.Assets, root)
	if err != nil {
		return err
	}

	return fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == "." {
			return nil
		}
		if path == ".mkproj-overlay" {
			return fs.SkipDir
		}
		if strings.HasPrefix(path, ".mkproj-overlay/") {
			return fs.SkipDir
		}

		targetPath := filepath.Join(targetDir, mapOutputPath(path))
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		mode := fs.FileMode(0o644)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		data, err := fs.ReadFile(sub, path)
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, ".tmpl") {
			rendered, renderErr := renderTemplate(path, data, vars)
			if renderErr != nil {
				return renderErr
			}
			targetPath = strings.TrimSuffix(targetPath, ".tmpl")
			return os.WriteFile(targetPath, rendered, mode)
		}

		if filepath.Base(path) == "guard" || filepath.Base(path) == "secret-scan.sh" {
			mode = 0o755
		}

		return os.WriteFile(targetPath, data, mode)
	})
}

func mapOutputPath(path string) string {
	switch {
	case strings.HasPrefix(path, "claude/"):
		return filepath.Join(".claude", strings.TrimPrefix(path, "claude/"))
	case strings.HasPrefix(path, "codex/"):
		return filepath.Join(".codex", strings.TrimPrefix(path, "codex/"))
	default:
		return path
	}
}

func renderTemplate(name string, data []byte, vars project.Variables) ([]byte, error) {
	tmpl, err := template.New(name).Option("missingkey=error").Parse(string(data))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (w Writer) writeGitIgnore(targetDir string, language string) error {
	base, err := fs.ReadFile(w.Assets, "templates/common/gitignore.base")
	if err != nil {
		return err
	}

	stem, err := gitignoreStem(language)
	if err != nil {
		return err
	}
	lang, err := fs.ReadFile(w.Assets, filepath.ToSlash(filepath.Join("templates/gitignore", stem+".gitignore")))
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	buf.Write(base)
	if len(base) > 0 && base[len(base)-1] != '\n' {
		buf.WriteByte('\n')
	}
	fmt.Fprintf(&buf, "# ===== %s gitignore =====\n", strings.ToLower(stem))
	buf.Write(lang)

	return os.WriteFile(filepath.Join(targetDir, ".gitignore"), buf.Bytes(), 0o644)
}

func gitignoreStem(language string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "go":
		return "Go", nil
	case "python":
		return "Python", nil
	case "csharp":
		return "VisualStudio", nil
	default:
		return "", fmt.Errorf("unsupported language %q", language)
	}
}
