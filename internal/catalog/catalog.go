// Package catalog defines the executable v1 stack boundary for mkproj.
package catalog

import "strings"

var v1Stacks = []Stack{
	{Key: "go-cli-cobra", Language: "go", ProjectType: "cli"},
	{Key: "go-api-chi", Language: "go", ProjectType: "api"},
	{Key: "python-cli-typer", Language: "python", ProjectType: "cli"},
	{Key: "python-fastapi", Language: "python", ProjectType: "api"},
	{Key: "csharp-cli", Language: "csharp", ProjectType: "cli"},
	{Key: "csharp-webapi", Language: "csharp", ProjectType: "api"},
}

// Stack identifies one shippable mkproj stack in the v1 catalog.
type Stack struct {
	Key         string
	Language    string
	ProjectType string
}

// V1Stacks returns the shipped v1 catalog in stable picker order.
func V1Stacks() []Stack {
	stacks := make([]Stack, len(v1Stacks))
	copy(stacks, v1Stacks)

	return stacks
}

// SelectableStacks returns the v1 stacks that match the requested language and project type.
func SelectableStacks(language, projectType string) []Stack {
	language = normalize(language)
	projectType = normalize(projectType)

	var stacks []Stack
	for _, stack := range v1Stacks {
		if language != "" && stack.Language != language {
			continue
		}
		if projectType != "" && stack.ProjectType != projectType {
			continue
		}

		stacks = append(stacks, stack)
	}

	return stacks
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
