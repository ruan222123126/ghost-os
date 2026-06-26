import {
  editorStateFromProvider,
  emptyProviderEditorState,
  providerInputFromEditor,
} from './configProviders';
import type { ProviderConfig } from '@/lib/types';

describe('lib/configProviders', () => {
  it('maps provider context window tokens into editor state', () => {
    const provider: ProviderConfig = {
      name: 'deepseekv4',
      type: 'custom',
      base_url: 'https://api.deepseek.com/v1',
      provider_id: 'deepseekv4',
      updated_at: '2026-01-01T00:00:00Z',
      models: ['deepseek-v4-pro'],
      context_window_tokens: 1000000,
      api_key_set: true,
    };

    expect(editorStateFromProvider(provider)).toEqual({
      name: 'deepseekv4',
      providerType: 'custom',
      baseURL: 'https://api.deepseek.com/v1',
      apiKey: '',
      models: 'deepseek-v4-pro',
      contextWindowTokens: '1000000',
    });
  });

  it('emits provider context window tokens from editor state', () => {
    expect(providerInputFromEditor({
      ...emptyProviderEditorState,
      name: 'deepseekv4',
      providerType: 'custom',
      baseURL: 'https://api.deepseek.com/v1',
      models: 'deepseek-v4-pro',
      contextWindowTokens: '1000000',
    })).toEqual({
      name: 'deepseekv4',
      type: 'custom',
      base_url: 'https://api.deepseek.com/v1',
      models: ['deepseek-v4-pro'],
      context_window_tokens: 1000000,
    });
  });

  it('rejects invalid provider context window tokens', () => {
    expect(() => {
      providerInputFromEditor({
        ...emptyProviderEditorState,
        name: 'deepseekv4',
        providerType: 'custom',
        contextWindowTokens: '0',
      });
    }).toThrow('context_window_tokens must be a positive integer');
  });
});
