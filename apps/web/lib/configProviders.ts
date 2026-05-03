import type { ProviderConfig, ProviderConfigInput } from '@/lib/types';

export type EditorMode = 'create' | 'edit';

export interface ProviderEditorState {
  name: string;
  providerType: ProviderConfig['type'];
  baseURL: string;
  apiKey: string;
  models: string;
  contextWindowTokens: string;
}

export const providerTypeOptions: Array<{
  value: ProviderConfig['type'];
  label: string;
  defaultBaseURL: string | null;
}> = [
  { value: 'openai', label: 'OpenAI', defaultBaseURL: 'https://api.openai.com/v1' },
  { value: 'codex', label: 'Codex (Responses API)', defaultBaseURL: 'https://api.openai.com/v1' },
  { value: 'anthropic', label: 'Anthropic', defaultBaseURL: 'https://api.anthropic.com' },
  { value: 'custom', label: 'OpenAI-Compatible', defaultBaseURL: null },
];

export const emptyProviderEditorState: ProviderEditorState = {
  name: '',
  providerType: 'openai',
  baseURL: '',
  apiKey: '',
  models: '',
  contextWindowTokens: '',
};

export function defaultBaseURLForProviderType(
  providerType: ProviderConfig['type'],
): string {
  return providerTypeOptions.find((option) => option.value === providerType)?.defaultBaseURL ?? '';
}

export function labelForProviderType(providerType: ProviderConfig['type']): string {
  return providerTypeOptions.find((option) => option.value === providerType)?.label ?? providerType;
}

export function stringsEqualIgnoreCase(left: string, right: string): boolean {
  return left.trim().toLowerCase() === right.trim().toLowerCase();
}

export function editorStateFromProvider(provider: ProviderConfig): ProviderEditorState {
  return {
    name: provider.name,
    providerType: provider.type,
    baseURL: provider.base_url,
    apiKey: '',
    models: provider.models?.join(', ') ?? '',
    contextWindowTokens: provider.context_window_tokens ? String(provider.context_window_tokens) : '',
  };
}

export function nextEditorStateForProviderType(
  state: ProviderEditorState,
  providerType: ProviderConfig['type'],
): ProviderEditorState {
  const currentDefault = defaultBaseURLForProviderType(state.providerType);
  const nextDefault = defaultBaseURLForProviderType(providerType);
  const shouldReplaceBaseURL = state.baseURL.trim() === '' || state.baseURL === currentDefault;

  return {
    ...state,
    providerType,
    baseURL: shouldReplaceBaseURL ? nextDefault : state.baseURL,
  };
}

export function providerInputFromEditor(editor: ProviderEditorState): ProviderConfigInput {
  const input: ProviderConfigInput = {
    name: editor.name.trim(),
    type: editor.providerType,
    models: parseProviderModels(editor.models),
  };
  if (editor.baseURL.trim()) {
    input.base_url = editor.baseURL.trim();
  }
  if (editor.apiKey.trim()) {
    input.api_key = editor.apiKey.trim();
  }
  if (editor.contextWindowTokens.trim()) {
    input.context_window_tokens = parsePositiveInteger(editor.contextWindowTokens, 'context_window_tokens');
  }

  return input;
}

function parseProviderModels(raw: string): string[] {
  return raw
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

function parsePositiveInteger(raw: string, fieldName: string): number {
  const trimmed = raw.trim();
  if (!/^[1-9]\d*$/.test(trimmed)) {
    throw new Error(`${fieldName} must be a positive integer`);
  }
  return Number(trimmed);
}
