import { fetchMock, installFetchMock, mockFetchJSON } from '@/lib/api.test.helpers';
import { getCodexModelCatalog } from './models';

describe('lib/api/agent/models', () => {
  beforeEach(() => {
    installFetchMock();
  });

  it('loads the Codex model catalog through the bridge bus', async () => {
    mockFetchJSON({
      status: 'success',
      payload: {
        models: ['gpt-5.6-sol', 'gpt-5.5'],
        default_model: 'gpt-5.6-sol',
      },
      error: '',
    });

    await expect(getCodexModelCatalog()).resolves.toEqual({
      models: ['gpt-5.6-sol', 'gpt-5.5'],
      default_model: 'gpt-5.6-sol',
    });
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(JSON.parse(String(init.body))).toEqual({
      action: 'EXTERNAL_AGENT_MODELS_GET',
      params: {},
      trace_id: expect.stringMatching(/^codex-models-/),
    });
  });

  it('rejects a default model that is absent from the catalog', async () => {
    mockFetchJSON({
      status: 'success',
      payload: { models: ['gpt-5.5'], default_model: 'missing' },
      error: '',
    });

    await expect(getCodexModelCatalog()).rejects.toThrow('default_model is not in models');
  });
});
