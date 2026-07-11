# Handoff: draft for `~/peter_code/ai_support/guidelines/typescript.md`

The frontend stacks (`vite-ts`, `sveltekit`, `angular`) ship for the `typescript` language,
so ADR-0010's conformance floor requires a canonical `typescript.md` guideline file at
`~/peter_code/ai_support/guidelines/typescript.md`. That path lives outside this repo on the
author's machine; this handoff carries a ready-to-install draft. The conformance test
(`test/guideline_conformance_test.go`) parses the MUST/SHOULD/PREFER lines below — the
backtick-quoted tool names are load-bearing needles; keep them if you rewrite the prose.

Install with:

```bash
mkdir -p ~/peter_code/ai_support/guidelines
cp -f docs/handoffs/typescript-guideline-draft.md ~/peter_code/ai_support/guidelines/typescript.md
```

---

# TypeScript / Frontend Guidelines

## Formatting

- Projects MUST format with `prettier`; framework-specific plugins
  (`prettier-plugin-svelte`) are used where the framework requires them.

## Linting

- Projects MUST lint with `eslint` flat config via `typescript-eslint`; Angular projects
  use `angular-eslint`, Svelte projects use `eslint-plugin-svelte`.

## Testing

- Projects MUST test with `vitest`. Angular runs it through `ng test` (the default unit-test
  runner since Angular 21); the others invoke it directly.
- Tests SHOULD stub network transport (`fetch`) rather than depend on a live backend.

## Type checking

- Projects MUST expose a `typecheck` script in `package.json` that fails on type errors:
  `tsc --noEmit` for Vite and Angular, `svelte-check` for SvelteKit.

## Auditing

- Projects MUST run `npm audit` (with `--audit-level=high`) as part of the ci gate.

## Structure

- Transport code SHOULD live in a dedicated api module (`src/api` or `src/lib/api`);
  screens and reusable UI stay separate (`src/pages`, `src/components`).
- Angular projects SHOULD PREFER standalone components and signals; RxJS stays reserved
  for genuinely event-streamed state.
