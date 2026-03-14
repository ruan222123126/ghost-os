import { mkdtemp, writeFile } from 'fs/promises';
import { tmpdir } from 'os';
import { join } from 'path';

import { forwardBridge } from './bridgeProxy';

describe('lib/bridgeProxy', () => {
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

  it('forwards direct bridge requests without a request body', async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ status: 'success', payload: { provider: 'openai' }, error: '' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    );

    const response = await forwardBridge({ path: '/api/config', method: 'GET' });

    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8080/api/config',
      expect.objectContaining({
        method: 'GET',
        cache: 'no-store',
      })
    );
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

  it('falls back to api_token in config.toml when env token is unset', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'ghost-web-proxy-'));
    const configPath = join(dir, 'config.toml');
    process.env.GHOST_CONFIG_PATH = configPath;
    await writeFile(configPath, 'api_token = "file-secret"\n');

    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ status: 'success', payload: { ok: true }, error: '' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    );

    await forwardBridge({ path: '/api/config/providers', method: 'GET' });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    const headers = new Headers(init.headers);
    expect(headers.get('X-API-Token')).toBe('file-secret');
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
