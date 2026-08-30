# TypeScript Guidelines

These guidelines apply to all TypeScript development. They extend the general principles in `~/.claude/CLAUDE.md`. Based on the TypeScript documentation and community best practices.

## Style Principles (Ordered by Priority)

1. **Type Safety** - Leverage the type system to prevent runtime errors
2. **Clarity** - Code's purpose and rationale is clear to the reader
3. **Simplicity** - Code accomplishes its goal in the simplest way possible
4. **Consistency** - Code is consistent with the broader codebase

## Formatting

- We MUST use `prettier` for all TypeScript and JavaScript source files
- We MUST NOT manually format code that prettier handles
- We SHOULD use the project's `.prettierrc` without overriding per-file

## Type System

### Type Checking

- Projects MUST expose a `typecheck` script in `package.json` that fails on type errors
- Vite and Angular projects SHOULD use `tsc --noEmit`; SvelteKit projects SHOULD use `svelte-check`

### Strict Mode
- We MUST enable `strict: true` in `tsconfig.json`
- We MUST NOT use `@ts-ignore` without a comment explaining why
- We SHOULD prefer `unknown` over `any` when the type is truly unknown
- We MUST NOT use `any` as a convenience; use proper generics or union types

### Type Declarations
- We SHOULD prefer `interface` for object shapes that may be extended
- We SHOULD prefer `type` for unions, intersections, and computed types
- We MUST NOT use `enum`; prefer `as const` objects or union literal types
- We SHOULD use discriminated unions for state modeling

```typescript
// Good:
type RequestState =
  | { status: "idle" }
  | { status: "loading" }
  | { status: "success"; data: Response }
  | { status: "error"; error: Error };

// Bad:
enum Status { Idle, Loading, Success, Error }
```

### Generics
- We SHOULD use generics to avoid code duplication
- We MUST name generic parameters descriptively when their purpose is non-obvious
- We SHOULD constrain generics with `extends` when applicable

## Linting

- We MUST use `eslint` with `@typescript-eslint/parser`
- We SHOULD use `@typescript-eslint/strict` config as a baseline
- We MUST NOT disable linting rules project-wide without documented justification
- We SHOULD treat lint warnings as errors in CI

## Project Structure

### Module Organization
- We SHOULD use `src/` layout for source code
- We MUST NOT create circular dependencies between modules
- We MUST NOT create catch-all modules (`utils`, `helpers`, `common`)
- We SHOULD co-locate tests with source files or use a parallel `tests/` directory

```
project/
  src/
    domain/       # Core business logic, no I/O
    application/  # Use cases, orchestration
    infrastructure/ # External concerns (DB, HTTP, filesystem)
    cli/          # Command-line interface (if applicable)
  tests/
    unit/
    integration/
  package.json
  tsconfig.json
```

### Naming Conventions
- Files/directories: `kebab-case`
- Classes/interfaces/types: `PascalCase`
- Functions/variables: `camelCase`
- Constants: `UPPER_SNAKE_CASE` for environment-derived values, `camelCase` for code constants

### Imports
- We MUST use ES module syntax (`import`/`export`)
- We MUST NOT use `require()` in TypeScript files
- We SHOULD use path aliases (`@/`) for deep imports

## Testing

### Framework and Style
- We SHOULD use `vitest` as the testing framework for new projects
- We SHOULD use the built-in assertion API (not chai/expect libraries)
- We SHOULD follow Arrange-Act-Assert (AAA) pattern
- We MUST NOT test implementation details; test behavior

### Coverage
- We SHOULD maintain meaningful coverage (not coverage theater)
- We MUST use `vitest --coverage` or equivalent for coverage reporting
- We SHOULD use `@vitest/coverage-v8` for coverage instrumentation

### Mocking
- We SHOULD prefer dependency injection over module mocking
- We SHOULD use `vi.fn()` for function mocks when injection is impractical
- We MUST NOT mock what you don't own — wrap external dependencies first

## Dependency Management

- We MUST use a lockfile (`package-lock.json` or `pnpm-lock.yaml`)
- We SHOULD prefer `pnpm` for workspace/monorepo projects
- We MUST NOT use floating version ranges (`*`, `latest`) in production dependencies
- We SHOULD pin major versions with `^` (caret) ranges

## Security

- We MUST run `npm audit` (or `pnpm audit`) as part of CI
- We MUST NOT store secrets in source code or environment files committed to git
- We SHOULD validate all external input at system boundaries (Zod, io-ts, or equivalent)
- We MUST use parameterized queries for database access (never string concatenation)

## Async Patterns

- We MUST use `async`/`await` over raw Promises for readability
- We MUST NOT use callbacks for async operations (use promisified versions)
- We SHOULD handle errors with try/catch at appropriate boundaries
- We SHOULD use `AbortController` for cancellation of long-running operations

## Error Handling

- We SHOULD use typed error classes that extend `Error`
- We MUST NOT swallow errors silently (empty catch blocks)
- We SHOULD use Result types (or similar) for expected failure modes
- We MUST let unexpected errors propagate to a top-level handler
