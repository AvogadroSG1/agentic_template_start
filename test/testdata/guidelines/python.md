# Python Guidelines

These guidelines apply to all Python development. They extend the general principles in `~/.claude/CLAUDE.md`.

## Virtual Environments

- We MUST use virtual environments for all Python projects
- We MUST run all Python programs within their virtual environments
- We SHOULD use `venv` or `uv venv` for environment creation
- We MUST NOT install packages globally with pip

```bash
# Creating a virtual environment
python3 -m venv .venv
source .venv/bin/activate

# Or with uv
uv venv
source .venv/bin/activate
```

## Language Features

### Type Hints
- We SHOULD use type hints for all function signatures
- We SHOULD use `from __future__ import annotations` for forward references
- We SHOULD use `TypedDict` or Pydantic models for structured data
- We SHOULD run `pyright` for type checking
- We SHOULD treat `pyright` as the shipped overlay type-checker for Python

### Data Classes
- We SHOULD use `dataclasses` for simple data containers
- We SHOULD use Pydantic models when validation is needed
- We SHOULD prefer immutable data (`frozen=True` for dataclasses)

```python
from dataclasses import dataclass
from typing import Literal

@dataclass(frozen=True)
class Signal:
    category: str
    severity: Literal["Low", "Medium", "High"]
    verbatim_quote: str
    speaker: str
    speaker_role: str
    executive_analysis: str
    source_link: str
    related_context: str | None = None
```

## Project Structure

### Package Organization
- We SHOULD use `src/` layout for packages
- We SHOULD separate concerns into subpackages
- We MUST NOT create circular imports

```
project/
  src/
    project_name/
      domain/       # Core business logic
      application/  # Use cases, orchestration
      infrastructure/  # External concerns
      cli/          # Command-line interface
  tests/
    unit/
    integration/
  pyproject.toml
```

### Naming Conventions
- Packages/modules: `snake_case`
- Classes: `PascalCase`
- Functions/variables: `snake_case`
- Constants: `UPPER_SNAKE_CASE`

## Testing

### Framework and Style
- We SHOULD use `pytest` as the testing framework
- We SHOULD use pytest fixtures for setup/teardown
- We SHOULD use `pytest-cov` for coverage
- We SHOULD use `pytest-mock` when tests need mocking helpers
- We SHOULD follow Arrange-Act-Assert (AAA) pattern

### Test Organization
```
tests/
  unit/
  test_scout_agent.py
  integration/
    test_slack_fetcher.py
  conftest.py  # Shared fixtures
```

### Test Naming
```python
def test_parse_valid_json_returns_evidence_log():
    ...

def test_scout_agent_filters_noise_messages():
    ...
```

## Audit/Security + Coverage

- We SHOULD use `pyright` for static type checking, especially when type-driven behavior crosses package boundaries
- We SHOULD use `pytest-cov` when we need to measure coverage for Python code under test
- We SHOULD use `pip-audit` to audit Python dependencies for known security vulnerabilities before release or in CI
- We SHOULD keep type checking, coverage, and vulnerability checks in the same quality gate family as linting and tests

## Dependencies

### Package Management
- We SHOULD use `pyproject.toml` for project configuration
- We SHOULD use `uv` or `poetry` for dependency management
- We MUST pin dependencies with lock files
- We SHOULD separate dev dependencies from runtime dependencies

### Recommended Packages
| Purpose | Package |
|---------|---------|
| CLI | typer, rich |
| Validation | pydantic |
| HTTP | httpx |
| Testing | pytest, pytest-cov |
| Mocking | pytest-mock |
| YAML | pyyaml |
| OpenTelemetry | opentelemetry-sdk |

## Configuration

### Settings Pattern
- We SHOULD use Pydantic Settings for configuration
- We SHOULD support both YAML files and environment variables
- We MUST NOT hardcode configuration values

```python
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    org_config_path: str
    prompts_directory: str
    otel_exporter_endpoint: str | None = None

    class Config:
        env_file = ".env"
```

## Error Handling

- We SHOULD use explicit exception types
- We SHOULD create domain-specific exceptions when needed
- We MUST NOT use bare `except:` clauses
- We SHOULD use `contextlib.suppress()` for intentional ignoring

```python
class ScoutAgentError(Exception):
    """Base exception for Scout Agent errors."""
    pass

class SlackFetchError(ScoutAgentError):
    """Error fetching data from Slack."""
    pass
```

## Code Style

### Formatting
- We MUST use `ruff` for linting and formatting
- We SHOULD use 88 character line length (black default)
- We SHOULD use double quotes for strings

### Imports
- We SHOULD use absolute imports
- We SHOULD group imports: stdlib, third-party, local
- We SHOULD use `ruff` to sort imports

## Async/Await

- We SHOULD use `async`/`await` for I/O-bound operations
- We SHOULD use `asyncio.gather()` for concurrent operations
- We SHOULD use `httpx.AsyncClient` for HTTP requests
