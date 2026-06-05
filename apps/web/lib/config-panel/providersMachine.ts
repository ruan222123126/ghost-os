import {
  type EditorMode,
  type ProviderEditorState,
  editorStateFromProvider,
  emptyProviderEditorState,
  nextEditorStateForProviderType,
} from '@/lib/configProviders';
import type { ProviderConfig, ProviderListResponse } from '@/lib/types';

export type ProvidersView = 'list' | 'editor';

export interface ProvidersState {
  view: ProvidersView;
  providers: ProviderConfig[];
  activeProvider: string;
  loading: boolean;
  saving: boolean;
  error: string;
  editorMode: EditorMode;
  editingName: string;
  editor: ProviderEditorState;
}

export type ProvidersAction =
  | { type: 'load_start' }
  | { type: 'load_success'; payload: ProviderListResponse }
  | { type: 'load_error'; error: string }
  | { type: 'enter_create' }
  | { type: 'enter_edit'; provider: ProviderConfig }
  | { type: 'exit_editor' }
  | { type: 'patch_editor'; patch: Partial<ProviderEditorState> }
  | { type: 'select_provider_type'; providerType: ProviderConfig['type'] }
  | { type: 'mutate_start' }
  | { type: 'mutate_error'; error: string }
  | { type: 'set_saving'; saving: boolean }
  | { type: 'set_error'; error: string };

export function createInitialProvidersState(): ProvidersState {
  return {
    view: 'list',
    providers: [],
    activeProvider: '',
    loading: false,
    saving: false,
    error: '',
    editorMode: 'create',
    editingName: '',
    editor: createEmptyProviderEditor(),
  };
}

export function providersReducer(
  state: ProvidersState,
  action: ProvidersAction,
): ProvidersState {
  switch (action.type) {
    case 'load_start':
      return { ...state, loading: true };
    case 'load_success':
      return applyProviderList(state, action.payload);
    case 'load_error':
      return { ...state, loading: false, error: action.error };
    case 'enter_create':
      return enterCreateProvider(state);
    case 'enter_edit':
      return enterEditProvider(state, action.provider);
    case 'exit_editor':
      return resetProviderEditor(state);
    case 'patch_editor':
      return { ...state, editor: { ...state.editor, ...action.patch } };
    case 'select_provider_type':
      return selectProviderType(state, action.providerType);
    case 'mutate_start':
      return { ...state, saving: true, error: '' };
    case 'mutate_error':
      return { ...state, saving: false, error: action.error };
    case 'set_saving':
      return { ...state, saving: action.saving };
    case 'set_error':
      return { ...state, error: action.error };
  }
}

export function firstAvailableProviderModel(provider: ProviderConfig): string | null {
  for (const model of provider.models ?? []) {
    const trimmed = model.trim();
    if (trimmed.length > 0) {
      return trimmed;
    }
  }
  return null;
}

function applyProviderList(
  state: ProvidersState,
  payload: ProviderListResponse,
): ProvidersState {
  return {
    ...state,
    providers: payload.providers,
    activeProvider: payload.active_provider,
    loading: false,
    error: '',
  };
}

function enterCreateProvider(state: ProvidersState): ProvidersState {
  return {
    ...state,
    view: 'editor',
    editorMode: 'create',
    editingName: '',
    editor: createEmptyProviderEditor(),
    error: '',
  };
}

function enterEditProvider(
  state: ProvidersState,
  provider: ProviderConfig,
): ProvidersState {
  return {
    ...state,
    view: 'editor',
    editorMode: 'edit',
    editingName: provider.name,
    editor: editorStateFromProvider(provider),
    error: '',
  };
}

function resetProviderEditor(state: ProvidersState): ProvidersState {
  return {
    ...state,
    view: 'list',
    editorMode: 'create',
    editingName: '',
    editor: createEmptyProviderEditor(),
  };
}

function selectProviderType(
  state: ProvidersState,
  providerType: ProviderConfig['type'],
): ProvidersState {
  return {
    ...state,
    editor: nextEditorStateForProviderType(state.editor, providerType),
  };
}

function createEmptyProviderEditor(): ProviderEditorState {
  return { ...emptyProviderEditorState };
}
