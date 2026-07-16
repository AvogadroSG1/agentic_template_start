# CLAUDE.md Best Practices - Deep Dive

Research-backed guidance from HumanLayer and empirical testing.

## The Fundamental Truth

> "LLMs are stateless functions whose weights freeze at inference time."

This means:
1. Claude knows **nothing** about your codebase at session start
2. Essential context MUST be communicated each session
3. CLAUDE.md is the standard delivery mechanism

## Instruction Limits (Research-Backed)

| Model Type | Instruction Limit | Degradation Pattern |
|------------|-------------------|---------------------|
| Frontier LLMs | ~150-200 | Linear degradation |
| Smaller models | ~50-100 | Exponential degradation |

**Critical insight:** Claude Code's system prompt already contains ~50 instructions. This means your CLAUDE.md budget is effectively **~100-150 instructions**.

## Context Window Dynamics

Research shows:
- LLMs bias toward **prompt peripheries** (beginning and end)
- Instruction-following quality decreases **uniformly** as count increases
- **Irrelevant context degrades ALL instruction-following**, not just related tasks

This is why Claude sometimes ignores your CLAUDE.md—not because it "doesn't care," but because your file contains too much noise.

## Strategic Ignoring

Claude Code includes this system reminder about CLAUDE.md:

> "IMPORTANT: this context may or may not be relevant to your tasks. You should not respond to this context unless it is highly relevant."

Anthropic discovered that **filtering bad instructions improved overall harness performance**. Non-universally-applicable instructions get ignored uniformly.

**Implication:** Task-specific instructions in CLAUDE.md don't just waste space—they actively degrade instruction-following for everything else.

## The THREE PILLARS Framework

### WHAT: Document the Map

| Include | Exclude |
|---------|---------|
| Technology stack with versions | Historical decisions |
| Project structure (key directories) | Comprehensive file listings |
| Architecture patterns in use | UML diagrams |
| Build/test framework names | All possible configurations |

**Good WHAT:**
```markdown
- **.NET 8.0** with ASP.NET Core
- **Elasticsearch** via StackExchange.Elastic
- **MSTest** with NSubstitute
```

**Bad WHAT:**
```markdown
This project uses Microsoft .NET Framework version 8.0.100 which was released
in November 2023 and provides the latest C# 12 features including...
[continues for 50 lines]
```

### WHY: Explain Purpose

| Include | Exclude |
|---------|---------|
| Project's core purpose (1-2 sentences) | Full product vision document |
| Why key technologies were chosen | Alternative technologies considered |
| Component responsibilities | Implementation details |

**Good WHY:**
```markdown
Search microservice providing Elasticsearch-based lexical and semantic search for Stack Overflow.
```

**Bad WHY:**
```markdown
This service was created in Q3 2023 as part of the platform modernization initiative
to replace the legacy search system that was built in 2015 using Lucene.NET...
[continues for 100 lines]
```

### HOW: Provide Execution

| Include | Exclude |
|---------|---------|
| Primary build command | Every possible build flag |
| Primary test command + common filter | Complete test strategy document |
| Primary run command | Deployment procedures |
| Environment prerequisites | Troubleshooting guides |

**Good HOW:**
```bash
dotnet build StackOverflow.Search.sln
dotnet test
dotnet test --filter "FullyQualifiedName~TestName"
```

**Bad HOW:**
```bash
# To build the project, first ensure you have .NET 8.0 SDK installed.
# You can check this by running `dotnet --version`.
# If you don't have it installed, visit https://dotnet.microsoft.com/download
# and download the latest SDK for your platform...
[continues for 50 lines before showing the actual command]
```

## Progressive Disclosure Patterns

### Pattern 1: Task-Specific Documentation

```
project/
├── CLAUDE.md                    # Core context only
└── agent_docs/
    ├── database-migrations.md   # When Claude needs to run migrations
    ├── api-development.md       # When Claude is building APIs
    ├── testing-strategy.md      # When Claude is writing tests
    └── deployment.md            # When Claude is deploying
```

Claude will find and read these files when the task requires them.

### Pattern 2: File References

Instead of embedding code:

```markdown
## Authentication Pattern

See implementation at `src/auth/handler.ts:45-80`
```

This:
- Never goes stale
- Doesn't bloat context
- Points Claude to authoritative source

### Pattern 3: Hierarchical CLAUDE.md

```
~/.claude/CLAUDE.md              # Personal standards (RFC 2119, 12 Factor)
~/code/CLAUDE.md                 # Workspace-level patterns
~/code/project/CLAUDE.md         # Project-specific context
```

**Don't repeat.** Each level inherits from above.

## Anti-Pattern: LLM as Linter

> "Never send an LLM to do a linter's job."

**Why this fails:**
- LLMs are slow and expensive for deterministic tasks
- Style guidelines bloat context window
- Instruction-following degrades for everything else
- LLMs are **in-context learners**—they follow existing patterns naturally

**Instead:**
1. Configure proper linters (ESLint, Biome, Ruff, etc.)
2. Use Claude Code hooks to run formatters automatically:
```json
{
  "hooks": {
    "post-tool": {
      "write": "./scripts/format-code.sh"
    }
  }
}
```
3. Trust Claude to follow existing patterns in the codebase

## Anti-Pattern: Auto-Generation

Don't use `/init` or similar commands to auto-generate CLAUDE.md.

**Why this fails:**
- Creates bloated, generic content
- Includes things that don't matter for YOUR project
- Missing context that's obvious to you but not to Claude
- CLAUDE.md is "the highest leverage point of the harness"—a bad line cascades through everything

**Instead:** Craft manually with intent. Every line should pass this test: "Will this instruction help Claude with >90% of tasks in this project?"

## Measurement Framework

### Line Count

| Lines | Assessment |
|-------|------------|
| <60 | Excellent - likely well-optimized |
| 60-150 | Good - check for task-specific content |
| 150-300 | Needs review - progressive disclosure recommended |
| >300 | Critical - significant rewrite needed |

### Instruction Density

Rough heuristic: 1 instruction per 1-2 lines of content (excluding code blocks).

A 60-line file ≈ 30-60 instructions ≈ leaves ~90-120 instructions of budget.

### Universal Applicability Test

For each instruction, ask: "Is this relevant to >90% of tasks?"

| Instruction | Universal? | Action |
|-------------|------------|--------|
| "Build with `dotnet build`" | Yes | Keep |
| "When writing tests, use MSTest" | Yes | Keep |
| "Database migrations use Flyway" | No | Move to `agent_docs/database.md` |
| "Use 2-space indentation" | No | Use linter instead |

## References

- Source: https://www.humanlayer.dev/blog/writing-a-good-claude-md
- Claude Code Documentation: https://docs.anthropic.com/claude-code
