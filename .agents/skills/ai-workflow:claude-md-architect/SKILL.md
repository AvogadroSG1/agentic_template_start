---
name: claude-md-architect
description: Create and improve CLAUDE.md files using the THREE PILLARS framework (WHAT, WHY, HOW) and research-backed best practices. Use when starting a new project, auditing existing CLAUDE.md files, or when Claude seems to ignore project context.
version: 1.0.0
license: MIT
metadata:
  source: humanlayer.dev/blog/writing-a-good-claude-md
---

# CLAUDE.md Architect

Create high-impact CLAUDE.md files that Claude actually follows. Based on research showing LLMs can follow ~150-200 instructions consistently, and Claude Code's system prompt already uses ~50—this skill helps you maximize the remaining budget.

## Triggers

- `create CLAUDE.md` - Create a new CLAUDE.md for the current project
- `improve CLAUDE.md` - Audit and optimize existing CLAUDE.md
- `audit CLAUDE.md` - Score CLAUDE.md against best practices
- `Claude ignores my instructions` - Diagnose why context isn't being followed
- `optimize project context` - Restructure for better instruction-following

## Quick Reference

| Pillar | Purpose | Key Question |
|--------|---------|--------------|
| **WHAT** | Technology stack, project structure, codebase architecture | "What map does Claude need?" |
| **WHY** | Project purpose, component functions, business context | "Why does each piece exist?" |
| **HOW** | Execution instructions, verification methods, testing procedures | "How does Claude do the work?" |

| Metric | Target | Why |
|--------|--------|-----|
| Line count | <60 ideal, <300 max | Instruction-following degrades with volume |
| Instruction count | <100 (leaving headroom) | ~150-200 limit, minus Claude's ~50 system instructions |
| Universal applicability | 100% | Non-universal instructions get ignored uniformly |

## Process

### Phase 1: Discovery

Scan the project to understand its structure and existing context.

- Identify languages, frameworks, and build tools
- Check for existing CLAUDE.md or related documentation
- Detect project type: monorepo, single-project, or workspace

### Phase 2: Three Pillars Analysis

Gather information for each pillar.

**WHAT** (Map the codebase):
- Key directories and their purposes
- Technology stack with versions
- Architecture patterns in use

**WHY** (Explain the context):
- Project's purpose and target users
- Why specific technologies were chosen
- Component responsibilities

**HOW** (Execution guidance):
- Build commands (copy-paste ready)
- Test commands with common filters
- Run/execution commands

### Phase 3: Optimization

Refine content for maximum instruction-following.

