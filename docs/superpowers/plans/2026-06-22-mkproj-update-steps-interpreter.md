# mkproj Update Steps Interpreter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the `agentic_template_start-369` maintainer-path step interpreter so `mkproj update --stack <key>` executes `sources.yaml` recipes in a temp workspace.

**Architecture:** Add a narrow `internal/update` package that decodes embedded `sources.yaml`, selects a stack recipe, and executes ordered `checkout` / `run` steps through injected runner interfaces. Wire `cmd/mkproj/main.go` to call that package for `update`, but stop before normalization, snapshot writes, or `sources.yaml` mutation.

**Tech Stack:** Go 1.24, stdlib `os/exec`, embedded assets via `embed.FS`, existing `internal/delegate` runner abstractions, `go test`.

## Global Constraints

- The interpreter MUST run only on the maintainer `update` path and MUST NOT affect `init`.
- The implementation MUST use embedded `sources.yaml`, not filesystem-relative reads.
- The slice MUST support `mkproj update --stack <key>` only; full-repo refresh is out of scope.
- The slice MUST execute ordered `checkout` / `run` steps plus `strip:` handling on checkout.
- The slice MUST fail loudly naming the stack and failing step.
- The slice MUST NOT implement normalization, golden snapshot writes, overlay seam checks, or `sources.yaml` re-pinning.

---

### Task 1: Build and test the recipe interpreter package

**Files:**
- Create: `internal/update/update.go`
- Create: `internal/update/update_test.go`

**Interfaces:**
- Consumes: `fs.FS` embedded assets, `context.Context`, `delegate.Runner`
- Produces: `func Run(ctx context.Context, assets fs.FS, stack string, runner CommandRunner, git GitRunner) error`

- [ ] **Step 1: Write the failing tests**

```go
func TestRunExecutesSingleStepRecipeInOrder(t *testing.T) { /* ... */ }
func TestRunExecutesCheckoutStripAndRunRecipe(t *testing.T) { /* ... */ }
func TestRunReturnsHelpfulErrorWhenToolIsMissing(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/update -count=1`
Expected: FAIL because `internal/update` does not exist yet.

- [ ] **Step 3: Write the minimal implementation**

```go
type CommandRunner interface {
    Run(ctx context.Context, dir string, step string, command string, args ...string) error
}

type GitRunner interface {
    Clone(ctx context.Context, repo string, dir string) error
    Checkout(ctx context.Context, dir string, ref string) error
}

func Run(ctx context.Context, assets fs.FS, stack string, runner CommandRunner, git GitRunner) error
```

Implement typed `sources.yaml` decoding, recipe selection, ordered step execution, temp workspace
setup, checkout-strip behavior, and contextual errors.

- [ ] **Step 4: Run test to verify it passes**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/update -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/update/update.go internal/update/update_test.go
git commit -m "feat: add mkproj update step interpreter" -m "Co-Authored-By: Peter O'Connor <poconnor@stackoverflow.com>\nCo-Authored-By: Codex <noreply@anthropic.com> - GPT-5"
```

### Task 2: Wire the CLI entrypoint to the interpreter

**Files:**
- Modify: `cmd/mkproj/main.go`
- Modify: `cmd/mkproj/main_test.go`
- Test: `cmd/mkproj/main_test.go`

**Interfaces:**
- Consumes: `update.Run(ctx, assets, stack, runner, git)`
- Produces: `runUpdate(args []string, assets fs.FS) error`

- [ ] **Step 1: Write the failing tests**

```go
func TestSelectCommandRecognizesUpdate(t *testing.T) { /* ... */ }
func TestRunUpdateRequiresStackFlag(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=$PWD/.cache/go-build go test ./cmd/mkproj -count=1`
Expected: FAIL because `update` still returns `not implemented yet`.

- [ ] **Step 3: Write the minimal implementation**

```go
func runUpdate(args []string, assets fs.FS) error
```

Parse `--stack`, reject empty values, and call `update.Run(...)` with real runner implementations.

- [ ] **Step 4: Run test to verify it passes**

Run: `GOCACHE=$PWD/.cache/go-build go test ./cmd/mkproj -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/mkproj/main.go cmd/mkproj/main_test.go
git commit -m "feat: wire mkproj update command" -m "Co-Authored-By: Peter O'Connor <poconnor@stackoverflow.com>\nCo-Authored-By: Codex <noreply@anthropic.com> - GPT-5"
```

### Task 3: Run full verification for the slice

**Files:**
- Verify only

**Interfaces:**
- Consumes: Task 1 and Task 2 outputs
- Produces: verified maintainer-path `update` slice ready for review and landing

- [ ] **Step 1: Run focused tests**

Run: `GOCACHE=$PWD/.cache/go-build go test ./internal/update ./cmd/mkproj -count=1`
Expected: PASS

- [ ] **Step 2: Run full Go suite**

Run: `GOCACHE=$PWD/.cache/go-build go test ./... -count=1`
Expected: PASS

- [ ] **Step 3: Request code review**

Use `superpowers:requesting-code-review` against the update slice before landing.

- [ ] **Step 4: Land only the update commit(s) onto main**

Use a selective cherry-pick or equivalent clean landing path so unrelated in-progress work in
`codex-mkproj-template-system` is not merged accidentally.

- [ ] **Step 5: Push**

```bash
git push
```
