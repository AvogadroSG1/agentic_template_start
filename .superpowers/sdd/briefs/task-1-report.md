# Task 1 Report: TUI Select Menu for Enumerated Choices

## Status: DONE

## What was implemented

`cmd/mkproj/main.go` — the only file changed for the feature (plus `go.mod`/`go.sum` for the new dependency).

### Dependency added

`github.com/charmbracelet/huh v1.0.0` — chosen over `promptui` because it has a cleaner, idiomatic Go API for select-from-list with no extra struct setup required. huh's `NewSelect[T]()` with `NewOptions(choices...)` is the entire select surface.

### Implementation

`terminalPrompter.Ask()` now dispatches on whether `choices` is non-empty:

```
Ask()
 ├── len(choices) > 0  → askSelect()  (new: huh.NewForm / huh.NewSelect[string])
 └── len(choices) == 0 → askText()    (unchanged ReadString behavior)
```

`askSelect()`:
- Seeds `selected` with `defaultValue` (or `choices[0]` if defaultValue is empty) so huh positions the cursor on the right item at first render.
- Builds a single-group `huh.NewForm` with one `huh.NewSelect[string]` field pointed at `&selected`.
- Calls `form.Run()` which blocks until the user presses Enter (or Esc/Ctrl-C, which returns `huh.ErrUserAborted`).
- Returns `selected`, which huh has written the confirmed choice into.

`askText()`:
- The original ReadString body, extracted verbatim. No behavior changes.

The `terminalPrompter` struct fields (`reader *bufio.Reader`, `out io.Writer`) are unchanged.

## Acceptance criteria review

| Criterion | Status |
|---|---|
| Arrow key navigation, Enter to confirm | huh provides this natively |
| Highlighted current selection | huh provides this natively |
| Default value positions cursor | Seeded via `selected = defaultValue` before `form.Run()` |
| Empty choices keeps text-input path | `askText()` is the unchanged original |
| `Prompter` interface unchanged | `internal/prompt/prompt.go` not touched |
| All existing tests pass | 139/139 pass |
| `go build ./cmd/mkproj` succeeds | Confirmed |

## Concerns

None. The implementation is minimal and closely matches the brief. One note: the three checkpoint commits (`checkpoint: edit cmd/mkproj/main.go [claude-auto]`) from auto-save during editing appear in history before the proper commit. They are harmless but noisy — a squash before merge would clean the history if desired.

## Commits

- `dd8029d` feat(beads): add issue for TUI select menu implementation
- `b31a3d9` chore(deps): add github.com/charmbracelet/huh for TUI select menu
- `c22f805` checkpoint: edit cmd/mkproj/main.go [claude-auto]
- `3fc30cf` checkpoint: edit cmd/mkproj/main.go [claude-auto]
- `794f857` checkpoint: edit cmd/mkproj/main.go [claude-auto]
- `a3505d9` feat(prompt): implement TUI select menu for enumerated choices

## Test summary

139 tests passed across 12 packages. No tests modified. `go build ./cmd/mkproj` succeeds.
