package prompt

import (
	"testing"

	"mkproj/internal/project"
)

type promptCall struct {
	name         string
	label        string
	choices      []string
	defaultValue string
}

type stubPrompter struct {
	responses map[string]string
	calls     []promptCall
}

func (p *stubPrompter) Ask(name string, label string, choices []string, defaultValue string) (string, error) {
	p.calls = append(p.calls, promptCall{
		name:         name,
		label:        label,
		choices:      append([]string(nil), choices...),
		defaultValue: defaultValue,
	})
	if response, ok := p.responses[name]; ok {
		return response, nil
	}
	return defaultValue, nil
}

func TestResolveRequiresMissingFlagsWithoutATTY(t *testing.T) {
	t.Parallel()

	_, err := Resolve(Inputs{IsTTY: false}, nil)
	if err == nil || err.Error() != "missing required flag: --project-name" {
		t.Fatalf("Resolve() error = %v, want missing project-name", err)
	}
}

func TestResolveRejectsUnsupportedStacks(t *testing.T) {
	t.Parallel()

	_, err := Resolve(Inputs{
		ProjectName: "Sample App",
		Language:    "typescript",
		ProjectType: "cli",
		Stack:       "ts-cli",
		AuthorName:  "Ada",
		AuthorEmail: "ada@example.com",
		Remote:      "none",
		IsTTY:       false,
	}, nil)
	if err == nil {
		t.Fatal("Resolve() error = nil, want invalid language error")
	}
}

func TestResolveBuildsTheApprovedNonInteractiveContract(t *testing.T) {
	t.Parallel()

	resolved, err := Resolve(Inputs{
		ProjectName: "Sample App",
		Language:    "go",
		ProjectType: "cli",
		Stack:       "go-cli-cobra",
		AuthorName:  "Ada",
		AuthorEmail: "ada@example.com",
		Remote:      "none",
		IsTTY:       false,
	}, nil)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if resolved.Remote != project.RemoteNone {
		t.Fatalf("Remote = %q, want %q", resolved.Remote, project.RemoteNone)
	}
}

func TestResolvePromptsForGitHubUserWhenRemoteIsGH(t *testing.T) {
	t.Parallel()

	prompter := &stubPrompter{responses: map[string]string{
		"remote":      "gh",
		"github-user": "octocat",
	}}
	resolved, err := Resolve(Inputs{
		ProjectName: "Sample App",
		Language:    "go",
		ProjectType: "cli",
		Stack:       "go-cli-cobra",
		AuthorName:  "Ada",
		AuthorEmail: "ada@example.com",
		IsTTY:       true,
	}, prompter)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.GitHubUser != "octocat" {
		t.Fatalf("GitHubUser = %q, want octocat", resolved.GitHubUser)
	}
	if !promptedFor(prompter.calls, "github-user") {
		t.Fatalf("Resolve() prompts = %#v, want github-user prompt", prompter.calls)
	}
}

func TestResolveSkipsGitHubUserPromptWhenRemoteIsNotGH(t *testing.T) {
	t.Parallel()

	prompter := &stubPrompter{responses: map[string]string{
		"remote": "none",
	}}
	resolved, err := Resolve(Inputs{
		ProjectName: "Sample App",
		Language:    "go",
		ProjectType: "cli",
		Stack:       "go-cli-cobra",
		AuthorName:  "Ada",
		AuthorEmail: "ada@example.com",
		IsTTY:       true,
	}, prompter)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.GitHubUser != "" {
		t.Fatalf("GitHubUser = %q, want empty", resolved.GitHubUser)
	}
	if promptedFor(prompter.calls, "github-user") {
		t.Fatalf("Resolve() prompts = %#v, should not ask for github-user", prompter.calls)
	}
}

func promptedFor(calls []promptCall, name string) bool {
	for _, call := range calls {
		if call.name == name {
			return true
		}
	}

	return false
}
