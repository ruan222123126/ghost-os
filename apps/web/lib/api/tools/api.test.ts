import { listTools, updateTool } from './api';
import { fetchMock, installFetchMock, mockFetchJSON } from '@/lib/api.test.helpers';

describe('lib/api/tools/api', () => {
  beforeEach(() => {
    installFetchMock();
  });

  it('listTools calls GET /api/tools', async () => {
    const expected = [
      { name: 'script_exec', enabled: true, prompt_override: 'custom prompt' },
      { name: 'web_search', enabled: false },
    ];

    mockFetchJSON({
      status: 'success',
      payload: expected,
      error: '',
    });

    await expect(listTools()).resolves.toEqual([
      { name: 'script_exec', enabled: true, prompt_override: 'custom prompt' },
      { name: 'web_search', enabled: false, prompt_override: undefined },
    ]);
    expect(fetchMock).toHaveBeenCalledWith('/api/tools', expect.any(Object));
  });

  it('updateTool calls PATCH /api/tools/:name', async () => {
    mockFetchJSON({
      status: 'success',
      payload: { name: 'script_exec', enabled: false, prompt_override: 'new prompt' },
      error: '',
    });

    await updateTool('script_exec', { enabled: false, prompt_override: 'new prompt' });

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/tools/script_exec',
      expect.objectContaining({
        method: 'PATCH',
        body: JSON.stringify({ enabled: false, prompt_override: 'new prompt' }),
      }),
    );
  });
});
