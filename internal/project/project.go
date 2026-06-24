package project

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

type RemoteKind string

const (
	RemoteGH   RemoteKind = "gh"
	RemoteURL  RemoteKind = "url"
	RemoteNone RemoteKind = "none"
)

type Input struct {
	ProjectName string
	Language    string
	ProjectType string
	Stack       string
	AuthorName  string
	AuthorEmail string
	GitHubUser  string
	Remote      RemoteKind
	RemoteURL   string
	ModulePath  string
	BdPrefix    string
}

type Variables struct {
	ProjectName     string
	Language        string
	ProjectType     string
	Stack           string
	AuthorName      string
	AuthorEmail     string
	Remote          RemoteKind
	RemoteURL       string
	BdPrefix        string
	ModulePath      string
	GoModule        string
	PythonPackage   string
	CSharpNamespace string
	IncludePersonal bool
}

var nonAlphaNumeric = regexp.MustCompile(`[^a-z0-9]+`)

func ResolveVariables(input Input) (Variables, error) {
	projectName := strings.TrimSpace(input.ProjectName)
	if projectName == "" {
		return Variables{}, fmt.Errorf("project name is required")
	}
	if strings.TrimSpace(input.Language) == "" {
		return Variables{}, fmt.Errorf("language is required")
	}
	if strings.TrimSpace(input.ProjectType) == "" {
		return Variables{}, fmt.Errorf("project type is required")
	}
	if strings.TrimSpace(input.Stack) == "" {
		return Variables{}, fmt.Errorf("stack is required")
	}
	if strings.TrimSpace(input.AuthorName) == "" || strings.TrimSpace(input.AuthorEmail) == "" {
		return Variables{}, fmt.Errorf("author identity is required")
	}

	slugWords := slugWords(projectName)
	if len(slugWords) == 0 {
		return Variables{}, fmt.Errorf("project name %q does not contain usable characters", projectName)
	}

	bdPrefix := strings.TrimSpace(input.BdPrefix)
	if bdPrefix == "" {
		bdPrefix = strings.Join(slugWords, "")
	}

	modulePath := strings.TrimSpace(input.ModulePath)
	if modulePath == "" {
		if input.Remote == RemoteGH && strings.TrimSpace(input.GitHubUser) != "" {
			modulePath = fmt.Sprintf("github.com/%s/%s", strings.TrimSpace(input.GitHubUser), slugKebab(slugWords))
		} else {
			modulePath = fmt.Sprintf("github.com/your-org/%s", slugKebab(slugWords))
		}
	}

	return Variables{
		ProjectName:     projectName,
		Language:        normalize(input.Language),
		ProjectType:     normalize(input.ProjectType),
		Stack:           strings.TrimSpace(input.Stack),
		AuthorName:      strings.TrimSpace(input.AuthorName),
		AuthorEmail:     strings.TrimSpace(input.AuthorEmail),
		Remote:          input.Remote,
		RemoteURL:       strings.TrimSpace(input.RemoteURL),
		BdPrefix:        bdPrefix,
		ModulePath:      modulePath,
		GoModule:        modulePath,
		PythonPackage:   strings.Join(slugWords, "_"),
		CSharpNamespace: pascalCase(slugWords),
		IncludePersonal: false,
	}, nil
}

func slugWords(value string) []string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = nonAlphaNumeric.ReplaceAllString(value, " ")
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return nil
	}

	return parts
}

func slugKebab(words []string) string {
	return strings.Join(words, "-")
}

func pascalCase(words []string) string {
	var b strings.Builder
	for _, word := range words {
		if word == "" {
			continue
		}
		runes := []rune(word)
		b.WriteRune(unicode.ToUpper(runes[0]))
		for _, r := range runes[1:] {
			b.WriteRune(unicode.ToLower(r))
		}
	}

	return b.String()
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
