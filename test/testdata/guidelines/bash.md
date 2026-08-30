# Bash / Shell Script Guidelines

These guidelines apply to all Bash and POSIX shell script development. They extend the general principles in `~/.claude/CLAUDE.md`. Based on the Google Shell Style Guide and ShellCheck recommendations.

## Style Principles (Ordered by Priority)

1. **Safety** - Scripts MUST fail loudly on errors, never silently proceed
2. **Portability** - Prefer POSIX-compatible constructs unless Bash features are required
3. **Clarity** - Code's purpose and rationale is clear to the reader
4. **Simplicity** - Code accomplishes its goal in the simplest way possible

## When to Use Shell Scripts

- We SHOULD use shell scripts for: task automation, CI glue, file manipulation, tool orchestration
- We MUST NOT use shell scripts for: complex data processing, anything over ~200 lines, applications with state
- We SHOULD prefer a real language (Go, Python) when logic complexity exceeds simple conditionals and loops

## Formatting

- We MUST use 2-space indentation (no tabs)
- We MUST use `shfmt` for automated formatting
- We SHOULD run `shfmt -d -i 2 -ci` in CI to enforce consistent style
- We MUST NOT exceed 100 characters per line where avoidable

## Safety Defaults

### Strict Mode
- We MUST start every script with a strict mode preamble:

```bash
#!/usr/bin/env bash
set -euo pipefail
```

- `set -e` — exit on any command failure
- `set -u` — treat unset variables as errors
- `set -o pipefail` — propagate failures through pipes

### Variable Handling
- We MUST quote all variable expansions: `"$var"`, `"${array[@]}"`
- We MUST NOT use unquoted `$var` in command arguments or assignments
- We MUST use `${var:-default}` for optional variables with fallbacks
- We MUST use `readonly` or `declare -r` for constants

```bash
# Good:
readonly config_path="${1:?usage: script.sh <config-path>}"
local output_dir="${OUTPUT_DIR:-./out}"

# Bad:
config_path=$1
output_dir=$OUTPUT_DIR
```

## Linting

- We MUST use `shellcheck` for all shell scripts
- We MUST run `shellcheck --severity=warning` in CI
- We MUST NOT use `# shellcheck disable=...` without a comment explaining why
- We SHOULD target `shellcheck` source directive (`# shellcheck shell=bash`)

## Project Structure

### Script Organization
- We SHOULD place scripts in a `scripts/` or `bin/` directory
- We MUST make scripts executable (`chmod +x`)
- We MUST include a shebang line (`#!/usr/bin/env bash`)
- We SHOULD provide a usage function for scripts with arguments

```
project/
  scripts/
    build.sh
    test.sh
    deploy.sh
  bin/
    my-tool         # Installed scripts (no .sh extension)
  tests/
    test_build.bats
    test_deploy.bats
```

### Naming Conventions
- Script files: `kebab-case.sh` (or no extension for installed tools)
- Functions: `snake_case`
- Local variables: `snake_case`
- Constants/environment: `UPPER_SNAKE_CASE`

## Functions

- We MUST use `local` for all function-scoped variables
- We SHOULD declare functions with the `name() {` syntax (not `function name {`)
- We MUST return exit codes (0 success, non-zero failure) not stdout for boolean logic
- We SHOULD keep functions under 30 lines

```bash
# Good:
validate_input() {
  local path="$1"
  [[ -f "$path" ]] || return 1
}

# Bad:
function validate_input {
  path=$1  # leaks to global scope
  if [ -f $path ]; then echo "true"; fi
}
```

## Testing

### Framework
- We SHOULD use `bats` (Bash Automated Testing System) for script testing
- We SHOULD use `bats-support` and `bats-assert` helper libraries
- We MUST test both success and failure paths
- We SHOULD test edge cases (empty input, missing files, permission errors)

### Test Structure
```bash
#!/usr/bin/env bats

setup() {
  export TEST_DIR
  TEST_DIR="$(mktemp -d)"
}

teardown() {
  rm -rf "$TEST_DIR"
}

@test "build script creates output directory" {
  run ./scripts/build.sh "$TEST_DIR/out"
  assert_success
  assert [ -d "$TEST_DIR/out" ]
}
```

## Security

- We MUST NOT use `eval` under any circumstances
- We MUST NOT use unquoted variable expansion in command positions
- We MUST NOT store secrets in scripts; use environment variables or secret managers
- We MUST use `mktemp` for temporary files (never hardcoded `/tmp/foo`)
- We SHOULD validate and sanitize all external input before use
- We MUST NOT source untrusted files

```bash
# Good:
tmp_file="$(mktemp)"
trap 'rm -f "$tmp_file"' EXIT

# Bad:
tmp_file=/tmp/my-script-output
```

## Error Handling

- We MUST use `trap` for cleanup on exit (especially for temporary files)
- We SHOULD provide meaningful error messages to stderr
- We MUST exit with non-zero codes on failure
- We SHOULD use a `die()` helper for fatal errors

```bash
die() {
  printf '%s\n' "$*" >&2
  exit 1
}

trap 'rm -f "$tmp_file"' EXIT
```

## Command Execution

- We MUST check return codes of critical commands
- We SHOULD use `||` chains for error handling instead of `set +e`
- We MUST NOT suppress stderr globally; redirect per-command when justified
- We SHOULD use `command -v` to check for required tools at script start

```bash
# Good:
command -v jq >/dev/null 2>&1 || die "jq is required but not installed"

# Bad:
which jq  # not portable, not reliable
```

## Portability

- We SHOULD prefer `printf` over `echo` for consistent behavior
- We MUST use `$()` for command substitution (never backticks)
- We SHOULD use `[[` over `[` when targeting Bash specifically
- We MUST NOT rely on GNU-specific flags without checking the platform
