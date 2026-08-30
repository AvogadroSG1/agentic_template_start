# C# / .NET Guidelines

These guidelines apply to all C# and .NET development. They extend the general principles in `~/.claude/CLAUDE.md`.

## Language Features

### Types and Data
- We SHOULD use records over classes for immutable data models
- We SHOULD use `required` properties over constructor injection for DTOs/records
- We SHOULD use nullable reference types (`<Nullable>enable</Nullable>`)
- We SHOULD prefer `init` setters for immutable properties
- We SHOULD use pattern matching where it improves readability

### Visibility and Encapsulation
- We SHOULD use the lowest visibility modifiers possible
  - Prefer: `private` > `internal` > `protected internal` > `protected` > `public`
- We SHOULD use `file`-scoped types for implementation details
- We SHOULD use file-scoped namespaces (`namespace Foo;` not `namespace Foo { }`)

### Async/Await
- We MUST use `async`/`await` for I/O-bound operations
- We MUST NOT use `.Result` or `.Wait()` on tasks (causes deadlocks)
- We SHOULD use `ConfigureAwait(false)` in library code
- We SHOULD suffix async methods with `Async`

## Project Structure

### Solution Organization
- We SHOULD organize code into small, focused projects that separate concerns
- We MUST NOT create circular dependencies between projects
- We MUST NOT create "god" projects with catch-all shared utilities
- We SHOULD use `#region` to make code portions easier to filter when viewing.
- We SHOULD follow this layering pattern:
  ```
  Domain        - Core business logic, no external dependencies
  Application   - Use cases, orchestration, interfaces
  Infrastructure - External concerns (DB, API, file system)
  Presentation  - CLI, API, UI
  ```

### Naming Conventions
- Projects: `{SolutionName}.{Layer}` (e.g., `LeadershipChatCockpit.Domain`)
- Interfaces: `I{Name}` (e.g., `IScoutAgent`)
- Async methods: `{Name}Async` (e.g., `AnalyzeAsync`)
- Test projects: `{ProjectName}.Tests`

### Identifiers
- We SHOULD use UUIDv7 as the default identifier strategy
- We SHOULD use `Guid.CreateVersion7()` (native .NET 9+)
- We SHOULD NOT use auto-increment integers for distributed systems

## Testing

### Framework and Style
- We PREFER xUnit as the testing framework
- We SHOULD use FluentAssertions for readable assertions
- We SHOULD use NSubstitute for mocking
- We SHOULD follow Arrange-Act-Assert (AAA) pattern
- WE SHOULD use BDD patterns.

### Test Organization
- Unit tests: `{ClassName}Tests.cs`
- Test methods: `{MethodName}_{Scenario}_{ExpectedResult}`
  ```csharp
  [Fact]
  public void Parse_ValidJson_ReturnsEvidenceLog()
  ```

### Coverage
- We MUST have tests for all business logic
- We SHOULD have integration tests for external boundaries
- We MAY skip tests for trivial code (simple mappings, pass-through)

## Audit/Security + Coverage

- We SHOULD use `StyleCop.Analyzers` to enforce the style and design rules that the formatter does not cover
- We MUST use `dotnet format` to normalize code style and formatting before review or release
- We SHOULD use `coverlet` when collecting coverage for testable code paths
- We SHOULD use `dotnet list package --vulnerable` to audit NuGet dependencies for known security vulnerabilities
- We SHOULD keep formatting, coverage, and vulnerability checks in the same quality gate family as tests

## Dependencies

### Package Management
- We SHOULD use Central Package Management (`Directory.Packages.props`)
- We MUST pin package versions explicitly
- We SHOULD audit packages for security vulnerabilities regularly

### Recommended Packages
| Purpose | Package |
|---------|---------|
| CLI | Spectre.Console |
| JSON | System.Text.Json (prefer over Newtonsoft) |
| YAML | YamlDotNet |
| Logging | Microsoft.Extensions.Logging |
| DI | Microsoft.Extensions.DependencyInjection |
| Result Pattern | ErrorOr |
| OpenTelemetry | OpenTelemetry.Extensions.Hosting |
| Testing | xUnit, FluentAssertions, NSubstitute |
| Validation | FluentValidation |

## Configuration

### Settings Pattern
- We SHOULD use `IOptions<T>` pattern for configuration
- We SHOULD use strongly-typed configuration classes
- We MUST NOT hardcode configuration values
- Configuration files SHOULD be YAML (converted to IConfiguration)

### Example
```csharp
public sealed record OrgConfig
{
    public required string Name { get; init; }
    public required string ContextFile { get; init; }
    public required IReadOnlyList<ChannelConfig> Channels { get; init; }
}
```

## Dependency Injection

- We SHOULD use dependency injection for all services
- We SHOULD use `Microsoft.Extensions.DependencyInjection` as the DI container
- We SHOULD register dependencies in a composition root (e.g., `Program.cs`)
- We SHOULD prefer constructor injection over property injection
- We SHOULD use appropriate lifetimes: `Singleton`, `Scoped`, `Transient`

## Error Handling

- We SHOULD use the Result pattern over exceptions for expected failures
- We SHOULD use ErrorOr package for Result pattern implementation
- We MUST use exceptions for unexpected/exceptional conditions
- We SHOULD create domain-specific exception types when needed
- We MUST NOT swallow exceptions silently

### Result Pattern Example
```csharp
using ErrorOr;

public ErrorOr<EvidenceLog> AnalyzeAsync(RawSlackData data)
{
    if (data.Messages.Count == 0)
        return Error.Validation("NoMessages", "No messages to analyze");

    // ... analysis logic
    return evidenceLog;
}

// Usage with Match
var result = await _scout.AnalyzeAsync(data);
return result.Match(
    success => ProcessSuccess(success),
    errors => HandleErrors(errors));

// Usage with IsError
if (result.IsError)
{
    _logger.LogWarning("Analysis failed: {Errors}", result.Errors);
    return;
}
var log = result.Value;
```

## Code Style

### General
- We SHOULD use expression-bodied members for simple operations
- We SHOULD use target-typed `new()` when type is obvious
- We SHOULD prefer `var` when type is obvious from context
- We MUST NOT use `var` when type is not obvious

### Example Record
```csharp
namespace LeadershipChatCockpit.Domain.Models;

public sealed record Signal
{
    public required string Category { get; init; }
    public required Severity Severity { get; init; }
    public required string VerbatimQuote { get; init; }
    public required string Speaker { get; init; }
    public required string SpeakerRole { get; init; }
    public required string ExecutiveAnalysis { get; init; }
    public required string SourceLink { get; init; }
    public string? RelatedContext { get; init; }
}

public enum Severity { Low, Medium, High }
```
