import { resolveConfigPanelTabError } from './configPanelTabError';

describe('components/config/configPanelTabError', () => {
  it('returns the prompts error for both prompts tabs', () => {
    expect(resolveConfigPanelTabError({
      activeTab: 'prompts_library',
      generalError: 'general',
      providerError: 'provider',
      promptError: 'prompt',
      taskError: 'task',
      skillError: 'skill',
      toolError: 'tool',
    })).toBe('prompt');

    expect(resolveConfigPanelTabError({
      activeTab: 'prompts_preview',
      generalError: 'general',
      providerError: 'provider',
      promptError: 'prompt',
      taskError: 'task',
      skillError: 'skill',
      toolError: 'tool',
    })).toBe('prompt');
  });
});
