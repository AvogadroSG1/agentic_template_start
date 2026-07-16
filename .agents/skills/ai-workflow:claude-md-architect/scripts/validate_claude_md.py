#!/usr/bin/env python3
"""
CLAUDE.md Validator

Analyzes a CLAUDE.md file against best practices from the THREE PILLARS framework.
Provides scoring, anti-pattern detection, and actionable recommendations.

Usage:
    python validate_claude_md.py /path/to/CLAUDE.md [--json]

Exit codes:
    0  - Excellent (score >= 13)
    1  - Error running validation
    10 - Good (score 10-12)
    11 - Needs work (score < 10)
"""

import argparse
import json
import re
import sys
from dataclasses import dataclass, field, asdict
from pathlib import Path
from typing import Optional


@dataclass
class PillarCoverage:
    """Coverage assessment for a single pillar."""
    name: str
    score: int  # 0-3
    evidence: list[str] = field(default_factory=list)
    missing: list[str] = field(default_factory=list)


@dataclass
class AntiPattern:
    """Detected anti-pattern."""
    name: str
    description: str
    lines: list[int] = field(default_factory=list)
    recommendation: str = ""


@dataclass
class ValidationResult:
    """Complete validation result."""
    file_path: str
    line_count: int
    estimated_instructions: int
    what_coverage: PillarCoverage
    why_coverage: PillarCoverage
    how_coverage: PillarCoverage
    universal_applicability: float
    anti_patterns: list[AntiPattern] = field(default_factory=list)
    total_score: int = 0
    max_score: int = 15
    verdict: str = ""
    recommendations: list[str] = field(default_factory=list)


def count_instructions(content: str) -> int:
    """Estimate instruction count from content."""
    lines = content.split('\n')
    instruction_count = 0
    in_code_block = False

    for line in lines:
        stripped = line.strip()

        # Skip code blocks
        if stripped.startswith('```'):
            in_code_block = not in_code_block
            continue

        if in_code_block:
            continue

        # Skip empty lines and pure headers
        if not stripped or stripped.startswith('#'):
            continue

        # Skip table separators
        if re.match(r'^[\|\-\s:]+$', stripped):
            continue

        # Count lines with actual content as ~1 instruction
        # Bullet points and table rows count as instructions
        if stripped.startswith(('-', '*', '|')) or len(stripped) > 10:
            instruction_count += 1

    return instruction_count


def assess_what_pillar(content: str) -> PillarCoverage:
    """Assess WHAT pillar coverage (technology stack, structure, architecture)."""
    coverage = PillarCoverage(name="WHAT", score=0)

    # Check for technology stack
    tech_patterns = [
        r'(?:technologies?|tech stack|stack)',
        r'(?:\.NET|Python|JavaScript|TypeScript|Go|Rust|Java)',
        r'(?:React|Vue|Angular|ASP\.NET|FastAPI|Django|Express)',
        r'(?:PostgreSQL|MySQL|MongoDB|Redis|Elasticsearch)',
    ]

    for pattern in tech_patterns:
        if re.search(pattern, content, re.IGNORECASE):
            if "technology" not in [e.lower() for e in coverage.evidence]:
                coverage.evidence.append("Technology stack documented")
            break
    else:
        coverage.missing.append("Technology stack not clearly documented")

    # Check for project structure
    structure_patterns = [
        r'(?:structure|directory|directories|folders?|layout)',
        r'(?:src/|lib/|tests?/|packages?/)',
        r'(?:overview|architecture)',
    ]

    for pattern in structure_patterns:
        if re.search(pattern, content, re.IGNORECASE):
            coverage.evidence.append("Project structure referenced")
            break
    else:
        coverage.missing.append("Project structure not documented")

    # Check for architecture patterns
    arch_patterns = [
        r'(?:monorepo|microservice|monolith)',
        r'(?:API|service|controller|model)',
        r'(?:architecture|pattern)',
    ]

    for pattern in arch_patterns:
        if re.search(pattern, content, re.IGNORECASE):
            coverage.evidence.append("Architecture patterns mentioned")
            break
    else:
        coverage.missing.append("Architecture patterns not described")

    # Score: 3 points max
    coverage.score = len(coverage.evidence)
    return coverage


