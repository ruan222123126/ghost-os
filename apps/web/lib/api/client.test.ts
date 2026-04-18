import { requestJSON } from '@/lib/api/client';
import { fetchMock, installFetchMock } from '@/lib/api.test.helpers';

describe('lib/api/client', () => {
  beforeEach(() => {
    installFetchMock();
  });

  it('parses standard JSON envelope responses', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ 'content-type': 'application/json' }),
      text: async () => JSON.stringify({
        status: 'success',
        payload: { value: 42 },
        error: '',
      }),
    });

    const payload = await requestJSON<{ value: number }>('/api/test');
    expect(payload).toEqual({ value: 42 });
  });

  it('returns explicit error for non-JSON bridge response', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 404,
      headers: new Headers({ 'content-type': 'text/plain; charset=utf-8' }),
      text: async () => '404 page not found',
    });

    await expect(requestJSON('/api/test')).rejects.toThrow(
      'bridge returned non-JSON response (status 404): 404 page not found',
    );
  });
});
