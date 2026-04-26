import { forwardBridge, forwardBridgeDownload } from './index';

describe('lib/server/bridge', () => {
  const fetchMock = jest.fn();
  const originalEnv = { ...process.env };

  beforeEach(() => {
    fetchMock.mockReset();
    process.env = { ...originalEnv };
    delete process.env.GHOST_API_TOKEN;
    delete process.env.GHOST_CONFIG_PATH;
    Object.defineProperty(global, 'fetch', {
      value: fetchMock,
      writable: true,
    });
  });

  afterAll(() => {
    process.env = originalEnv;
  });

  it('passes bridge responses through without rebuilding headers or body streams', async () => {
    const upstreamResponse = new Response(
      JSON.stringify({ status: 'success', payload: { provider: 'openai' }, error: '' }),
      {
        status: 200,
        headers: {
          'Content-Type': 'application/json',
          'X-Trace-ID': 'bridge-trace-123',
          'Cache-Control': 'no-store',
        },
      }
    );
    fetchMock.mockResolvedValue(upstreamResponse);

    const response = await forwardBridge({ path: '/api/config', method: 'GET' });

    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8080/api/config',
      expect.objectContaining({
        method: 'GET',
        cache: 'no-store',
      })
    );
    expect(response).toBe(upstreamResponse);
    expect(response.headers.get('X-Trace-ID')).toBe('bridge-trace-123');
    expect(response.headers.get('Cache-Control')).toBe('no-store');
    expect(await response.json()).toEqual({
      status: 'success',
      payload: { provider: 'openai' },
      error: '',
    });
  });

  it('forwards bridge auth from GHOST_API_TOKEN when configured', async () => {
    process.env.GHOST_API_TOKEN = 'secret-token';
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ status: 'success', payload: { ok: true }, error: '' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    );

    await forwardBridge({ path: '/api/config', method: 'GET' });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    const headers = new Headers(init.headers);
    expect(headers.get('X-API-Token')).toBe('secret-token');
  });

  it('does not inject bridge auth when request and env token are both absent', async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ status: 'success', payload: { ok: true }, error: '' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    );

    await forwardBridge({ path: '/api/config/providers', method: 'GET' });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    const headers = new Headers(init.headers);
    expect(headers.has('X-API-Token')).toBe(false);
    expect(headers.has('Authorization')).toBe(false);
  });

  it('forwards incoming auth headers on GET requests', async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ status: 'success', payload: { ok: true }, error: '' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    );

    const request = new Request('http://localhost/api/config', {
      method: 'GET',
      headers: { 'X-API-Token': 'request-secret' },
    });

    await forwardBridge({ path: '/api/config', method: 'GET', request });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    const headers = new Headers(init.headers);
    expect(headers.get('X-API-Token')).toBe('request-secret');
  });

  it('parses JSON request bodies before forwarding them', async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ status: 'success', payload: { ok: true }, error: '' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    );

    const request = new Request('http://localhost/api/agent', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message: 'hello' }),
    });

    await forwardBridge({
      path: '/api/questions/answer',
      method: 'POST',
      request,
      headers: { 'X-Trace-ID': 'web-123' },
    });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    const headers = new Headers(init.headers);

    expect(init.body).toBe(JSON.stringify({ message: 'hello' }));
    expect(headers.get('Content-Type')).toBe('application/json');
    expect(headers.get('X-Trace-ID')).toBe('web-123');
  });

  it('passes download responses through without filtering bridge trace headers', async () => {
    const upstreamResponse = new Response('artifact-bytes', {
      status: 200,
      headers: {
        'Content-Type': 'application/octet-stream',
        'Content-Disposition': 'attachment; filename="trace.log"',
        ETag: 'artifact-etag',
        'X-Artifact-SHA256': 'sha256-value',
        'X-Trace-ID': 'artifact-trace-456',
      },
    });
    fetchMock.mockResolvedValue(upstreamResponse);

    const request = new Request('http://localhost/api/sessions/sess-1/artifacts/art-1', {
      method: 'GET',
    });

    const response = await forwardBridgeDownload('/api/sessions/sess-1/artifacts/art-1', request);

    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8080/api/sessions/sess-1/artifacts/art-1',
      expect.objectContaining({
        method: 'GET',
        cache: 'no-store',
      })
    );
    expect(response).toBe(upstreamResponse);
    expect(response.headers.get('X-Trace-ID')).toBe('artifact-trace-456');
    expect(response.headers.get('Content-Disposition')).toBe('attachment; filename="trace.log"');
    expect(await response.text()).toBe('artifact-bytes');
  });

  it('returns a 400 envelope for invalid JSON bodies', async () => {
    const request = new Request('http://localhost/api/agent', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: '{',
    });

    const response = await forwardBridge({
      path: '/api/agent',
      method: 'POST',
      request,
    });

    expect(fetchMock).not.toHaveBeenCalled();
    expect(response.status).toBe(400);
    expect(await response.json()).toEqual({
      status: 'error',
      payload: {},
      error: 'invalid JSON body',
    });
  });

  it('forwards bodyless POST requests without treating them as invalid JSON', async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ status: 'success', payload: { ok: true }, error: '' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    );

    const request = new Request('http://localhost/api/tasks/task-5/run', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });

    const response = await forwardBridge({
      path: '/api/tasks/task-5/run',
      method: 'POST',
      request,
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(init.body).toBeUndefined();
    expect(response.status).toBe(200);
    expect(await response.json()).toEqual({
      status: 'success',
      payload: { ok: true },
      error: '',
    });
  });

  it('returns a 502 envelope when the bridge is unavailable', async () => {
    fetchMock.mockRejectedValue(new Error('bridge offline'));

    const response = await forwardBridge({ path: '/api/config', method: 'GET' });

    expect(response.status).toBe(502);
    expect(await response.json()).toEqual({
      status: 'error',
      payload: {},
      error: 'bridge offline',
    });
  });
});