def assess_why_pillar(content: str) -> PillarCoverage:
    """Assess WHY pillar coverage (purpose, context, component roles)."""
    coverage = PillarCoverage(name="WHY", score=0)

    # Check for project purpose
    purpose_patterns = [
        r'(?:provides?|enables?|allows?|helps?)',
        r'(?:purpose|goal|objective)',
        r'(?:this (?:is|project|service|app))',
    ]

    for pattern in purpose_patterns:
        if re.search(pattern, content, re.IGNORECASE):
            coverage.evidence.append("Project purpose explained")
            break
    else:
        coverage.missing.append("Project purpose not clearly stated")

    # Check for component descriptions
    component_patterns = [
        r'(?:responsible for|handles|manages)',
        r'(?:component|module|service)\s*[-:]\s*\w+',
        r'(?:\*\*\w+\*\*\s*[-:]\s*)',
    ]

    for pattern in component_patterns:
        if re.search(pattern, content, re.IGNORECASE):
            coverage.evidence.append("Component roles described")
            break
    else:
        coverage.missing.append("Component responsibilities not documented")

    # Check for business context
    context_patterns = [
        r'(?:user|customer|team|developer)',
        r'(?:workflow|process|integration)',
        r'(?:production|deployment|environment)',
    ]

    for pattern in context_patterns:
        if re.search(pattern, content, re.IGNORECASE):
            coverage.evidence.append("Business context provided")
            break
    else:
        coverage.missing.append("Business context missing")

    coverage.score = len(coverage.evidence)
    return coverage


def assess_how_pillar(content: str) -> PillarCoverage:
    """Assess HOW pillar coverage (commands, verification, testing)."""
    coverage = PillarCoverage(name="HOW", score=0)

    # Check for build commands
    build_patterns = [
        r'(?:dotnet build|npm (?:run )?build|make|cargo build)',
        r'(?:build(?:ing)?|compile|compilation)',
        r'```(?:bash|sh|shell)?\s*\n[^\`]*(?:build|compile)',
    ]

    for pattern in build_patterns:
        if re.search(pattern, content, re.IGNORECASE):
            coverage.evidence.append("Build commands documented")
            break
    else:
        coverage.missing.append("Build commands not documented")

    # Check for test commands
    test_patterns = [
        r'(?:dotnet test|npm test|pytest|jest|vitest)',
        r'(?:test(?:ing)?|spec)',
        r'```(?:bash|sh|shell)?\s*\n[^\`]*test',
    ]

    for pattern in test_patterns:
        if re.search(pattern, content, re.IGNORECASE):
            coverage.evidence.append("Test commands documented")
            break
    else:
        coverage.missing.append("Test commands not documented")

    # Check for run/execution commands
    run_patterns = [
        r'(?:dotnet run|npm start|python|node)',
        r'(?:run(?:ning)?|start|execute)',
        r'```(?:bash|sh|shell)?\s*\n[^\`]*(?:run|start)',
    ]

    for pattern in run_patterns:
        if re.search(pattern, content, re.IGNORECASE):
            coverage.evidence.append("Run commands documented")
            break
    else:
        coverage.missing.append("Run/execution commands not documented")

    coverage.score = len(coverage.evidence)
    return coverage


def assess_universal_applicability(content: str) -> tuple[float, list[tuple[str, int]]]:
    """
    Assess what percentage of content is universally applicable.
    Returns (percentage, list of (issue, line_number) tuples).
    """
    issues = []
    lines = content.split('\n')

    task_specific_patterns = [
        (r'migration', "Database migration instructions (task-specific)"),
        (r'deploy(?:ment)?(?:\s+to)?(?:\s+prod)', "Deployment instructions (task-specific)"),
        (r'(?:indent|spacing|tabs?\s+vs)', "Code style/indentation rules (use linter)"),
        (r'(?:naming convention|camelCase|snake_case)', "Naming conventions (use linter)"),
        (r'(?:semicolon|trailing comma)', "Style rules (use linter)"),
        (r'troubleshoot', "Troubleshooting guides (move to separate docs)"),
        (r'(?:onboard|getting started|first time)', "Onboarding content (move to README)"),
    ]

    non_universal_lines = 0
    total_content_lines = 0
    in_code_block = False

    for i, line in enumerate(lines, 1):
        stripped = line.strip()

        if stripped.startswith('```'):
            in_code_block = not in_code_block
            continue

        if in_code_block or not stripped or stripped.startswith('#'):
            continue

        total_content_lines += 1

        for pattern, description in task_specific_patterns:
            if re.search(pattern, stripped, re.IGNORECASE):
                issues.append((description, i))
                non_universal_lines += 1
                break

    if total_content_lines == 0:
        return 1.0, issues

    applicability = (total_content_lines - non_universal_lines) / total_content_lines
    return applicability, issues


