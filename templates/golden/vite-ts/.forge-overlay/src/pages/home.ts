import { health } from '../api/client';

// Pages own layout and data loading; components stay presentational.
export function renderHome(root: HTMLElement): void {
  root.innerHTML = `
    <section>
      <h1>Walking skeleton</h1>
      <p>API health: <span id="api-status">checking…</span></p>
    </section>
  `;

  const status = root.querySelector<HTMLSpanElement>('#api-status')!;
  health()
    .then((payload) => {
      status.textContent = payload.status;
    })
    .catch((error: unknown) => {
      status.textContent = `unreachable (${error instanceof Error ? error.message : String(error)})`;
    });
}
