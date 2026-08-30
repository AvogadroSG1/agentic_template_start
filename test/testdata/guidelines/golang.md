# Go Guidelines

These guidelines apply to all Go development. They extend the general principles in `~/.claude/CLAUDE.md`. Based on [Google's Go Style Guide](https://google.github.io/styleguide/go/).

## Style Principles (Ordered by Priority)

1. **Clarity** - Code's purpose and rationale is clear to the reader
2. **Simplicity** - Code accomplishes its goal in the simplest way possible
3. **Concision** - Code has a high signal-to-noise ratio
4. **Maintainability** - Code can be easily maintained over time
5. **Consistency** - Code is consistent with the broader codebase

## Formatting

- We MUST use `gofmt` for all Go source files
- We MUST NOT set a fixed line length; if a line feels too long, prefer refactoring over splitting
- We MUST NOT split lines before indentation changes (function declarations, conditionals)
- We MUST NOT split long strings (URLs, etc.) across multiple lines

## Naming

### General Rules
- We MUST use `MixedCaps` or `mixedCaps` (camel case), never `snake_case`
- We MUST NOT use `SCREAMING_SNAKE_CASE` for constants
- We SHOULD keep names short; length proportional to scope size, inversely proportional to usage frequency
- We MUST NOT repeat context already clear from package name, type, or enclosing scope

### Package Names
- We MUST use concise, lowercase names without underscores
- We MUST NOT use uninformative names: `util`, `utility`, `common`, `helper`, `model`
- We SHOULD avoid names likely shadowed by common variables (prefer `usercount` over `count`)

### Receiver Names
- We MUST use short (one or two letter) abbreviations for the type
- We MUST apply receiver names consistently across all methods of a type
- We MUST NOT use `this` or `self`

| Type | Receiver |
|------|----------|
| `Tray` | `t` |
| `ResearchInfo` | `ri` |
| `ReportWriter` | `w` |

### Constants
- We MUST use `MixedCaps` based on role, not values
- We MUST NOT use `SCREAMING_SNAKE_CASE` or `K` prefix

```go
// Good:
const MaxPacketSize = 512

// Bad:
const MAX_PACKET_SIZE = 512
const kMaxBufferSize = 1024
```

### Initialisms
- We MUST maintain consistent case: `URL` or `url`, never `Url`
- We MUST keep each initialism in the same internal case: `XMLAPI` or `xmlAPI`, not `XmlApi`

### Getters
- We MUST NOT use `Get` prefix on getters (use `Counts` not `GetCounts`)
- We SHOULD use `Compute` or `Fetch` for complex/remote operations

### Exported Symbol Repetition
- We MUST NOT repeat the package name in exported symbols

```go
// Good:
widget.New          // not widget.NewWidget
db.Load             // not db.LoadFromDatabase
```

### Variable Name vs Type Repetition

| Repetitive | Better |
|------------|--------|
| `var numUsers int` | `var users int` |
| `var nameString string` | `var name string` |
| `var primaryProject *Project` | `var primary *Project` |

## Project Structure

### Module Organization
- We SHOULD organize code into focused packages that separate concerns
- We MUST NOT create circular dependencies between packages
- We MUST NOT create catch-all packages (`util`, `common`, `helper`)
- We SHOULD follow the least mechanism principle: core language > stdlib > well-known libraries

### Package Size
- We SHOULD place tightly coupled unexported types together
- We SHOULD split large packages across files by domain (e.g., `reader.go`, `writer.go`)
- We MAY use `doc.go` for package-level documentation

## Imports

### Import Grouping
We MUST order imports in these groups, separated by blank lines:

```go
import (
    "fmt"                    // 1. Standard library
    "hash/adler32"

    "github.com/pkg/errors"  // 2. Third-party packages

    foopb "myproj/foo/proto" // 3. Protocol Buffer imports

    _ "myproj/rpc/protocols" // 4. Side-effect imports
)
```

### Import Rules
- We SHOULD NOT rename imports unless avoiding name collision
- We MUST NOT use dot imports (`import .`)
- We SHOULD use side-effect imports (`import _`) only in `main` packages or tests

## Error Handling

### Returning Errors
- We MUST use `error` as the last return parameter
- We MUST return `nil` for successful operations
- We MUST NOT return concrete error types from exported functions; return `error` interface

```go
// Good:
func Good() error { /* ... */ }
func GoodLookup() (*Result, error) { return res, nil }

// Bad:
func Bad() *os.PathError { /* ... */ }
```

### Error Strings
- We MUST NOT capitalize error strings (unless proper noun/acronym)
- We MUST NOT end error strings with punctuation

```go
// Good:
err := fmt.Errorf("something bad happened")

// Bad:
err := fmt.Errorf("Something bad happened.")
```

### Error Flow
- We MUST handle errors before proceeding (early return pattern)
- We MUST NOT discard errors with `_` unless explicitly safe and documented
- We MUST NOT distinguish errors by string matching; use sentinel values or `errors.Is`/`errors.As`

```go
// Good:
if err != nil {
    return fmt.Errorf("operation failed: %v", err)
}
// normal code continues

// Bad:
if err != nil {
    // error handling
} else {
    // normal code in abnormal indentation
}
```

### Error Wrapping
- We SHOULD use `%v` when adding context at system boundaries or for human display
- We SHOULD use `%w` when callers need to inspect the underlying error programmatically
- We SHOULD place `%w` at the end of the error string: `"context: %w"`

```go
// %v - drops structured info (system boundaries):
return fmt.Errorf("couldn't find fortune database: %v", err)

// %w - preserves error chain (programmatic inspection):
return fmt.Errorf("couldn't find remote file: %w", err)
```

### Sentinel Errors
- We SHOULD use global sentinel values for programmatic error inspection

```go
var (
    ErrDuplicate = errors.New("duplicate")
    ErrNotFound  = errors.New("not found")
)
```

### In-Band Errors
- We MUST NOT signal errors via special return values (`-1`, `null`, empty string)
- We SHOULD use multiple return values with `ok` pattern

```go
// Good:
func Lookup(key string) (value string, ok bool)

// Bad:
func Lookup(key string) int // returns -1 on not found
```

## Interfaces

- We SHOULD define interfaces in the consuming package, not the implementing package
- We MUST return concrete types from implementing packages
- We MUST NOT define interfaces before they're needed (YAGNI)
- We MUST NOT export unused interfaces

```go
// Good - interface in consumer:
package consumer

type Thinger interface { Thing() bool }

func Foo(t Thinger) string { /* ... */ }
```

## Concurrency

### Goroutine Lifetimes
- We MUST make clear when and whether goroutines exit
- We MUST use `sync.WaitGroup` or similar to prevent goroutines from outliving their parent

```go
// Good:
func (w *Worker) Run(ctx context.Context) error {
    var wg sync.WaitGroup
    for item := range w.q {
        wg.Add(1)
        go func() {
            defer wg.Done()
            process(ctx, item)
        }()
    }
    wg.Wait()
    return nil
}
```

### Synchronous Functions
- We SHOULD prefer synchronous functions; let callers add concurrency
- We SHOULD return results directly rather than through channels/callbacks

### Channel Direction
- We SHOULD specify channel direction in function parameters

```go
func sum(values <-chan int) int { /* ... */ }
```

### Copying
- We MUST NOT copy `sync.Mutex` or types containing sync objects
- We MUST NOT copy `bytes.Buffer` types (aliasing issues)

## Context

- We MUST pass `context.Context` explicitly as the first parameter
- We MUST NOT add context to struct members
- We SHOULD use `context.Background()` only in entrypoints (`main`, `init`)

```go
// Good:
func F(ctx context.Context, other args) {}
```

## Panics and Recovery

- We MUST use `error` and multiple returns for normal error handling
- We MUST NOT use `panic` for expected failures
- We SHOULD use `log.Fatal` or `log.Exit` for unrecoverable program errors
- We SHOULD use `MustXYZ` naming convention for helpers that panic on failure
- We SHOULD call `Must` functions only during program startup or in test helpers

```go
func MustParse(version string) *Version {
    v, err := Parse(version)
    if err != nil {
        panic(fmt.Sprintf("MustParse(%q) = _, %v", version, err))
    }
    return v
}

var DefaultVersion = MustParse("1.2.3")
```

## Variable Declarations

- We SHOULD use `:=` when initializing with non-zero values
- We SHOULD use `var` for zero-value declarations conveying "empty, ready for later use"
- We SHOULD use composite literals when initial elements are known
- We SHOULD prefer `nil` slices over empty slice literals (`var t []string` not `t := []string{}`)
- We MUST use `len()` to check emptiness, not `== nil`
- We SHOULD preallocate slices/maps only when final size is known or justified by profiling

```go
// Non-zero init:
i := 42

// Zero-value declaration:
var coords Point
if err := json.Unmarshal(data, &coords); err != nil { /* ... */ }

// Nil slice preferred:
var t []string  // not: t := []string{}
```

## Struct Literals

- We MUST use field names for struct literals of types from other packages
- We SHOULD omit zero-value fields when clarity is not lost
- We MUST place closing braces on their own line for multi-line literals

```go
// Good:
r := csv.Reader{
    Comma:   ',',
    Comment: '#',
}
```

## Function Design

### Argument Lists
- We SHOULD use option structs when signatures grow beyond 4-5 parameters
- We MUST NOT include `context.Context` in option structs
- We SHOULD use variadic option pattern (`func(...Option)`) when most callers don't need options

```go
// Option struct pattern:
type ReplicationOptions struct {
    PrimaryRegions    []string
    ReadonlyRegions   []string
    OverwritePolicies bool
}

func EnableReplication(ctx context.Context, opts ReplicationOptions) { /* ... */ }

// Variadic option pattern:
type Option func(*options)

func WithTimeout(d time.Duration) Option {
    return func(o *options) { o.timeout = d }
}

func New(ctx context.Context, opts ...Option) *Client { /* ... */ }
```

### Pass Values
- We SHOULD pass values, not pointers, unless the function mutates or the struct is large
- We MUST NOT pass pointers just to save bytes for small types

## Testing

### Framework and Style
- We MUST use the standard `testing` package
- We MUST NOT use assertion libraries; they fragment developer experience
- We SHOULD use `cmp.Equal` and `cmp.Diff` from `github.com/google/go-cmp` for deep comparisons
- We SHOULD follow the `got`-before-`want` convention: `YourFunc(%v) = %v, want %v`

### Test Organization
- We SHOULD use table-driven tests for repetitive test cases
- We SHOULD use field names in test case struct literals
- We SHOULD use `t.Run` for subtests
- We SHOULD use `t.Helper()` in all test helper functions

```go
func TestStrJoin(t *testing.T) {
    tests := []struct {
        name      string
        slice     []string
        separator string
        want      string
    }{
        {
            name:      "basic join",
            slice:     []string{"a", "b", "c"},
            separator: ",",
            want:      "a,b,c",
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := StrJoin(tt.slice, tt.separator)
            if got != tt.want {
                t.Errorf("StrJoin(%v, %q) = %q, want %q", tt.slice, tt.separator, got, tt.want)
            }
        })
    }
}
```

### Failure Reporting
- We SHOULD use `t.Error` to continue after failures (show all bugs)
- We SHOULD use `t.Fatal` only when subsequent assertions are meaningless without success
- We MUST NOT call `t.Fatal` from separate goroutines
- We SHOULD use `t.Helper()` in all test helper functions for proper line attribution

### Test Helpers
- Test helpers SHOULD call `t.Fatal` for setup failures (precondition failures)
- Test helpers MUST NOT decide test correctness; return values for the test to assert on
- We SHOULD use `t.Cleanup()` for teardown

```go
func mustLoadDataset(t *testing.T) []byte {
    t.Helper()
    data, err := os.ReadFile("testdata/dataset")
    if err != nil {
        t.Fatalf("Could not load dataset: %v", err)
    }
    return data
}
```

## Audit/Security + Coverage

- We SHOULD use `golangci-lint` to run the Go lint set before review or in CI
- We SHOULD use `go test -cover` when we need to measure test coverage for behavior-bearing packages
- We SHOULD use `govulncheck` to audit Go dependencies for known security vulnerabilities before release or in CI
- We SHOULD keep coverage and vulnerability checks alongside the other test and release gates, not as ad hoc local-only steps

## Documentation

### Doc Comments
- We MUST have doc comments on all top-level exported names
- We SHOULD start doc comments with the symbol name as a full sentence
- We SHOULD document concurrency safety for types with mutating operations
- We SHOULD document explicit cleanup requirements

```go
// A Request represents a request to run a command.
type Request struct { /* ... */ }

// Encode writes the JSON encoding of req to w.
func Encode(w io.Writer, req *Request) { /* ... */ }
```

### Package Comments
- We MUST place package comments immediately above the `package` clause
- We SHOULD have a single package comment per package (in `doc.go` for multi-file packages)

```go
// Package math provides basic constants and mathematical functions.
package math
```

### Comment Style
- Comments SHOULD explain **why**, not **what**
- Doc comments MUST be complete sentences
- End-of-line comments MAY be phrases
- We SHOULD use `%q` verb for string formatting in error/log messages

## Generics

- We SHOULD NOT use generics prematurely; prefer concrete types until multiple instantiations are needed
- We MUST NOT use generics to create DSLs
- We MAY use generics where they genuinely reduce code duplication across multiple types

## Crypto

- We MUST use `crypto/rand` for key generation, never `math/rand`

```go
buf := make([]byte, 16)
if _, err := rand.Read(buf); err != nil {
    log.Fatalf("out of randomness: %v", err)
}
return fmt.Sprintf("%x", buf)
```

## Dependencies

### Package Management
- We MUST use Go modules (`go.mod`) for dependency management
- We MUST pin dependency versions via `go.sum`
- We SHOULD minimize external dependencies (least mechanism principle)

### Recommended Packages

| Purpose | Package |
|---------|---------|
| CLI (simple) | `github.com/google/subcommands` |
| CLI (complex) | `github.com/spf13/cobra` |
| Deep comparison | `github.com/google/go-cmp/cmp` |
| Structured logging | `log/slog` (stdlib, Go 1.21+) |
| HTTP | `net/http` (stdlib) |
| JSON | `encoding/json` (stdlib) |
| Testing | `testing` (stdlib) + `go-cmp` |
| Mocking | interfaces + test doubles in `*test` packages |

## Configuration

- We MUST NOT hardcode configuration values
- We SHOULD use environment variables for runtime configuration (12 Factor III)
- We SHOULD use flags only in `package main`; general-purpose packages use APIs

```go
// Flags in main only:
var pollInterval = flag.Duration("poll_interval", time.Minute, "Interval for polling.")

// General packages: accept config via API
type Config struct {
    PollInterval time.Duration
    MaxRetries   int
}

func New(cfg Config) *Client { /* ... */ }
```

## Logging

- We SHOULD use `log/slog` (Go 1.21+) for structured logging
- We SHOULD use `log.Fatal` for unrecoverable startup errors
- We SHOULD NOT use expensive function calls in log statements that may be filtered

```go
// Good - guard expensive calls:
if log.V(2) {
    log.Infof("Query details: %v", sql.Explain())
}
```

## Type Aliases vs Type Definitions

- We SHOULD use type definitions (`type T1 T2`) for creating new types
- We SHOULD use type aliases (`type T1 = T2`) only for package migrations
- We SHOULD prefer `any` over `interface{}` in Go 1.18+ code
