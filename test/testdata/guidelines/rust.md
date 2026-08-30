# Rust Guidelines

These guidelines apply to all Rust development. They extend the general principles in `~/.claude/CLAUDE.md`. Based on the Rust API Guidelines and community best practices.

## Style Principles (Ordered by Priority)

1. **Safety** - Leverage the ownership system; avoid unsafe unless proven necessary
2. **Clarity** - Code's purpose and rationale is clear to the reader
3. **Performance** - Zero-cost abstractions over runtime indirection
4. **Simplicity** - Code accomplishes its goal in the simplest way possible

## Formatting

- We MUST use `rustfmt` for all Rust source files
- We MUST NOT override `rustfmt` defaults without documented justification in `rustfmt.toml`
- We MUST run `cargo fmt --check` in CI

## Linting

- We MUST use `clippy` with `#![warn(clippy::all)]` at the crate root
- We SHOULD enable `clippy::pedantic` for library crates
- We MUST NOT use `#[allow(clippy::...)]` without a comment explaining why
- We MUST run `cargo clippy -- -D warnings` in CI (deny all warnings)

## Type System and Safety

### Ownership
- We MUST prefer borrowing (`&T`, `&mut T`) over cloning unless ownership transfer is needed
- We SHOULD prefer `impl Trait` over `dyn Trait` when the concrete type is known at compile time
- We MUST NOT use `unsafe` without a `// SAFETY:` comment explaining the invariant

### Error Handling
- We MUST use `Result<T, E>` for operations that can fail
- We MUST NOT use `unwrap()` or `expect()` in library code (use `?` propagation)
- We SHOULD define domain-specific error types using `thiserror`
- We SHOULD use `anyhow` only in binary crates (not libraries)
- We MUST NOT use `panic!` for expected error conditions

```rust
// Good:
use thiserror::Error;

#[derive(Error, Debug)]
pub enum ConfigError {
    #[error("missing key: {0}")]
    MissingKey(String),
    #[error("parse failed: {0}")]
    Parse(#[from] toml::de::Error),
}

// Bad:
fn load_config(path: &str) -> Config {
    let content = std::fs::read_to_string(path).unwrap(); // panics
    toml::from_str(&content).unwrap() // panics
}
```

### Newtypes
- We SHOULD use newtypes to distinguish domain concepts that share a primitive representation
- We SHOULD derive common traits (`Debug`, `Clone`, `PartialEq`) on domain types

## Project Structure

### Crate Organization
- We SHOULD separate library (`lib.rs`) from binary (`main.rs`) concerns
- We MUST NOT create circular dependencies between crates
- We SHOULD use workspace members for multi-crate projects
- We MUST NOT create catch-all modules (`utils`, `helpers`, `common`)

```
project/
  src/
    lib.rs        # Public API surface
    main.rs       # Binary entry point (thin: parse args, call lib)
    domain/       # Core types and business logic
    application/  # Use cases, orchestration
    infrastructure/ # External concerns (DB, HTTP, filesystem)
  tests/
    integration/  # Integration tests (separate compilation unit)
  Cargo.toml
```

### Naming Conventions
- Crates/modules: `snake_case`
- Types/traits: `PascalCase`
- Functions/variables: `snake_case`
- Constants/statics: `UPPER_SNAKE_CASE`
- Lifetime parameters: short lowercase (`'a`, `'ctx`)

## Testing

### Framework and Style
- We MUST use the built-in `#[test]` framework for unit tests
- We SHOULD place unit tests in a `#[cfg(test)] mod tests` block in the same file
- We SHOULD place integration tests in the `tests/` directory
- We SHOULD use `assert_eq!` and `assert_ne!` over bare `assert!` for better diagnostics

### Coverage
- We SHOULD use `cargo-tarpaulin` or `cargo-llvm-cov` for coverage reporting
- We MUST NOT chase coverage numbers at the expense of meaningful tests

### Property Testing
- We SHOULD use `proptest` or `quickcheck` for invariant-heavy code
- We SHOULD use `insta` for snapshot testing of complex output

## Dependency Management

- We MUST specify exact minimum versions in `Cargo.toml` (use `cargo-minimal-versions` to verify)
- We SHOULD prefer well-maintained crates with active security advisories
- We MUST NOT use `*` version requirements
- We SHOULD run `cargo update` periodically to stay current within SemVer bounds

## Security

- We MUST run `cargo audit` as part of CI to detect known vulnerabilities
- We MUST NOT use `unsafe` without exhaustive justification and review
- We SHOULD use `secrecy` crate for sensitive values (prevents accidental logging)
- We MUST validate all external input at system boundaries

## Performance

- We SHOULD prefer stack allocation over heap allocation where feasible
- We SHOULD use `&str` over `String` in function parameters that don't need ownership
- We MUST NOT prematurely optimize; profile first with `criterion` benchmarks
- We SHOULD use `#[inline]` only when benchmarks demonstrate benefit

## Concurrency

- We SHOULD use `tokio` as the async runtime for network-bound applications
- We MUST NOT mix blocking and async code without `spawn_blocking`
- We SHOULD prefer channels (`mpsc`, `broadcast`) over shared mutable state
- We MUST use `Arc<Mutex<T>>` or `Arc<RwLock<T>>` when shared state is unavoidable
