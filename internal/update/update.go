package update

import (
	"context"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type CommandRunner interface {
	Run(ctx context.Context, dir string, step string, command string, args ...string) error
}

type GitRunner interface {
	Clone(ctx context.Context, repo string, dir string) error
	Checkout(ctx context.Context, dir string, ref string) error
}

type ExecGitRunner struct{}

type sourceRow struct {
	Kind  string       `yaml:"kind"`
	Steps []sourceStep `yaml:"steps"`
}

type sourceStep struct {
	Run      string   `yaml:"run"`
	Checkout string   `yaml:"checkout"`
	Ref      string   `yaml:"ref"`
	Strip    []string `yaml:"strip"`
}

func Run(ctx context.Context, assets fs.FS, stack string, runner CommandRunner, git GitRunner) error {
	if strings.TrimSpace(stack) == "" {
		return fmt.Errorf("update stack key is required")
	}
	if runner == nil {
		return fmt.Errorf("update command runner is required")
	}
	if git == nil {
		return fmt.Errorf("update git runner is required")
	}

	rows, err := loadSources(assets)
	if err != nil {
		return err
	}
	row, ok := rows[stack]
	if !ok {
		return fmt.Errorf("unknown update stack %q", stack)
	}
	if len(row.Steps) == 0 {
		return fmt.Errorf("stack %s has no update steps", stack)
	}

	workspaceRoot, err := os.MkdirTemp("", "mkproj-update-")
	if err != nil {
		return fmt.Errorf("create update workspace: %w", err)
	}
	defer os.RemoveAll(workspaceRoot)

	workspaceDir := filepath.Join(workspaceRoot, "workspace")
	for index, step := range row.Steps {
		stepLabel := fmt.Sprintf("stack %s step %d", stack, index+1)
		switch {
		case step.Run != "" && step.Checkout != "":
			return fmt.Errorf("%s must choose exactly one of run or checkout", stepLabel)
		case step.Run != "":
			if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
				return fmt.Errorf("%s: create workspace: %w", stepLabel, err)
			}
			command, args, err := splitCommandLine(step.Run)
			if err != nil {
				return fmt.Errorf("%s: parse run command %q: %w", stepLabel, step.Run, err)
			}
			if err := runner.Run(ctx, workspaceDir, stepLabel, command, args...); err != nil {
				return fmt.Errorf("%s command %q failed: %w", stepLabel, command, err)
			}
		case step.Checkout != "":
			if strings.TrimSpace(step.Ref) == "" {
				return fmt.Errorf("%s checkout %q is missing ref", stepLabel, step.Checkout)
			}
			if _, err := os.Stat(workspaceDir); err == nil {
				entries, readErr := os.ReadDir(workspaceDir)
				if readErr != nil {
					return fmt.Errorf("%s: read workspace: %w", stepLabel, readErr)
				}
				if len(entries) > 0 {
					return fmt.Errorf("%s cannot checkout into non-empty workspace", stepLabel)
				}
			}
			if err := git.Clone(ctx, step.Checkout, workspaceDir); err != nil {
				return fmt.Errorf("%s clone %q failed: %w", stepLabel, step.Checkout, err)
			}
			if err := git.Checkout(ctx, workspaceDir, step.Ref); err != nil {
				return fmt.Errorf("%s checkout ref %q failed: %w", stepLabel, step.Ref, err)
			}
			if err := stripPaths(workspaceDir, step.Strip); err != nil {
				return fmt.Errorf("%s strip paths failed: %w", stepLabel, err)
			}
		default:
			return fmt.Errorf("%s must define run or checkout", stepLabel)
		}
	}

	return nil
}

func (ExecGitRunner) Clone(ctx context.Context, repo string, dir string) error {
	normalizedRepo := normalizeRepo(repo)
	cmd := exec.CommandContext(ctx, "git", "clone", "--quiet", normalizedRepo, dir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (ExecGitRunner) Checkout(ctx context.Context, dir string, ref string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "checkout", "--quiet", ref)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func loadSources(assets fs.FS) (map[string]sourceRow, error) {
	data, err := fs.ReadFile(assets, "sources.yaml")
	if err != nil {
		return nil, fmt.Errorf("read embedded sources.yaml: %w", err)
	}

	rows := map[string]sourceRow{}
	if err := yaml.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("decode embedded sources.yaml: %w", err)
	}

	return rows, nil
}

func stripPaths(root string, patterns []string) error {
	for _, pattern := range patterns {
		fullPattern := filepath.Join(root, filepath.Clean(pattern))
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			return fmt.Errorf("glob %q: %w", pattern, err)
		}
		if len(matches) == 0 {
			continue
		}
		for _, match := range matches {
			if err := os.RemoveAll(match); err != nil {
				return fmt.Errorf("remove %q: %w", match, err)
			}
		}
	}
	return nil
}

func splitCommandLine(commandLine string) (string, []string, error) {
	var tokens []string
	var current strings.Builder
	var quote rune

	flush := func() {
		if current.Len() == 0 {
			return
		}
		tokens = append(tokens, current.String())
		current.Reset()
	}

	for _, r := range strings.TrimSpace(commandLine) {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t' || r == '\n':
			flush()
		default:
			current.WriteRune(r)
		}
	}
	if quote != 0 {
		return "", nil, fmt.Errorf("unterminated quote")
	}
	flush()
	if len(tokens) == 0 {
		return "", nil, fmt.Errorf("empty command")
	}

	return tokens[0], tokens[1:], nil
}

func normalizeRepo(repo string) string {
	if parsed, err := url.Parse(repo); err == nil && parsed.Scheme != "" {
		return repo
	}
	return "https://" + strings.TrimPrefix(repo, "https://")
}