def detect_anti_patterns(content: str) -> list[AntiPattern]:
    """Detect common anti-patterns in CLAUDE.md."""
    anti_patterns = []
    lines = content.split('\n')

    # Anti-pattern: Embedded code samples (>10 lines of code)
    code_block_start = None
    code_block_lines = 0
    in_code_block = False

    for i, line in enumerate(lines, 1):
        if line.strip().startswith('```'):
            if in_code_block:
                if code_block_lines > 10:
                    anti_patterns.append(AntiPattern(
                        name="Embedded code samples",
                        description=f"Code block with {code_block_lines} lines at line {code_block_start}",
                        lines=list(range(code_block_start, i + 1)),
                        recommendation="Use file:line references instead of embedding code"
                    ))
                in_code_block = False
                code_block_lines = 0
            else:
                in_code_block = True
                code_block_start = i
        elif in_code_block:
            code_block_lines += 1

    # Anti-pattern: Style guidelines
    style_lines = []
    for i, line in enumerate(lines, 1):
        if re.search(r'(?:indent|spacing|style|format|lint|prettier|eslint)', line, re.IGNORECASE):
            if not re.search(r'(?:run|command|hook)', line, re.IGNORECASE):
                style_lines.append(i)

    if style_lines:
        anti_patterns.append(AntiPattern(
            name="Style guidelines in CLAUDE.md",
            description="LLMs are in-context learners; use linters and hooks instead",
            lines=style_lines,
            recommendation="Configure linter hooks, remove style instructions"
        ))

    # Anti-pattern: Verbose explanations before commands
    for i, line in enumerate(lines, 1):
        if re.search(r'(?:first|before|make sure|ensure).{20,}(?:install|download|run)', line, re.IGNORECASE):
            anti_patterns.append(AntiPattern(
                name="Verbose command explanations",
                description=f"Line {i} has excessive explanation before command",
                lines=[i],
                recommendation="Use terse, copy-paste-ready commands"
            ))

    return anti_patterns


def calculate_conciseness_score(line_count: int) -> int:
    """Score conciseness (25% weight, max 3 points)."""
    if line_count < 60:
        return 3
    elif line_count < 150:
        return 2
    elif line_count < 300:
        return 1
    return 0


