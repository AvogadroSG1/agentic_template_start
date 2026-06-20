# mkproj — Product Requirements Document

**Status:** canonical · 2026-06-20 · **Author:** Peter O'Connor with Claude Code (databricks-claude-opus-4-8)

This is the canonical statement of *why* `mkproj` exists, *who* it is for, and *what* it must
achieve. It describes intent and outcomes only — **no mechanism**. For *how* the system works,
see [`SPEC.md`](./SPEC.md). For *why specific decisions were made*, see [`docs/adr/`](./adr/).
For *domain vocabulary*, see [`CONTEXT.md`](../CONTEXT.md).

---

## 1. Problem

Every new project begins with 10–15 minutes of identical manual setup: `git init`, copying a
known-good agent/beads configuration from a prior project, pruning unwanted skills, re-adding
starter skills, linking the agent file, installing and configuring beads with known-good
permissions, and wiring up quality gates. This is pure, repeatable overhead — and because it
is done by hand each time, it drifts: every repo ends up slightly different, and the safety and
quality wiring is only as good as the last copy-paste.

## 2. Goal

**One command turns an empty directory into a fully configured, AI-native project —
indistinguishable from what the author would have built by hand, with zero follow-up
configuration.**

The single canonical acceptance test for the whole product: *run the command in an empty
folder; the result must be a complete, working project with no manual steps after.*

## 3. Users

| User | What they need from mkproj |
|---|---|
| **The author (primary)** | Start a new project in one command, with their vetted standards, security floor, and quality gates already enforced — on their own machine. |
| **A collaborator / junior engineer** | Clone a scaffolded repo and have it open cleanly and work, even without `mkproj` installed. Be unable to accidentally select an unsupported or unvetted configuration. |
| **The maintainer (the author, in a second role)** | Refresh the vendored upstream snapshots under control, trusting that a diff means "upstream changed" rather than tool noise — without re-vetting the opinions layered on top. |
| **Coding agents (Claude Code, Codex)** | Operate inside the scaffolded repo under a consistent safety floor, with the same context and skills available, regardless of which agent. |

## 4. What success looks like (product outcomes)

1. **Zero-touch day one.** After the command runs, the repo is a working project: it builds,
   it has at least one real passing test, its quality gates run locally and in CI, and its
   safety floor is active — before a line of feature code is written.
2. **Indistinguishable from hand-built.** No scaffolding residue, no "TODO: wire this up," no
   half-configured tooling. A reviewer cannot tell it was generated.
3. **Safe by default for agents and humans.** Irreversible and secret-exposing actions are
   blocked; convenient actions run without friction. The floor holds in every permission mode
   and for both supported agents.
4. **Consistent, not drifting.** Every repo gets the same vetted standards from one canonical
   source, and can be refreshed from it — the standards cannot rot independently per repo.
5. **Offline and reproducible.** Creating a project never depends on the network or on
   external services being up; the same inputs always produce the same project.
6. **Honest about its limits.** Where a safety guarantee cannot be fully enforced, the gap is
   documented rather than pretended-closed.

## 5. Scope

### In scope for v1

- One-command initialization of a new project, fully configured.
- A catalog of **six vetted stacks** across **three languages** — Go, Python, and C# — each
  of which produces a project that runs end-to-end with a real test.
- A safety floor (block the irreversible and the secret-exposing) and a convenience allowlist,
  active for both Claude Code and Codex.
- Quality gates (format, lint, test) enforced identically locally and in CI.
- Optional remote publishing as part of the standard path.
- A maintainer-only path to refresh the vendored upstreams.

### Explicitly out of scope for v1

- **Languages without a written guideline file** (TypeScript, Rust, Bash). A stack ships only
  after its language standard is written down; expansion is follow-on work.
- **Initializing into a non-empty directory** (no `--force` / `--in-place`).
- **Auto-deleting a remote** on any failure (an irreversible outward action mkproj never takes).
- A graphical UI, a hosted service, or a plugin system.

## 6. Guiding principles (the "why" behind the product)

- **The opinions are the value.** The inert scaffolding underneath is commodity; the worth is
  in the vetted, enforced standards layered on top.
- **One source of truth per concern.** Standards, decisions, vocabulary, and behavior each have
  exactly one authoritative home; nothing is restated in two places that can drift.
- **Fail loud, name the thing.** Every failure stops at the failing step and tells the user
  exactly what to do next.
- **Defense in depth, honestly bounded.** Multiple independent safety layers, with the
  irreducible gaps written down.
- **Value visible at every increment.** Each unit of delivered work produces something a user
  can see or do that they could not before.

## 7. Relationship to the rest of the documentation

| Question | Authoritative document |
|---|---|
| Why does this product exist / what must it achieve? | **This PRD** |
| How does the system actually work? | [`SPEC.md`](./SPEC.md) |
| Why was a specific decision made (and what was rejected)? | [`docs/adr/`](./adr/) (ADR-0001 … ADR-0009) |
| What does a domain term mean? | [`CONTEXT.md`](../CONTEXT.md) |

---

*Authored By Peter O'Connor with Assistance from Claude Code (databricks-claude-opus-4-8) · 2026-06-20 · mkproj product requirements*
