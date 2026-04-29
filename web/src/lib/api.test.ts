import { describe, expect, it, vi } from 'vitest';
import { HekateClient } from './api';

describe('HekateClient', () => {
  it('healthz returns true on {ok:true}', async () => {
    const fetchMock = vi.fn(async () =>
      new Response(JSON.stringify({ ok: true }), { status: 200 })
    );
    vi.stubGlobal('fetch', fetchMock);

    const c = new HekateClient('http://test');
    expect(await c.healthz()).toBe(true);
    expect(fetchMock).toHaveBeenCalledWith('http://test/healthz');
  });

  it('healthz returns false on non-200', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => new Response('', { status: 503 }))
    );
    const c = new HekateClient('http://test');
    expect(await c.healthz()).toBe(false);
  });
});