def validate_claude_md(file_path: str) -> ValidationResult:
    """Main validation function."""
    path = Path(file_path)

    if not path.exists():
        raise FileNotFoundError(f"File not found: {file_path}")

    content = path.read_text(encoding='utf-8')
    lines = content.split('\n')
    line_count = len(lines)

    # Core assessments
    instruction_count = count_instructions(content)
    what_coverage = assess_what_pillar(content)
    why_coverage = assess_why_pillar(content)
    how_coverage = assess_how_pillar(content)
    applicability, applicability_issues = assess_universal_applicability(content)
    anti_patterns = detect_anti_patterns(content)

    # Calculate scores
    conciseness_score = calculate_conciseness_score(line_count)
    applicability_score = 2 if applicability >= 0.9 else (1 if applicability >= 0.8 else 0)

    total_score = (
        conciseness_score +  # 0-3
        what_coverage.score +  # 0-3
        why_coverage.score +  # 0-3
        how_coverage.score +  # 0-3
        applicability_score +  # 0-2 (scaled to match weight)
        (1 if not anti_patterns else 0)  # Bonus for no anti-patterns
    )

    # Determine verdict
    if total_score >= 13:
        verdict = "Excellent - minimal changes needed"
    elif total_score >= 10:
        verdict = "Good - minor optimization opportunities"
    elif total_score >= 7:
        verdict = "Needs work - apply progressive disclosure"
    else:
        verdict = "Significant rewrite recommended"

    # Generate recommendations
    recommendations = []

    if line_count >= 150:
        recommendations.append(f"Reduce from {line_count} lines to <60 using progressive disclosure")

    for issue, line_num in applicability_issues:
        recommendations.append(f"Line {line_num}: {issue}")

    for anti_pattern in anti_patterns:
        recommendations.append(f"{anti_pattern.name}: {anti_pattern.recommendation}")

    if not what_coverage.evidence:
        recommendations.append("Add technology stack and project structure (WHAT pillar)")

    if not why_coverage.evidence:
        recommendations.append("Add project purpose and component roles (WHY pillar)")

    if not how_coverage.evidence:
        recommendations.append("Add build, test, and run commands (HOW pillar)")

    return ValidationResult(
        file_path=str(path.absolute()),
        line_count=line_count,
        estimated_instructions=instruction_count,
        what_coverage=what_coverage,
        why_coverage=why_coverage,
        how_coverage=how_coverage,
        universal_applicability=applicability,
        anti_patterns=anti_patterns,
        total_score=total_score,
        max_score=15,
        verdict=verdict,
        recommendations=recommendations
    )


def format_pillar_bar(score: int, max_score: int = 3) -> str:
    """Create a visual progress bar for pillar coverage."""
    filled = '█' * (score * 3)
    empty = '░' * ((max_score - score) * 3)
    return filled + empty


def print_report(result: ValidationResult) -> None:
    """Print formatted validation report."""
    print("\nCLAUDE.md Audit Report")
    print("══════════════════════\n")

    print(f"File: {result.file_path}")
    print(f"Lines: {result.line_count} (target: <60, max: 300)")
    print(f"Estimated Instructions: ~{result.estimated_instructions}")
    print()

    print("THREE PILLARS Coverage:")
    print(f"  WHAT: {format_pillar_bar(result.what_coverage.score)} ", end="")
    if result.what_coverage.evidence:
        print(f"({', '.join(result.what_coverage.evidence)})")
    else:
        print("(Missing)")

    print(f"  WHY:  {format_pillar_bar(result.why_coverage.score)} ", end="")
    if result.why_coverage.evidence:
        print(f"({', '.join(result.why_coverage.evidence)})")
    else:
        print("(Missing)")

    print(f"  HOW:  {format_pillar_bar(result.how_coverage.score)} ", end="")
    if result.how_coverage.evidence:
        print(f"({', '.join(result.how_coverage.evidence)})")
    else:
        print("(Missing)")
    print()

    print(f"Universal Applicability: {result.universal_applicability:.0%}")
    if result.universal_applicability < 1.0:
        for ap in result.anti_patterns:
            if ap.lines:
                print(f"  ⚠ {ap.description}")
    print()

    if result.anti_patterns:
        print("Anti-Patterns Detected:")
        for ap in result.anti_patterns:
            print(f"  ⚠ {ap.name}: {ap.description}")
        print()

    print(f"Score: {result.total_score}/{result.max_score} ({result.verdict})")
    print()

    if result.recommendations:
        print("Recommendations:")
        for i, rec in enumerate(result.recommendations, 1):
            print(f"  {i}. {rec}")
        print()


def main():
    parser = argparse.ArgumentParser(
        description="Validate CLAUDE.md against best practices"
    )
    parser.add_argument("file_path", help="Path to CLAUDE.md file")
    parser.add_argument("--json", action="store_true", help="Output as JSON")

    args = parser.parse_args()

    try:
        result = validate_claude_md(args.file_path)

        if args.json:
            # Convert to JSON-serializable dict
            output = asdict(result)
            print(json.dumps(output, indent=2))
        else:
            print_report(result)

        # Exit codes based on score
        if result.total_score >= 13:
            sys.exit(0)  # Excellent
        elif result.total_score >= 10:
            sys.exit(10)  # Good
        else:
            sys.exit(11)  # Needs work

    except FileNotFoundError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
