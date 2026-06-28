package prompt

import (
	"fmt"
	"strings"

	"mkproj/internal/catalog"
	"mkproj/internal/project"
)

type Prompter interface {
	Ask(name string, prompt string, choices []string, defaultValue string) (string, error)
}

type Inputs struct {
	ProjectName string
	Language    string
	ProjectType string
	Stack       string
	AuthorName  string
	AuthorEmail string
	GitHubUser  string
	Remote      string
	RemoteURL   string
	ModulePath  string
	BdPrefix    string
	IsTTY       bool
}

func Resolve(input Inputs, prompter Prompter) (project.Input, error) {
	projectName, err := resolveValue(input.ProjectName, input.IsTTY, prompter, "project-name", "Project name", nil, "")
	if err != nil {
		return project.Input{}, err
	}

	languageChoices := []string{"go", "python", "csharp"}
	language, err := resolveValue(input.Language, input.IsTTY, prompter, "language", "Language", languageChoices, "")
	if err != nil {
		return project.Input{}, err
	}
	language = normalize(language)
	if !contains(languageChoices, language) {
		return project.Input{}, fmt.Errorf("invalid --language %q (valid choices: %s)", language, strings.Join(languageChoices, ", "))
	}

	projectTypeChoices := []string{"cli", "api"}
	projectType, err := resolveValue(input.ProjectType, input.IsTTY, prompter, "project-type", "Project type", projectTypeChoices, "")
	if err != nil {
		return project.Input{}, err
	}
	projectType = normalize(projectType)
	if !contains(projectTypeChoices, projectType) {
		return project.Input{}, fmt.Errorf("invalid --project-type %q (valid choices: %s)", projectType, strings.Join(projectTypeChoices, ", "))
	}

	stackChoices := make([]string, 0)
	for _, stack := range catalog.SelectableStacks(language, projectType) {
		stackChoices = append(stackChoices, stack.Key)
	}
	stack, err := resolveValue(input.Stack, input.IsTTY, prompter, "stack", "Stack", stackChoices, "")
	if err != nil {
		return project.Input{}, err
	}
	stack = strings.TrimSpace(stack)
	if !contains(stackChoices, stack) {
		return project.Input{}, fmt.Errorf("invalid --stack %q (valid choices: %s)", stack, strings.Join(stackChoices, ", "))
	}

	authorName, err := resolveValue(input.AuthorName, input.IsTTY, prompter, "author-name", "Author name", nil, "")
	if err != nil {
		return project.Input{}, err
	}
	authorEmail, err := resolveValue(input.AuthorEmail, input.IsTTY, prompter, "author-email", "Author email", nil, "")
	if err != nil {
		return project.Input{}, err
	}

	remoteChoices := []string{"gh", "url", "none"}
	remoteValue, err := resolveValue(input.Remote, input.IsTTY, prompter, "remote", "Remote", remoteChoices, "gh")
	if err != nil {
		return project.Input{}, err
	}
	remoteValue = normalize(remoteValue)
	if !contains(remoteChoices, remoteValue) {
		return project.Input{}, fmt.Errorf("invalid --remote %q (valid choices: %s)", remoteValue, strings.Join(remoteChoices, ", "))
	}

	gitHubUser := strings.TrimSpace(input.GitHubUser)
	if remoteValue == string(project.RemoteGH) && gitHubUser == "" {
		gitHubUser, err = resolveValue(input.GitHubUser, input.IsTTY, prompter, "github-user", "GitHub user", nil, "")
		if err != nil {
			return project.Input{}, err
		}
		gitHubUser = strings.TrimSpace(gitHubUser)
	}

	resolved := project.Input{
		ProjectName: projectName,
		Language:    language,
		ProjectType: projectType,
		Stack:       stack,
		AuthorName:  authorName,
		AuthorEmail: authorEmail,
		GitHubUser:  gitHubUser,
		Remote:      project.RemoteKind(remoteValue),
		RemoteURL:   strings.TrimSpace(input.RemoteURL),
		ModulePath:  strings.TrimSpace(input.ModulePath),
		BdPrefix:    strings.TrimSpace(input.BdPrefix),
	}
	if resolved.Remote == project.RemoteURL && resolved.RemoteURL == "" {
		return project.Input{}, fmt.Errorf("missing required flag: --remote-url")
	}

	return resolved, nil
}

func resolveValue(current string, isTTY bool, prompter Prompter, flagName string, label string, choices []string, defaultValue string) (string, error) {
	current = strings.TrimSpace(current)
	if current != "" {
		return current, nil
	}
	if !isTTY {
		return "", fmt.Errorf("missing required flag: --%s", flagName)
	}
	if prompter == nil {
		return "", fmt.Errorf("interactive prompt requested for --%s but no prompter is available", flagName)
	}

	return prompter.Ask(flagName, label, choices, defaultValue)
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}

	return false
}
