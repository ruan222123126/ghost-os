import {
  configToolsReducer,
  initialConfigToolsState,
} from './useConfigTools';
import type { ToolPayload } from '@/lib/types';

describe('hooks/useConfigTools reducer', () => {
  const sampleTools: ToolPayload[] = [
    { name: 'script_exec', enabled: true, prompt_override: 'custom script prompt' },
    { name: 'web_search', enabled: false },
  ];

  it('tracks loading state for refresh flow', () => {
    const loading = configToolsReducer(initialConfigToolsState, { type: 'load_start' });
    expect(loading.loading).toBe(true);

    const loaded = configToolsReducer(loading, { type: 'load_success', tools: sampleTools });
    expect(loaded.loading).toBe(false);
    expect(loaded.error).toBe('');
    expect(loaded.tools).toEqual(sampleTools);
  });

  it('updates single tool in-place after patch success', () => {
    const base = { ...initialConfigToolsState, tools: sampleTools, saving: true };

    const next = configToolsReducer(base, {
      type: 'update_success',
      tool: { name: 'script_exec', enabled: false, prompt_override: 'new prompt' },
    });
    expect(next.saving).toBe(false);
    expect(next.tools).toEqual([
      { name: 'script_exec', enabled: false, prompt_override: 'new prompt' },
      { name: 'web_search', enabled: false },
    ]);
  });

  it('stores explicit update failure error state', () => {
    const saving = configToolsReducer(initialConfigToolsState, { type: 'update_start' });
    expect(saving.saving).toBe(true);

    const failed = configToolsReducer(saving, { type: 'update_error', error: 'bad request' });
    expect(failed.saving).toBe(false);
    expect(failed.error).toBe('bad request');
  });
});
