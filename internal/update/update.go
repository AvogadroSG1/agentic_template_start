package update

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"mkproj/internal/project"

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
	Kind      string          `yaml:"kind"`
	GitIgnore string          `yaml:"gitignore"`
	Steps     []sourceStep    `yaml:"steps"`
	Normalize []normalizeRule `yaml:"normalize"`
	Resolved  resolvedSource  `yaml:"resolved"`
}

type sourceStep struct {
	Run      string   `yaml:"run,omitempty"`
	Checkout string   `yaml:"checkout,omitempty"`
	Ref      string   `yaml:"ref,omitempty"`
	Strip    []string `yaml:"strip,omitempty"`
}

type normalizeRule struct {
	Type  string   `yaml:"type"`
	Paths []string `yaml:"paths,omitempty"`
}

type resolvedSource struct {
	Ref      string `yaml:"ref"`
	Captured string `yaml:"captured"`
}

type seamCheckReport struct {
	Collisions []string
}

var guidPattern = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)

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

	repoRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	vars, err := representativeVariables(stack, row)
	if err != nil {
		return err
	}

	workspaceRoot, err := os.MkdirTemp("", "mkproj-update-")
	if err != nil {
		return fmt.Errorf("create update workspace: %w", err)
	}
	defer os.RemoveAll(workspaceRoot)

	workspaceDir := filepath.Join(workspaceRoot, representativeWorkspaceName(vars))
	for index, step := range row.Steps {
		stepLabel := fmt.Sprintf("stack %s step %d", stack, index+1)
		switch {
		case step.Run != "" && step.Checkout != "":
			return fmt.Errorf("%s must choose exactly one of run or checkout", stepLabel)
		case step.Run != "":
			if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
				return fmt.Errorf("%s: create workspace: %w", stepLabel, err)
			}
			renderedCommand, err := renderString(step.Run, vars)
			if err != nil {
				return fmt.Errorf("%s: render run command %q: %w", stepLabel, step.Run, err)
			}
			command, args, err := splitCommandLine(renderedCommand)
			if err != nil {
				return fmt.Errorf("%s: parse run command %q: %w", stepLabel, renderedCommand, err)
			}
			if err := runner.Run(ctx, workspaceDir, stepLabel, command, args...); err != nil {
				return fmt.Errorf("%s command %q failed: %w", stepLabel, command, err)
			}
		case step.Checkout != "":
			repo, err := renderString(step.Checkout, vars)
			if err != nil {
				return fmt.Errorf("%s: render checkout repo %q: %w", stepLabel, step.Checkout, err)
			}
			ref, err := renderString(step.Ref, vars)
			if err != nil {
				return fmt.Errorf("%s: render checkout ref %q: %w", stepLabel, step.Ref, err)
			}
			if strings.TrimSpace(ref) == "" {
				return fmt.Errorf("%s checkout %q is missing ref", stepLabel, repo)
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
			if err := git.Clone(ctx, repo, workspaceDir); err != nil {
				return fmt.Errorf("%s clone %q failed: %w", stepLabel, repo, err)
			}
			if err := git.Checkout(ctx, workspaceDir, ref); err != nil {
				return fmt.Errorf("%s checkout ref %q failed: %w", stepLabel, ref, err)
			}
			if err := stripPaths(workspaceDir, step.Strip); err != nil {
				return fmt.Errorf("%s strip paths failed: %w", stepLabel, err)
			}
		default:
			return fmt.Errorf("%s must define run or checkout", stepLabel)
		}
	}

	stackRoot := filepath.Join(repoRoot, "templates", "golden", stack)
	refreshedVanilla := filepath.Join(workspaceRoot, "refreshed")
	if err := snapshotVanilla(workspaceDir, refreshedVanilla, stackRoot, row.Normalize, vars); err != nil {
		return err
	}
	vanillaChanged, err := vanillaLayerChanged(stackRoot, refreshedVanilla)
	if err != nil {
		return err
	}

	seamReport, err := checkOverlaySeam(stackRoot, refreshedVanilla)
	if err != nil {
		return err
	}
	if err := replaceVanillaLayer(stackRoot, refreshedVanilla); err != nil {
		return err
	}
	if err := repinMutableSources(filepath.Join(repoRoot, "sources.yaml"), stack, row, vanillaChanged); err != nil {
		return err
	}
	if len(seamReport.Collisions) > 0 {
		fmt.Fprintf(os.Stderr, "warning: overlay collision(s) for stack %s: %s\n", stack, strings.Join(seamReport.Collisions, ", "))
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

func representativeVariables(stack string, row sourceRow) (project.Variables, error) {
	language, err := languageForRow(stack, row)
	if err != nil {
		return project.Variables{}, err
	}

	return project.ResolveVariables(project.Input{
		ProjectName: "Mkproj Template Fixture",
		Language:    language,
		ProjectType: row.Kind,
		Stack:       stack,
		AuthorName:  "Mkproj Maintainer",
		AuthorEmail: "maintainer@example.com",
		Remote:      project.RemoteNone,
	})
}

func languageForRow(stack string, row sourceRow) (string, error) {
	switch strings.ToLower(strings.TrimSpace(row.GitIgnore)) {
	case "go":
		return "go", nil
	case "python":
		return "python", nil
	case "visualstudio":
		return "csharp", nil
	}

	parts := strings.SplitN(strings.TrimSpace(stack), "-", 2)
	if len(parts) > 0 && parts[0] != "" {
		return parts[0], nil
	}

	return "", fmt.Errorf("infer language for stack %q", stack)
}

func representativeWorkspaceName(vars project.Variables) string {
	switch vars.Language {
	case "csharp":
		return vars.CSharpNamespace
	case "python":
		return vars.PythonPackage
	default:
		return vars.BdPrefix
	}
}

func renderString(input string, vars project.Variables) (string, error) {
	tmpl, err := template.New("update").Option("missingkey=error").Parse(input)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func snapshotVanilla(workspaceDir string, targetDir string, committedStackDir string, rules []normalizeRule, vars project.Variables) error {
	templateContract, err := loadTemplateContract(committedStackDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create refreshed vanilla dir: %w", err)
	}

	return filepath.WalkDir(workspaceDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(workspaceDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		committedRelPath := filepath.ToSlash(relPath)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(targetDir, filepath.FromSlash(committedRelPath)), 0o755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read refreshed file %q: %w", committedRelPath, err)
		}
		normalized, err := normalizeFile(committedRelPath, data, rules)
		if err != nil {
			return fmt.Errorf("normalize %q: %w", committedRelPath, err)
		}
		templated := applyTemplatePlaceholders(normalized, vars)
		if contractPath, ok := templateContract[renderedCatalogPath(committedRelPath)]; ok {
			committedRelPath = contractPath
		} else if !bytes.Equal(templated, normalized) {
			committedRelPath += ".tmpl"
		}

		targetPath := filepath.Join(targetDir, filepath.FromSlash(committedRelPath))
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("prepare refreshed file %q: %w", committedRelPath, err)
		}
		if err := os.WriteFile(targetPath, templated, 0o644); err != nil {
			return fmt.Errorf("write refreshed file %q: %w", committedRelPath, err)
		}
		return nil
	})
}

