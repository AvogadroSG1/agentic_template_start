import { afterEach, describe, expect, it, vi } from 'vitest';
import { apiBaseUrl, health } from './client';

describe('api client', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('parses the health payload from the API', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ status: 'ok' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    );
    vi.stubGlobal('fetch', fetchMock);

    const status = await health();

    expect(status).toEqual({ status: 'ok' });
    expect(fetchMock).toHaveBeenCalledWith(
      `${apiBaseUrl()}/health`,
      expect.objectContaining({ headers: expect.objectContaining({ Accept: 'application/json' }) }),
    );
  });

  it('throws on a non-2xx response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('nope', { status: 503 })));

    await expect(health()).rejects.toThrow('GET /health failed: 503');
  });
});
