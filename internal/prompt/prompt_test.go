package prompt

import (
	"strings"
	"testing"

	"forge/internal/project"
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

func TestResolveAsksFrontendOnlyForFullstackAPIBackends(t *testing.T) {
	t.Parallel()

	prompter := &stubPrompter{responses: map[string]string{
		"frontend": "vite-ts",
		"remote":   "none",
	}}
	resolved, err := Resolve(Inputs{
		ProjectName: "Sample App",
		Language:    "go",
		ProjectType: "fullstack",
		Stack:       "go-api-chi",
		AuthorName:  "Ada",
		AuthorEmail: "ada@example.com",
		IsTTY:       true,
	}, prompter)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Frontend != "vite-ts" {
		t.Fatalf("Frontend = %q, want vite-ts", resolved.Frontend)
	}
	if !promptedFor(prompter.calls, "frontend") {
		t.Fatalf("Resolve() prompts = %#v, want frontend prompt", prompter.calls)
	}
	for _, call := range prompter.calls {
		if call.name == "frontend" {
			want := []string{"vite-ts", "sveltekit", "angular"}
			if len(call.choices) != len(want) {
				t.Fatalf("frontend choices = %#v, want %#v", call.choices, want)
			}
		}
	}
}

func TestResolveSkipsFrontendPromptForNativeFullstackStacks(t *testing.T) {
	t.Parallel()

	prompter := &stubPrompter{responses: map[string]string{
		"remote": "none",
	}}
	resolved, err := Resolve(Inputs{
		ProjectName: "Sample App",
		Language:    "go",
		ProjectType: "fullstack",
		Stack:       "go-web-templ",
		AuthorName:  "Ada",
		AuthorEmail: "ada@example.com",
		IsTTY:       true,
	}, prompter)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Frontend != "" {
		t.Fatalf("Frontend = %q, want empty", resolved.Frontend)
	}
	if promptedFor(prompter.calls, "frontend") {
		t.Fatalf("Resolve() prompts = %#v, should not ask for frontend", prompter.calls)
	}
}

func TestResolveRequiresFrontendFlagWithoutATTY(t *testing.T) {
	t.Parallel()

	_, err := Resolve(Inputs{
		ProjectName: "Sample App",
		Language:    "go",
		ProjectType: "fullstack",
		Stack:       "go-api-chi",
		AuthorName:  "Ada",
		AuthorEmail: "ada@example.com",
		Remote:      "none",
		IsTTY:       false,
	}, nil)
	if err == nil || err.Error() != "missing required flag: --frontend" {
		t.Fatalf("Resolve() error = %v, want missing frontend", err)
	}
}

func TestResolveRejectsFrontendFlagOutsideFullstackBackends(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		inputs Inputs
	}{
		{
			name: "cli project",
			inputs: Inputs{
				ProjectName: "Sample App",
				Language:    "go",
				ProjectType: "cli",
				Stack:       "go-cli-cobra",
				Frontend:    "vite-ts",
				AuthorName:  "Ada",
				AuthorEmail: "ada@example.com",
				Remote:      "none",
			},
		},
		{
			name: "native fullstack stack",
			inputs: Inputs{
				ProjectName: "Sample App",
				Language:    "python",
				ProjectType: "fullstack",
				Stack:       "python-web-jinja",
				Frontend:    "angular",
				AuthorName:  "Ada",
				AuthorEmail: "ada@example.com",
				Remote:      "none",
			},
		},
		{
			name: "frontend-only project",
			inputs: Inputs{
				ProjectName: "Sample App",
				Language:    "typescript",
				Stack:       "vite-ts",
				Frontend:    "vite-ts",
				APIBaseURL:  "https://api.example.com",
				AuthorName:  "Ada",
				AuthorEmail: "ada@example.com",
				Remote:      "none",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Resolve(tt.inputs, nil)
			if err == nil || !strings.Contains(err.Error(), "--frontend only applies") {
				t.Fatalf("Resolve() error = %v, want frontend context error", err)
			}
		})
	}
}

func TestResolveAutoSelectsTheSoleProjectTypeForTypescript(t *testing.T) {
	t.Parallel()

	resolved, err := Resolve(Inputs{
		ProjectName: "Sample App",
		Language:    "typescript",
		Stack:       "vite-ts",
		APIBaseURL:  "https://api.example.com",
		AuthorName:  "Ada",
		AuthorEmail: "ada@example.com",
		Remote:      "none",
		IsTTY:       false,
	}, nil)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.ProjectType != "frontend" {
		t.Fatalf("ProjectType = %q, want frontend", resolved.ProjectType)
	}
}

func TestResolveRequiresAPIBaseURLForFrontendWithoutATTY(t *testing.T) {
	t.Parallel()

	_, err := Resolve(Inputs{
		ProjectName: "Sample App",
		Language:    "typescript",
		Stack:       "sveltekit",
		AuthorName:  "Ada",
		AuthorEmail: "ada@example.com",
		Remote:      "none",
		IsTTY:       false,
	}, nil)
	if err == nil || err.Error() != "missing required flag: --api-base-url" {
		t.Fatalf("Resolve() error = %v, want missing api-base-url", err)
	}
}

func TestResolvePromptsAPIBaseURLWithLocalhostDefault(t *testing.T) {
	t.Parallel()

	prompter := &stubPrompter{responses: map[string]string{
		"language":     "typescript",
		"stack":        "angular",
		"project-name": "Sample App",
		"remote":       "none",
	}}
	resolved, err := Resolve(Inputs{
		AuthorName:  "Ada",
		AuthorEmail: "ada@example.com",
		IsTTY:       true,
	}, prompter)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.APIBaseURL != "http://localhost:8080" {
		t.Fatalf("APIBaseURL = %q, want the localhost default", resolved.APIBaseURL)
	}
	if promptedFor(prompter.calls, "project-type") {
		t.Fatalf("Resolve() prompts = %#v, should auto-select the sole project type", prompter.calls)
	}
}

func TestResolveRejectsAPIBaseURLForBackendOnlyProjects(t *testing.T) {
	t.Parallel()

	_, err := Resolve(Inputs{
		ProjectName: "Sample App",
		Language:    "go",
		ProjectType: "api",
		Stack:       "go-api-chi",
		APIBaseURL:  "https://api.example.com",
		AuthorName:  "Ada",
		AuthorEmail: "ada@example.com",
		Remote:      "none",
		IsTTY:       false,
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "--api-base-url only applies") {
		t.Fatalf("Resolve() error = %v, want api-base-url context error", err)
	}
}

func TestResolveAcceptsAPIBaseURLOverrideForFullstack(t *testing.T) {
	t.Parallel()

	prompter := &stubPrompter{responses: map[string]string{}}
	resolved, err := Resolve(Inputs{
		ProjectName: "Sample App",
		Language:    "go",
		ProjectType: "fullstack",
		Stack:       "go-api-chi",
		Frontend:    "sveltekit",
		APIBaseURL:  "https://api.example.com",
		AuthorName:  "Ada",
		AuthorEmail: "ada@example.com",
		Remote:      "none",
		IsTTY:       true,
	}, prompter)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.APIBaseURL != "https://api.example.com" {
		t.Fatalf("APIBaseURL = %q, want the override", resolved.APIBaseURL)
	}
	if promptedFor(prompter.calls, "api-base-url") {
		t.Fatalf("Resolve() prompts = %#v, should not ask for api-base-url on fullstack", prompter.calls)
	}
}