func loadTemplateContract(committedStackDir string) (map[string]string, error) {
	contract := map[string]string{}
	if _, err := os.Stat(committedStackDir); errors.Is(err, fs.ErrNotExist) {
		return contract, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat committed stack dir: %w", err)
	}

	err := filepath.WalkDir(committedStackDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(committedStackDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		relPath = filepath.ToSlash(relPath)
		if relPath == ".mkproj-overlay" {
			return fs.SkipDir
		}
		if strings.HasPrefix(relPath, ".mkproj-overlay/") || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(relPath, ".tmpl") {
			contract[renderedCatalogPath(relPath)] = relPath
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk committed stack dir: %w", err)
	}

	return contract, nil
}

func normalizeFile(relPath string, data []byte, rules []normalizeRule) ([]byte, error) {
	normalized := append([]byte(nil), data...)
	for _, rule := range rules {
		switch strings.TrimSpace(rule.Type) {
		case "", "sort_files":
			continue
		case "line_endings":
			normalized = bytes.ReplaceAll(normalized, []byte("\r\n"), []byte("\n"))
			normalized = bytes.ReplaceAll(normalized, []byte("\r"), []byte("\n"))
		case "trailing_newline":
			normalized = bytes.TrimRight(normalized, "\n")
			if len(normalized) > 0 {
				normalized = append(normalized, '\n')
			}
		case "replace_guid":
			if matchesAnyPath(relPath, rule.Paths) {
				normalized = guidPattern.ReplaceAll(normalized, []byte("00000000-0000-0000-0000-000000000000"))
			}
		default:
			return nil, fmt.Errorf("unsupported normalize rule %q", rule.Type)
		}
	}

	return normalized, nil
}

func matchesAnyPath(relPath string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, filepath.Base(relPath))
		if err == nil && matched {
			return true
		}
		matched, err = filepath.Match(pattern, relPath)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func applyTemplatePlaceholders(data []byte, vars project.Variables) []byte {
	type replacement struct {
		value       string
		placeholder string
	}

	seen := map[string]struct{}{}
	replacements := []replacement{
		{value: vars.ModulePath, placeholder: "{{.ModulePath}}"},
		{value: vars.ProjectName, placeholder: "{{.ProjectName}}"},
		{value: vars.BdPrefix, placeholder: "{{.BdPrefix}}"},
		{value: vars.PythonPackage, placeholder: "{{.PythonPackage}}"},
		{value: vars.CSharpNamespace, placeholder: "{{.CSharpNamespace}}"},
		{value: vars.AuthorName, placeholder: "{{.AuthorName}}"},
		{value: vars.AuthorEmail, placeholder: "{{.AuthorEmail}}"},
	}

	filtered := make([]replacement, 0, len(replacements))
	for _, candidate := range replacements {
		if candidate.value == "" {
			continue
		}
		if _, ok := seen[candidate.value]; ok {
			continue
		}
		seen[candidate.value] = struct{}{}
		filtered = append(filtered, candidate)
	}
	sort.Slice(filtered, func(i int, j int) bool {
		return len(filtered[i].value) > len(filtered[j].value)
	})

	templated := append([]byte(nil), data...)
	for _, candidate := range filtered {
		templated = bytes.ReplaceAll(templated, []byte(candidate.value), []byte(candidate.placeholder))
	}
	return templated
}

func checkOverlaySeam(committedStackDir string, refreshedVanilla string) (seamCheckReport, error) {
	overlayDir := filepath.Join(committedStackDir, ".mkproj-overlay")
	if _, err := os.Stat(overlayDir); errors.Is(err, fs.ErrNotExist) {
		return seamCheckReport{}, nil
	} else if err != nil {
		return seamCheckReport{}, fmt.Errorf("stat overlay dir: %w", err)
	}

	fileOutputs := map[string]struct{}{}
	dirOutputs := map[string]struct{}{".": {}}
	if err := filepath.WalkDir(refreshedVanilla, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(refreshedVanilla, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		rendered := renderedCatalogPath(filepath.ToSlash(relPath))
		if d.IsDir() {
			dirOutputs[rendered] = struct{}{}
			return nil
		}

		fileOutputs[rendered] = struct{}{}
		for parent := filepath.ToSlash(filepath.Dir(rendered)); parent != "." && parent != "/"; parent = filepath.ToSlash(filepath.Dir(parent)) {
			dirOutputs[parent] = struct{}{}
		}
		return nil
	}); err != nil {
		return seamCheckReport{}, fmt.Errorf("walk refreshed vanilla: %w", err)
	}

	var orphans []string
	var collisions []string
	err := filepath.WalkDir(overlayDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(overlayDir, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		rendered := renderedCatalogPath(relPath)
		parent := filepath.ToSlash(filepath.Dir(rendered))
		if parent == "" {
			parent = "."
		}
		if _, ok := dirOutputs[parent]; !ok {
			orphans = append(orphans, relPath)
		}
		if _, ok := fileOutputs[rendered]; ok {
			collisions = append(collisions, relPath)
		}
		return nil
	})
	if err != nil {
		return seamCheckReport{}, fmt.Errorf("walk overlay dir: %w", err)
	}
	if len(orphans) > 0 {
		sort.Strings(orphans)
		return seamCheckReport{}, fmt.Errorf("orphan overlay path(s): %s", strings.Join(orphans, ", "))
	}

	sort.Strings(collisions)
	return seamCheckReport{Collisions: collisions}, nil
}

func vanillaLayerChanged(committedStackDir string, refreshedVanilla string) (bool, error) {
	committedSnapshot, err := captureVanillaSnapshot(committedStackDir)
	if err != nil {
		return false, err
	}
	refreshedSnapshot, err := captureVanillaSnapshot(refreshedVanilla)
	if err != nil {
		return false, err
	}
	if len(committedSnapshot) != len(refreshedSnapshot) {
		return true, nil
	}
	for path, contents := range committedSnapshot {
		if refreshedSnapshot[path] != contents {
			return true, nil
		}
	}
	return false, nil
}

func captureVanillaSnapshot(root string) (map[string]string, error) {
	snapshot := map[string]string{}
	if _, err := os.Stat(root); errors.Is(err, fs.ErrNotExist) {
		return snapshot, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat vanilla root: %w", err)
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		relPath = filepath.ToSlash(relPath)
		if relPath == ".mkproj-overlay" {
			return fs.SkipDir
		}
		if strings.HasPrefix(relPath, ".mkproj-overlay/") || d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read snapshot path %q: %w", relPath, err)
		}
		snapshot[relPath] = string(data)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk vanilla root: %w", err)
	}
	return snapshot, nil
}

func repinMutableSources(path string, stack string, row sourceRow, vanillaChanged bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read mutable sources.yaml: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("decode mutable sources.yaml: %w", err)
	}
	root := ensureMappingDocument(&doc)
	index := findMappingKey(root, stack)
	existingCaptured := ""
	if index >= 0 {
		existingCaptured = extractCaptured(root.Content[index+1])
	}

	updatedRow := row
	switch {
	case vanillaChanged:
		updatedRow.Resolved.Captured = currentDateString()
	case existingCaptured != "":
		updatedRow.Resolved.Captured = existingCaptured
	}
	rowNode, err := marshalRowNode(updatedRow)
	if err != nil {
		return err
	}
	if index >= 0 {
		root.Content[index+1] = &rowNode
	} else {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: stack},
			&rowNode,
		)
	}

	updatedData, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("encode mutable sources.yaml: %w", err)
	}
	if bytes.Equal(data, updatedData) {
		return nil
	}
	if err := os.WriteFile(path, updatedData, 0o644); err != nil {
		return fmt.Errorf("write mutable sources.yaml: %w", err)
	}
	return nil
}

