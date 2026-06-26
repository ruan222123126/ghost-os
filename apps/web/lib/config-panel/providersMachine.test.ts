import {
  createInitialProvidersState,
  firstAvailableProviderModel,
  providersReducer,
} from '@/lib/config-panel/providersMachine';
import type { ProviderConfig } from '@/lib/types';

describe('lib/config-panel/providersMachine', () => {
  it('enters create mode and patches editor fields', () => {
    let state = createInitialProvidersState();
    state = providersReducer(state, { type: 'enter_create' });
    state = providersReducer(state, { type: 'patch_editor', patch: { name: 'local', models: 'gpt-5' } });

    expect(state.view).toBe('editor');
    expect(state.editorMode).toBe('create');
    expect(state.editingName).toBe('');
    expect(state.editor.name).toBe('local');
    expect(state.editor.models).toBe('gpt-5');
  });

  it('enters edit mode from a provider payload', () => {
    const provider = buildProvider({
      name: 'custom-one',
      type: 'custom',
      base_url: 'https://llm.example/v1',
      models: ['alpha', 'beta'],
      context_window_tokens: 128000,
    });

    const state = providersReducer(createInitialProvidersState(), {
      type: 'enter_edit',
      provider,
    });

    expect(state.view).toBe('editor');
    expect(state.editorMode).toBe('edit');
    expect(state.editingName).toBe('custom-one');
    expect(state.editor.models).toBe('alpha, beta');
    expect(state.editor.contextWindowTokens).toBe('128000');
  });

  it('selects provider type and updates default base url', () => {
    const created = providersReducer(createInitialProvidersState(), { type: 'enter_create' });
    const state = providersReducer(created, {
      type: 'select_provider_type',
      providerType: 'anthropic',
    });

    expect(state.editor.providerType).toBe('anthropic');
    expect(state.editor.baseURL).toBe('https://api.anthropic.com');
  });

  it('loads providers and exits the editor', () => {
    const provider = buildProvider({ name: 'primary' });
    const loaded = providersReducer(createInitialProvidersState(), {
      type: 'load_success',
      payload: { providers: [provider], active_provider: 'primary' },
    });
    const editing = providersReducer(loaded, { type: 'enter_edit', provider });
    const state = providersReducer(editing, { type: 'exit_editor' });

    expect(state.providers).toEqual([provider]);
    expect(state.activeProvider).toBe('primary');
    expect(state.view).toBe('list');
    expect(state.editor.name).toBe('');
  });

  it('finds the first non-empty provider model', () => {
    const provider = buildProvider({ models: ['', '  ', 'gpt-5-mini'] });

    expect(firstAvailableProviderModel(provider)).toBe('gpt-5-mini');
  });
});

function buildProvider(overrides: Partial<ProviderConfig>): ProviderConfig {
  const name = overrides.name ?? 'openai';

  return {
    name,
    type: overrides.type ?? 'openai',
    base_url: overrides.base_url ?? 'https://api.openai.com/v1',
    provider_id: overrides.provider_id ?? name,
    updated_at: overrides.updated_at ?? '2026-01-01T00:00:00Z',
    deleted_at: overrides.deleted_at,
    models: overrides.models ?? [],
    context_window_tokens: overrides.context_window_tokens,
    response_reserve_tokens: overrides.response_reserve_tokens,
    model_context_window_tokens: overrides.model_context_window_tokens,
    model_response_reserve_tokens: overrides.model_response_reserve_tokens,
    api_key_set: overrides.api_key_set ?? false,
  };
}
