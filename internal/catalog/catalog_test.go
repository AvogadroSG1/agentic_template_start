package catalog

import (
	"reflect"
	"testing"
)

func TestV1StacksReturnsOnlyTheShippedStacks(t *testing.T) {
	want := []Stack{
		{Key: "go-cli-cobra", Language: "go", ProjectType: "cli"},
		{Key: "go-api-chi", Language: "go", ProjectType: "api"},
		{Key: "python-cli-typer", Language: "python", ProjectType: "cli"},
		{Key: "python-fastapi", Language: "python", ProjectType: "api"},
		{Key: "csharp-cli", Language: "csharp", ProjectType: "cli"},
		{Key: "csharp-webapi", Language: "csharp", ProjectType: "api"},
	}

	got := V1Stacks()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("V1Stacks() = %#v, want %#v", got, want)
	}
}

func TestSelectableStacksFiltersTheV1Boundary(t *testing.T) {
	tests := []struct {
		name        string
		language    string
		projectType string
		wantKeys    []string
	}{
		{
			name:        "go cli",
			language:    "Go",
			projectType: "CLI",
			wantKeys:    []string{"go-cli-cobra"},
		},
		{
			name:        "python api",
			language:    "python",
			projectType: "api",
			wantKeys:    []string{"python-fastapi"},
		},
		{
			name:        "csharp all project types",
			language:    "csharp",
			projectType: "",
			wantKeys:    []string{"csharp-cli", "csharp-webapi"},
		},
		{
			name:        "unsupported typescript stack",
			language:    "TypeScript",
			projectType: "CLI",
			wantKeys:    nil,
		},
		{
			name:        "unsupported rust stack",
			language:    "rust",
			projectType: "api",
			wantKeys:    nil,
		},
		{
			name:        "unsupported bash stack",
			language:    "bash",
			projectType: "util",
			wantKeys:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectableStacks(tt.language, tt.projectType)
			gotKeys := stackKeys(got)

			if !reflect.DeepEqual(gotKeys, tt.wantKeys) {
				t.Fatalf("SelectableStacks(%q, %q) = %#v, want %#v", tt.language, tt.projectType, gotKeys, tt.wantKeys)
			}
		})
	}
}

func stackKeys(stacks []Stack) []string {
	if len(stacks) == 0 {
		return nil
	}

	keys := make([]string, 0, len(stacks))
	for _, stack := range stacks {
		keys = append(keys, stack.Key)
	}

	return keys
}
