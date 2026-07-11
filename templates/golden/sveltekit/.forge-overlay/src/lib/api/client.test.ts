import { afterEach, describe, expect, it, vi } from 'vitest';
import { apiBaseUrl, getJson, health } from './client';

function okJson(body: unknown): Response {
	return new Response(JSON.stringify(body), {
		status: 200,
		headers: { 'Content-Type': 'application/json' }
	});
}

describe('api client', () => {
	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('parses the health payload from the API', async () => {
		const fetchMock = vi.fn().mockResolvedValue(okJson({ status: 'ok' }));
		vi.stubGlobal('fetch', fetchMock);

		const status = await health();

		expect(status).toEqual({ status: 'ok' });
		const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
		expect(url).toBe(`${apiBaseUrl()}/health`);
		expect(new Headers(init.headers).get('Accept')).toBe('application/json');
	});

	it('preserves caller-supplied headers when merging', async () => {
		const fetchMock = vi.fn().mockResolvedValue(okJson({ status: 'ok' }));
		vi.stubGlobal('fetch', fetchMock);

		await getJson('/health', { headers: new Headers({ 'X-Request-Id': 'abc-123' }) });

		const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
		const headers = new Headers(init.headers);
		expect(headers.get('X-Request-Id')).toBe('abc-123');
		expect(headers.get('Accept')).toBe('application/json');
	});

	it('throws on a non-2xx response', async () => {
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('nope', { status: 503 })));

		await expect(health()).rejects.toThrow('GET /health failed: 503');
	});
});
