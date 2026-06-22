# mkproj Update Steps Interpreter Design

## Goal

Implement the `agentic_template_start-369` slice so `mkproj update --stack <key>` can execute
the ordered `checkout` and `run` recipe steps from `sources.yaml` inside a temporary workspace,
with `strip:` handled as part of `checkout`.

## Scope

This design MUST cover only the steps interpreter portion of the maintainer update path.
It MUST NOT implement ADR-0006 normalization, golden snapshot writes, overlay seam checks, or
`sources.yaml` re-pinning. Those responsibilities belong to later slices.

## Constraints

- The implementation MUST live only on the maintainer `update` path and MUST NOT affect `init`.
- The interpreter MUST use the embedded `sources.yaml` data shipped with the binary.
- The CLI surface MUST support `mkproj update --stack <key>` and SHOULD fail clearly when the
  stack key is unknown or the needed native tool is unavailable.
- Step execution MUST be ordered exactly as declared in `sources.yaml`.
- `checkout` MUST clone a pinned ref into the temp workspace, then apply any `strip:` removals.
- `run` MUST execute in the current temp workspace directory so multi-step recipes compose.
- Errors MUST name the stack and step that failed.

## Design

Add a focused `internal/update` package with three responsibilities:

1. Load and decode `sources.yaml` into typed recipe rows.
2. Select one stack by key and validate its step schema.
3. Execute the row against a temp workspace through small, testable helper functions.

The package interface SHOULD be a single `Run` entrypoint that accepts:

- `context.Context`
- embedded assets `fs.FS`
- target stack key
- a `delegate.Runner`-compatible command runner
- a small `GitRunner` interface for `clone` and `checkout`

This keeps command execution mockable without introducing update-specific knowledge into `main`.

## Testing Strategy

The implementation MUST follow TDD with three initial behaviors:

1. A one-step scaffolder recipe runs its `run` command in order.
2. A checkout recipe clones, checks out the pinned ref, strips requested paths, then runs the
   remaining steps in order.
3. A missing native tool returns a non-zero error that names the failing step.

CLI tests SHOULD verify:

- `run()` dispatches `update`
- `runUpdate()` rejects missing `--stack`
- `runUpdate()` surfaces unknown-stack errors from `internal/update`

## Files

- Create: `internal/update/update.go`
- Create: `internal/update/update_test.go`
- Modify: `cmd/mkproj/main.go`
- Modify: `cmd/mkproj/main_test.go`
