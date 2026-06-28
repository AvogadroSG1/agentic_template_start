# Task 1: Implement TUI select menu for enumerated choices in terminalPrompter

## What to build

Replace the plain text line-reading in `terminalPrompter.Ask()` (in `cmd/mkproj/main.go`) with a TUI select menu when the `choices` parameter is non-empty. When `choices` is empty, keep the current text-input behavior for now (a follow-up task handles that).

The TUI select menu MUST provide:
- Arrow key navigation (up/down)
- Highlighted current selection
- Enter to confirm selection
- The selected value is returned as-is from `Ask()`

## Scope

- The ONLY file that changes is `cmd/mkproj/main.go` — specifically the `terminalPrompter` struct and its `Ask` method
- The `Prompter` interface in `internal/prompt/prompt.go` is UNCHANGED
- The `internal/prompt/prompt_test.go` tests MUST continue to pass without modification
- A Go TUI library must be added as a dependency (recommended: `github.com/charmbracelet/huh` or `github.com/manifoldco/promptui` — pick whichever has a cleaner API for select-from-list)

## Acceptance criteria

- [ ] `terminalPrompter.Ask()` renders a navigable select menu when `choices` is non-empty
- [ ] Arrow keys move selection, Enter confirms
- [ ] When `choices` is empty, behavior remains a text input (current ReadString behavior is acceptable)
- [ ] The `Prompter` interface signature is unchanged
- [ ] All existing tests pass: `GOCACHE=$PWD/.cache/go-build go test ./... -count=1`
- [ ] `go build ./cmd/mkproj` succeeds
- [ ] The `defaultValue` parameter is respected: if a default is provided and the user confirms without changing, the default is returned

## Constraints

- Go 1.26.2 (see go.mod)
- Module path: `mkproj`
- Existing dependency: only `gopkg.in/yaml.v3`
- The three-state precedence is unchanged: flag → prompt → error. This task only changes HOW the prompt renders, not WHEN it fires.

## File to modify

`cmd/mkproj/main.go` — the `terminalPrompter` struct (lines 198-226)

Current implementation:
```go
type terminalPrompter struct {
    reader *bufio.Reader
    out    io.Writer
}

func (p terminalPrompter) Ask(_ string, label string, choices []string, defaultValue string) (string, error) {
    promptText := label
    if len(choices) > 0 {
        promptText += " [" + strings.Join(choices, "/") + "]"
    }
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
```

## What NOT to change

- `internal/prompt/prompt.go` — the `Prompter` interface
- `internal/prompt/prompt_test.go` — existing tests
- The prompt ordering logic in `Resolve()`
- Any other package