- Apply progressive disclosure (link, don't embed)
- Remove task-specific guidance → separate `agent_docs/` files
- Eliminate linting rules → use hooks instead
- Verify universal applicability of each instruction
- Score against best practices rubric

### Phase 4: Output

Generate the final CLAUDE.md and report metrics.

- Write minimal, high-impact content
- Report: line count, instruction estimate, pillar coverage
- Suggest progressive disclosure files if needed

## Commands

| Command | Action |
|---------|--------|
| `/claude-md-architect create` | Create new CLAUDE.md for current project |
| `/claude-md-architect audit` | Score existing CLAUDE.md, suggest improvements |
| `/claude-md-architect optimize` | Rewrite existing CLAUDE.md following best practices |

## CLAUDE.md Structure Template

```markdown
# CLAUDE.md

Brief description of what this project is (1-2 sentences max).

## Key Technologies

- **Backend**: [Language] [Version], [Framework]
- **Frontend**: [Framework], [Bundler]
- **Data**: [Database], [Cache], [Search]

## Common Commands

### Building
\`\`\`bash
[primary build command]
\`\`\`

### Testing
\`\`\`bash
[test command]
[test command with common filter]
\`\`\`

### Running
\`\`\`bash
[run command]
\`\`\`

## Architecture Overview

[2-3 sentences on key architectural patterns]

## Development Guidelines

[Only universally-applicable guidelines, 3-5 bullet points max]

## Important Paths

- **Main entry**: `path/to/main`
- **Config**: `path/to/config`
- **Tests**: `path/to/tests`
```

## Anti-Patterns to Avoid

| Anti-Pattern | Why It Fails | Instead |
|--------------|--------------|---------|
| Auto-generating with `/init` | Creates bloated, generic content | Craft manually with intent |
| Including style guidelines | LLMs are in-context learners; they follow existing patterns | Use hooks + linters |
| Task-specific instructions | Distracts from unrelated work, gets ignored | Separate `agent_docs/` files |
| Embedding code samples | Becomes stale, bloats context | Use `file:line` references |
| Repeating global instructions | Wastes instruction budget | Use `~/.claude/CLAUDE.md` for globals |
| >300 lines | Instruction-following degrades uniformly | Split via progressive disclosure |

## Progressive Disclosure Pattern

Instead of cramming everything into CLAUDE.md, create targeted files:

```
project/
├── CLAUDE.md                    # Core WHAT/WHY/HOW (<60 lines)
└── agent_docs/                  # Task-specific guidance
    ├── building.md              # Detailed build instructions
    ├── testing.md               # Test strategies and fixtures
    ├── database-schema.md       # Schema documentation
    ├── api-patterns.md          # API conventions
    └── deployment.md            # Deployment procedures
```

Reference in CLAUDE.md:
```markdown
## Additional Documentation

For detailed guidance on specific tasks, see:
- Building: `agent_docs/building.md`
- Testing: `agent_docs/testing.md`
```

## Scoring Rubric

When auditing or creating, score against these criteria:

| Criterion | Weight | Excellent (3) | Good (2) | Needs Work (1) |
|-----------|--------|---------------|----------|----------------|
| **Conciseness** | 25% | <60 lines | 60-150 lines | >150 lines |
| **WHAT Coverage** | 20% | Complete tech stack + structure | Most covered | Missing key info |
| **WHY Coverage** | 20% | Clear purpose + component roles | Purpose clear | Unclear purpose |
| **HOW Coverage** | 20% | Copy-paste commands work | Most commands work | Commands incomplete |
| **Universal Applicability** | 15% | 100% universal | >80% universal | <80% universal |

**Score interpretation:**
- **13-15**: Excellent - minimal changes needed
- **10-12**: Good - minor optimization opportunities
- **7-9**: Needs work - apply progressive disclosure
- **<7**: Significant rewrite recommended

## Validation Script

Run after creating/updating:

```bash
python ~/.claude/skills/claude-md-architect/scripts/validate_claude_md.py /path/to/CLAUDE.md
```

Output includes:
- Line count and instruction estimate
- THREE PILLARS coverage assessment
- Universal applicability check
- Anti-pattern detection
- Overall score with recommendations

## Integration with Existing Patterns

This skill respects hierarchical CLAUDE.md loading:

1. **`~/.claude/CLAUDE.md`**: Global personal preferences, coding standards (RFC 2119, 12 Factor), collaboration guidelines
2. **`~/code/CLAUDE.md`** (if workspace): Multi-project overview, shared commands
3. **Project `CLAUDE.md`**: Project-specific WHAT/WHY/HOW

**Don't repeat globals.** If `~/.claude/CLAUDE.md` specifies RFC 2119 compliance, don't restate it in project CLAUDE.md.

## Examples

<details>
<summary><strong>Example: Minimal Effective CLAUDE.md (45 lines)</strong></summary>

```markdown
# CLAUDE.md

Search microservice providing Elasticsearch-based lexical and semantic search for Stack Overflow.

## Key Technologies

- **.NET 8.0** with ASP.NET Core
- **Elasticsearch** via StackExchange.Elastic
- **Docker** for containerized deployment
- **MSTest** with NSubstitute and Shouldly

## Common Commands

### Build & Test
\`\`\`bash
dotnet build StackOverflow.Search.sln
dotnet test
dotnet test --filter "FullyQualifiedName~YourTestName"
\`\`\`

### Run
\`\`\`bash
dotnet run --project src/Service/Service.csproj
\`\`\`

## Architecture

- **Service**: Main API (port 5000, Docker maps to 4999)
- **LexicalSearch**: Query building and Elasticsearch integration
- **Api.Client**: Auto-generated from OpenAPI spec

## Development Notes

- Nullable reference types enabled
- Central Package Management via `Directory.Packages.props`
- API changes: update `search-api.json`, rebuild SwaggerDefinition
```

</details>

<details>
<summary><strong>Example: Audit Output</strong></summary>

```
CLAUDE.md Audit Report
══════════════════════

File: /path/to/project/CLAUDE.md
Lines: 177 (target: <60, max: 300)
Estimated Instructions: ~85

THREE PILLARS Coverage:
  WHAT: ██████████ Complete (tech stack, structure, patterns)
  WHY:  ████████░░ Good (purpose clear, some component roles missing)
  HOW:  ██████████ Complete (build, test, run commands)

Universal Applicability: 72%
  ⚠ Lines 45-52: Database migration instructions (task-specific)
  ⚠ Lines 89-95: Code style guidelines (use linter instead)

Anti-Patterns Detected:
  ⚠ Embedded code samples at lines 102-130 (use file references)
  ⚠ Duplicates global instructions (RFC 2119 already in ~/.claude/CLAUDE.md)

Score: 9/15 (Needs Work)

Recommendations:
1. Extract database migration instructions to agent_docs/database.md
2. Remove code style guidelines, configure biome/eslint hooks instead
3. Replace embedded samples with file:line references
4. Remove duplicated RFC 2119 section (inherited from global)

Projected improvement: 177 → 52 lines, score 9 → 14
```

</details>

## Scripts

### validate_claude_md.py

Analyzes CLAUDE.md files against best practices.

```bash
python ~/.claude/skills/claude-md-architect/scripts/validate_claude_md.py /path/to/CLAUDE.md
python ~/.claude/skills/claude-md-architect/scripts/validate_claude_md.py /path/to/CLAUDE.md --json
```

**Exit Codes:**
| Code | Meaning |
|------|---------|
| 0 | Excellent (score ≥13) |
| 1 | Error running validation |
| 10 | Good (score 10-12) |
| 11 | Needs work (score <10) |

## Verification Checklist

After creating or updating a CLAUDE.md, verify:

- [ ] Line count is under 60 (ideal) or 300 (max)
- [ ] THREE PILLARS are covered (WHAT, WHY, HOW)
- [ ] All instructions are universally applicable
- [ ] No embedded code samples >10 lines (use file references)
- [ ] No style/formatting rules (use linters + hooks instead)
- [ ] No duplication of global `~/.claude/CLAUDE.md` content
- [ ] Build/test/run commands are copy-paste ready
- [ ] Validation script passes with score ≥10

## Extension Points

1. **Custom scoring weights**: Modify `validate_claude_md.py` to adjust pillar weights
2. **Additional anti-patterns**: Add regex patterns for project-specific issues
3. **Template variations**: Create domain-specific templates in `references/`
4. **CI integration**: Use validation script in pre-commit hooks or CI pipelines

## References

- [Best Practices Deep Dive](references/best-practices.md) - Full HumanLayer research summary

## Why This Matters

LLMs are stateless functions. Their weights freeze at inference time. They know **nothing** about your codebase unless you tell them. CLAUDE.md is your highest leverage point—a bad line here cascades through every research query, plan, and implementation.

The goal isn't comprehensive documentation. It's **minimum viable context** that Claude consistently follows.
