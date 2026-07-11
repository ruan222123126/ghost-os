import { getProviders } from '@/lib/api/config/api';
import { listTools } from '@/lib/api/tools/api';
import type { ProviderConfig, ToolPayload } from '@/lib/types';
import { loadWorkflowAgentRuntimeCatalog } from './useWorkflowAgentRuntimeCatalog';

jest.mock('@/lib/api/config/api', () => ({
  getProviders: jest.fn(),
}));

jest.mock('@/lib/api/tools/api', () => ({
  listTools: jest.fn(),
}));

const mockedGetProviders = getProviders as jest.MockedFunction<typeof getProviders>;
const mockedListTools = listTools as jest.MockedFunction<typeof listTools>;

describe('hooks/workflow/useWorkflowAgentRuntimeCatalog', () => {
  beforeEach(() => {
    mockedGetProviders.mockReset();
    mockedListTools.mockReset();
  });

  it('loads providers and tools into a workflow runtime catalog', async () => {
    const provider = buildProvider();
    const tool = buildTool();
    mockedGetProviders.mockResolvedValue({
      providers: [provider],
      active_provider: provider.name,
    });
    mockedListTools.mockResolvedValue([tool]);

    await expect(loadWorkflowAgentRuntimeCatalog()).resolves.toEqual({
      providers: [provider],
      activeProvider: provider.name,
      tools: [tool],
    });
  });
});

function buildProvider(): ProviderConfig {
  return {
    name: 'openai-main',
    type: 'openai',
    base_url: 'https://api.openai.com/v1',
    provider_id: 'provider-1',
    updated_at: '2026-01-01T00:00:00Z',
    api_key_set: true,
  };
}

function buildTool(): ToolPayload {
  return {
    name: 'screen_control',
    enabled: true,
  };
}
