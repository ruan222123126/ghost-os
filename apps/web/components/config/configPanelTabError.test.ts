import { resolveConfigPanelTabError } from './configPanelTabError';

describe('components/config/configPanelTabError', () => {
  it('returns the prompts error for both prompts tabs', () => {
    expect(resolveConfigPanelTabError({
      activeTab: 'prompts_library',
      generalError: 'general',
      providerError: 'provider',
      presetError: 'preset',
      promptError: 'prompt',
      taskError: 'task',
      skillError: 'skill',
      toolError: 'tool',
    })).toBe('prompt');

    expect(resolveConfigPanelTabError({
      activeTab: 'prompts_preview',
      generalError: 'general',
      providerError: 'provider',
      presetError: 'preset',
      promptError: 'prompt',
      taskError: 'task',
      skillError: 'skill',
      toolError: 'tool',
    })).toBe('prompt');
  });

  it('returns the preset error for presets tab', () => {
    expect(resolveConfigPanelTabError({
      activeTab: 'presets',
      generalError: 'general',
      providerError: 'provider',
      presetError: 'preset',
      promptError: 'prompt',
      taskError: 'task',
      skillError: 'skill',
      toolError: 'tool',
    })).toBe('preset');
  });

  it('returns the general config error for relay tab', () => {
    expect(resolveConfigPanelTabError({
      activeTab: 'relay',
      generalError: 'general',
      providerError: 'provider',
      presetError: 'preset',
      promptError: 'prompt',
      taskError: 'task',
      skillError: 'skill',
      toolError: 'tool',
    })).toBe('general');
  });
});
