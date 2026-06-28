package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"

	"mkproj"
	allowlist "mkproj/internal/allowlist"
	"mkproj/internal/delegate"
	initcmd "mkproj/internal/init"
	"mkproj/internal/project"
	"mkproj/internal/prompt"
	"mkproj/internal/scaffold"
	updatepkg "mkproj/internal/update"
)

func main() {
	if err := run(os.Args[1:], mkproj.Assets()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, assets fs.FS) error {
	command, args := selectCommand(args)

	switch command {
	case "init":
		return runInit(args, assets)
	case "sync-allowlist":
		return runSyncAllowlist(args, assets)
	case "update":
		return runUpdate(args, assets)
	default:
		return fmt.Errorf("unsupported command %q", command)
	}
}

func selectCommand(args []string) (string, []string) {
	command := "init"
	if len(args) > 0 {
		switch args[0] {
		case "init", "sync-allowlist", "update":
			return args[0], args[1:]
		}
	}

	return command, args
}

func runInit(args []string, assets fs.FS) error {
	flags := flag.NewFlagSet("mkproj init", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var inputs prompt.Inputs
	flags.StringVar(&inputs.ProjectName, "project-name", "", "Project name")
	flags.StringVar(&inputs.Language, "language", "", "Language")
	flags.StringVar(&inputs.ProjectType, "project-type", "", "Project type")
	flags.StringVar(&inputs.Stack, "stack", "", "Stack key")
	flags.StringVar(&inputs.AuthorName, "author-name", gitConfig("user.name"), "Author name")
	flags.StringVar(&inputs.AuthorEmail, "author-email", gitConfig("user.email"), "Author email")
	flags.StringVar(&inputs.GitHubUser, "github-user", "", "GitHub user for module path derivation")
	flags.StringVar(&inputs.Remote, "remote", "", "Remote choice (gh|url|none)")
	flags.StringVar(&inputs.RemoteURL, "remote-url", "", "Remote URL when --remote url is selected")
	flags.StringVar(&inputs.ModulePath, "module-path", "", "Module path override")
	flags.StringVar(&inputs.BdPrefix, "bd-prefix", "", "Beads prefix override")
	if err := flags.Parse(args); err != nil {
		return err
	}

	var prompter prompt.Prompter
	inputs.IsTTY = isInteractiveSession()
	if inputs.IsTTY {
		prompter = terminalPrompter{reader: bufio.NewReader(os.Stdin), out: os.Stdout}
	}

	resolved, err := prompt.Resolve(inputs, prompter)
	if err != nil {
		return err
	}
	vars, err := project.ResolveVariables(resolved)
	if err != nil {
		return err
	}

	cwd, err := currentWorkingDir()
	if err != nil {
		return err
	}

	initializer := initcmd.Initializer{
		Writer: scaffold.Writer{Assets: assets},
		Runner: delegate.NewVerboseRunner(os.Stdout),
	}

	return initializer.Run(context.Background(), cwd, vars)
}

func runSyncAllowlist(args []string, assets fs.FS) error {
	cwd, err := currentWorkingDir()
	if err != nil {
		return err
	}

	flags := flag.NewFlagSet("mkproj sync-allowlist", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var checkOnly bool
	var includePersonal bool
	var settingsPath string
	flags.BoolVar(&checkOnly, "check", false, "Only report staleness")
	flags.BoolVar(&includePersonal, "include-personal", false, "Include personal allowlist rules")
	flags.StringVar(&settingsPath, "path", filepath.Join(cwd, ".claude", "settings.local.json"), "settings.local.json path")
	if err := flags.Parse(args); err != nil {
		return err
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return err
	}
	language, err := allowlist.InferLanguage(string(data))
	if err != nil {
		return err
	}
	block, err := allowlist.CanonicalBlock(assets, language, includePersonal)
	if err != nil {
		return err
	}

	status, err := allowlist.Sync(settingsPath, block, checkOnly)
	if err != nil {
		return err
	}
	if checkOnly {
		if status.Stale {
			fmt.Printf("allowlist is %d version(s) behind; run mkproj sync-allowlist\n", status.Embedded-status.CurrentVersion)
		}
		return nil
	}

	fmt.Printf("allowlist synced to version %d\n", status.CurrentVersion)
	return nil
}

func runUpdate(args []string, assets fs.FS) error {
	flags := flag.NewFlagSet("mkproj update", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var stack string
	flags.StringVar(&stack, "stack", "", "Stack key to refresh")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(stack) == "" {
		return fmt.Errorf("missing required flag: --stack")
	}

	return updatepkg.Run(context.Background(), assets, stack, delegate.ExecRunner{}, updatepkg.ExecGitRunner{})
}

func currentWorkingDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	return wd, nil
}

func gitConfig(key string) string {
	output, err := exec.Command("git", "config", "--global", key).Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

func isInteractiveSession() bool {
	stdinInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	stdoutInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return (stdinInfo.Mode()&os.ModeCharDevice) != 0 && (stdoutInfo.Mode()&os.ModeCharDevice) != 0
}

type terminalPrompter struct {
	reader *bufio.Reader
	out    io.Writer
}

func (p terminalPrompter) Ask(_ string, label string, choices []string, defaultValue string) (string, error) {
	if len(choices) > 0 {
		return p.askSelect(label, choices, defaultValue)
	}
	return p.askText(label, defaultValue)
}

func (p terminalPrompter) askSelect(label string, choices []string, defaultValue string) (string, error) {
	// Seed the target variable so huh positions the cursor on the default.
	selected := defaultValue
	if selected == "" {
		selected = choices[0]
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(label).
				Options(huh.NewOptions(choices...)...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return "", err
	}

	return selected, nil
}

func (p terminalPrompter) askText(label string, defaultValue string) (string, error) {
	promptText := label
	if defaultValue != "" {
		promptText += " (default: " + defaultValue + ")"
	}
	promptText += ": "

	if _, err := io.WriteString(p.out, promptText); err != nil {
		return "", err
	}
	line, err := p.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultValue, nil
	}

	return line, nil
}
