# Frontend dimension: typescript stacks reused as fragments under web/

**Status:** accepted · 2026-07-11

## Context

forge v1 shipped six backend-only stacks. The product now needs frontends that connect
either to third-party APIs or to a backend selected in the same `forge init` run — with a
specific language-native provider per backend (Go → templ + HTMX, Python → FastAPI +
Jinja2 + HTMX, dotnet → Blazor Interactive Auto) and three cross-cutting JS frontends
(SvelteKit, Angular, Vite + TypeScript) offered with any backend or standalone. Everything
must stay a walking skeleton (ADR-0005), keep one gate pipeline (ADR-0003), and keep the
vanilla/overlay refresh seam (ADR-0006). `forge init` produces exactly one repo
(ADR-0008), so pairing needs a topology decision.

## Decision

**1. Everything is an ordinary catalog Stack row; there is no separate frontend type.**
Two new project types join `cli`/`api`:

- `fullstack` — one repo holding a backend and a frontend. The three native frontends
  (`go-web-templ`, `python-web-jinja`, `csharp-blazor`) are single-tree `fullstack` stacks
  in their backend's language. The three api stacks gain a `FullstackBackend` flag; picking
  one under `fullstack` triggers exactly one extra question: `--frontend`.
- `frontend` — a standalone repo for the `typescript` language stacks (`vite-ts`,
  `sveltekit`, `angular`), whose generated API client points at `--api-base-url`
  (prompted; a third-party API or any existing backend).

**2. The same typescript golden tree serves both modes.** For fullstack, the writer
composes the chosen frontend's vanilla + overlay a second time under `web/` (the
*frontend fragment*), with one carve-out: the fragment's **root gate files**
(`mise.toml`, `lefthook.yml`, `.github/`) are skipped. The backend overlay's `mise.toml`
is a template whose `{{if .Frontend}}` half adds the node toolchain and `web-fmt`/
`web-lint`/`web-test` tasks that shell into `npm --prefix web run <script>`. `lefthook.yml`
and `ci.yml` need no conditional — they only call mise tasks (ADR-0003 pays off).

**3. The frontend wiring contract is the backend's `/health`.** Each frontend overlay
ships a typed API client whose scaffold-time default base URL derives from the backend's
port (chi 8080, uvicorn 8000, Kestrel 5000), overridable at runtime via
`VITE_API_BASE_URL` (Angular: the environment file) — plus a stubbed-`fetch` Vitest test
so the walking skeleton is green offline. `csharp-webapi` gains a `/health` controller so
all three backends expose the same contract. Native stacks wire the same contract
server-side (HTMX polling `/health/fragment`; Blazor injecting the same reporter the
endpoint uses), with HTMX vendored locally — no CDN.

## Considered Options

- **Pairing-matrix registry (backend × frontend combo stacks).** Rejected: 9+ combo keys
  duplicate snapshots and explode `sources.yaml`; the fragment model keeps one tree per key.
- **Per-combo overlays.** Rejected for the same duplication; the only per-combo variance
  (gate tasks, API base URL) fits in two templates.
- **Monorepo workspace managers (Nx, Turborepo).** Rejected: forge's gate pipeline is mise;
  a second orchestrator adds a competing task graph and upstream churn without adding value
  at this scale.
- **Frontend-only repos with no pairing.** Rejected: "connect to the backend I selected in
  forge" is the core ask; two disconnected repos push integration back onto the user.

## Consequences

- `Writer.Write` composes at most two trees; `copyTree` gains a destination prefix and a
  skip predicate. The `.gitignore` merge becomes multi-stem (base + language + Node).
- `forge update` interprets one recipe per key regardless of mode; JS ecosystems need the
  `strip_paths` normalize rule and npm placeholder mappings before refresh works (§15 —
  captured manually until then).
- The hermetic smoke (SPEC §16.1) stubs npm; the real JS toolchain proof lives in a
  network-gated tier (`FORGE_SMOKE_NETWORK=1`), mirroring the gh-gated tier (§16.2).
