package allowlist

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"text/template"
)

const Version = 1

const (
	beginMarker = "// BEGIN MKPROJ ALLOW v:"
	endMarker   = "// END MKPROJ ALLOW"
)

type Status struct {
	CurrentVersion int
	Embedded       int
	Stale          bool
}

func Sync(path string, block string, checkOnly bool) (Status, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Status{}, err
	}

	status, err := Detect(string(data))
	if err != nil {
		return Status{}, err
	}
	if checkOnly {
		return status, nil
	}

	text := string(data)
	startIdx := strings.Index(text, beginMarker)
	endIdx := strings.Index(text, endMarker)
	if startIdx == -1 || endIdx == -1 || endIdx < startIdx {
		return Status{}, fmt.Errorf("managed block markers not found in %s", path)
	}

	lineStart := strings.LastIndex(text[:startIdx], "\n") + 1
	afterEnd := endIdx + len(endMarker)
	lineEnd := afterEnd
	for lineEnd < len(text) && text[lineEnd] != '\n' {
		lineEnd++
	}

	indent := strings.Repeat(" ", startIdx-lineStart)
	beginLine := fmt.Sprintf(`%s"%s%d",`, indent, beginMarker, Version)
	endLine := fmt.Sprintf(`%s"%s",`, indent, endMarker)
	replacement := beginLine + "\n" + block + "\n" + endLine

	updated := text[:lineStart] + replacement + text[lineEnd:]

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return Status{}, err
	}

	return Detect(updated)
}

func Detect(contents string) (Status, error) {
	start := strings.Index(contents, beginMarker)
	if start == -1 {
		return Status{Embedded: Version}, nil
	}

	start += len(beginMarker)
	end := strings.Index(contents[start:], "\n")
	if end == -1 {
		return Status{}, fmt.Errorf("invalid managed block version: missing newline")
	}

	versionText := strings.TrimSpace(contents[start : start+end])
	if versionText == "" {
		return Status{}, fmt.Errorf("invalid managed block version: empty")
	}

	var current int
	if _, err := fmt.Sscanf(versionText, "%d", &current); err != nil {
		return Status{}, fmt.Errorf("invalid managed block version %q: %w", versionText, err)
	}

	return Status{
		CurrentVersion: current,
		Embedded:       Version,
		Stale:          current < Version,
	}, nil
}

func InferLanguage(contents string) (string, error) {
	block, err := extractManagedBlock(contents)
	if err != nil {
		return "", err
	}

	matches := make([]string, 0, 3)
	if strings.Contains(block, `"Bash(go:*)",`) {
		matches = append(matches, "go")
	}
	if strings.Contains(block, `"Bash(python:*)",`) || strings.Contains(block, `"Bash(uv:*)",`) {
		matches = append(matches, "python")
	}
	if strings.Contains(block, `"Bash(dotnet:*)",`) {
		matches = append(matches, "csharp")
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", fmt.Errorf("could not infer project language from managed allowlist block")
	default:
		return "", fmt.Errorf("managed allowlist block contains conflicting language markers: %s", strings.Join(matches, ", "))
	}
}

func CanonicalBlock(assets fs.FS, language string, includePersonal bool) (string, error) {
	data, err := fs.ReadFile(assets, "templates/common/claude/settings.local.json.tmpl")
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("settings.local.json.tmpl").Option("missingkey=error").Parse(string(data))
	if err != nil {
		return "", err
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, struct {
		Language        string
		IncludePersonal bool
	}{
		Language:        strings.TrimSpace(language),
		IncludePersonal: includePersonal,
	}); err != nil {
		return "", err
	}

	return extractManagedBlock(rendered.String())
}

func extractManagedBlock(contents string) (string, error) {
	start := strings.Index(contents, beginMarker)
	end := strings.Index(contents, endMarker)
	if start == -1 || end == -1 || end < start {
		return "", fmt.Errorf("managed block markers not found")
	}

	lineEnd := strings.Index(contents[start:], "\n")
	if lineEnd == -1 {
		return "", fmt.Errorf("managed block start marker missing newline")
	}
	contentStart := start + lineEnd + 1

	block := strings.TrimRight(contents[contentStart:end], "\n")
	if strings.TrimSpace(block) == "" {
		return "", fmt.Errorf("managed block content is empty")
	}

	return block, nil
}
