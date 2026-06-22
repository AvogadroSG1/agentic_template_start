package prompt

import (
	"testing"

	"mkproj/internal/project"
)

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