func ensureMappingDocument(doc *yaml.Node) *yaml.Node {
	if doc.Kind == 0 {
		doc.Kind = yaml.DocumentNode
	}
	if len(doc.Content) == 0 || doc.Content[0] == nil {
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}
	root := doc.Content[0]
	if root.Kind == 0 {
		root.Kind = yaml.MappingNode
		root.Tag = "!!map"
	}
	return root
}

func findMappingKey(node *yaml.Node, key string) int {
	if node == nil || node.Kind != yaml.MappingNode {
		return -1
	}
	for index := 0; index < len(node.Content)-1; index += 2 {
		if node.Content[index].Value == key {
			return index
		}
	}
	return -1
}

func extractCaptured(node *yaml.Node) string {
	resolvedIndex := findMappingKey(node, "resolved")
	if resolvedIndex < 0 {
		return ""
	}
	resolved := node.Content[resolvedIndex+1]
	capturedIndex := findMappingKey(resolved, "captured")
	if capturedIndex < 0 {
		return ""
	}
	return strings.TrimSpace(resolved.Content[capturedIndex+1].Value)
}

func marshalRowNode(row sourceRow) (yaml.Node, error) {
	encoded, err := yaml.Marshal(row)
	if err != nil {
		return yaml.Node{}, fmt.Errorf("encode source row: %w", err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(encoded, &doc); err != nil {
		return yaml.Node{}, fmt.Errorf("decode source row: %w", err)
	}
	if len(doc.Content) == 0 || doc.Content[0] == nil {
		return yaml.Node{}, fmt.Errorf("encoded source row was empty")
	}
	return *doc.Content[0], nil
}

func currentDateString() string {
	return time.Now().Format("2006-01-02")
}

func replaceVanillaLayer(committedStackDir string, refreshedVanilla string) error {
	if err := os.MkdirAll(committedStackDir, 0o755); err != nil {
		return fmt.Errorf("ensure committed stack dir: %w", err)
	}

	entries, err := os.ReadDir(committedStackDir)
	if err != nil {
		return fmt.Errorf("read committed stack dir: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() == ".mkproj-overlay" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(committedStackDir, entry.Name())); err != nil {
			return fmt.Errorf("remove stale vanilla path %q: %w", entry.Name(), err)
		}
	}

	return filepath.WalkDir(refreshedVanilla, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(refreshedVanilla, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		targetPath := filepath.Join(committedStackDir, relPath)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read refreshed path %q: %w", relPath, err)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("prepare committed path %q: %w", relPath, err)
		}
		if err := os.WriteFile(targetPath, data, 0o644); err != nil {
			return fmt.Errorf("write committed path %q: %w", relPath, err)
		}
		return nil
	})
}

func renderedCatalogPath(path string) string {
	path = strings.TrimSuffix(filepath.ToSlash(path), ".tmpl")
	switch {
	case strings.HasPrefix(path, "claude/"):
		return ".claude/" + strings.TrimPrefix(path, "claude/")
	case strings.HasPrefix(path, "codex/"):
		return ".codex/" + strings.TrimPrefix(path, "codex/")
	default:
		return path
	}
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
