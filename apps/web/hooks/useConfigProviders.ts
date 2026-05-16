'use client';

import { useCallback, useEffect, useState } from 'react';
import {
  createProvider,
  deleteProvider,
  getProviders,
  updateProvider,
} from '@/lib/api/config/api';
import {
  EditorMode,
  ProviderEditorState,
  editorStateFromProvider,
  emptyProviderEditorState,
  nextEditorStateForProviderType,
  providerInputFromEditor,
  stringsEqualIgnoreCase,
} from '@/lib/configProviders';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ConfigUpdate, ProviderConfig, ProviderListResponse } from '@/lib/types';

interface UseConfigProvidersOptions {
  open: boolean;
  onReloadConfig: () => Promise<void>;
  onActivateRuntimeConfig: (update: ConfigUpdate) => Promise<boolean>;
  modelSelectionEnabled: boolean;
}

interface UseConfigProvidersResult {
  providers: ProviderConfig[];
  activeProvider: string;
  providersLoading: boolean;
  providerSaving: boolean;
  providerError: string;
  editorMode: EditorMode;
  editor: ProviderEditorState;
  refreshProviders: () => Promise<void>;
  beginCreateProvider: () => void;
  editProvider: (provider: ProviderConfig) => void;
  updateEditor: (patch: Partial<ProviderEditorState>) => void;
  selectProviderType: (providerType: ProviderConfig['type']) => void;
  submitProvider: () => Promise<boolean>;
  activateProvider: (name: string) => Promise<void>;
  deleteProviderByName: (name: string) => Promise<void>;
  cancelEditing: () => void;
}

export function useConfigProviders(options: UseConfigProvidersOptions): UseConfigProvidersResult {
  const { copy } = useWebLocale();
  const { open, onReloadConfig, onActivateRuntimeConfig, modelSelectionEnabled } = options;
  const [providers, setProviders] = useState<ProviderConfig[]>([]);
  const [activeProvider, setActiveProviderName] = useState('');
  const [providersLoading, setProvidersLoading] = useState(false);
  const [providerSaving, setProviderSaving] = useState(false);
  const [providerError, setProviderError] = useState('');
  const [editorMode, setEditorMode] = useState<EditorMode>('create');
  const [editingName, setEditingName] = useState('');
  const [editor, setEditor] = useState<ProviderEditorState>(emptyProviderEditorState);

  const applyProviderList = useCallback((payload: ProviderListResponse) => {
    setProviders(payload.providers);
    setActiveProviderName(payload.active_provider);
  }, []);

  const resetEditor = useCallback((mode: EditorMode = 'create') => {
    setEditorMode(mode);
    setEditingName('');
    setEditor(emptyProviderEditorState);
  }, []);

  const refreshProviders = useCallback(async () => {
    setProvidersLoading(true);
    try {
      applyProviderList(await getProviders());
      setProviderError('');
    } catch (error) {
      setProviderError(toErrorMessage(error, copy.system.failedToLoadProviders));
    } finally {
      setProvidersLoading(false);
    }
  }, [applyProviderList, copy.system.failedToLoadProviders]);

  useEffect(() => {
    if (open) {
      ignorePromise(refreshProviders());
    }
  }, [open, refreshProviders]);

  const runProviderMutation = useCallback(async (
    action: () => Promise<ProviderListResponse>,
    errorMessage: string,
    onSuccess?: () => void | Promise<void>,
  ): Promise<boolean> => {
    setProviderSaving(true);
    setProviderError('');
    try {
      applyProviderList(await action());
      await onReloadConfig();
      await onSuccess?.();
      return true;
    } catch (error) {
      setProviderError(toErrorMessage(error, errorMessage));
      return false;
    } finally {
      setProviderSaving(false);
    }
  }, [applyProviderList, onReloadConfig]);

  const updateEditor = useCallback((patch: Partial<ProviderEditorState>) => {
    setEditor((state) => ({ ...state, ...patch }));
  }, []);

  const selectProviderType = useCallback((providerType: ProviderConfig['type']) => {
    setEditor((state) => nextEditorStateForProviderType(state, providerType));
  }, []);

  const submitProvider = useCallback(async (): Promise<boolean> => {
    const action = editorMode === 'edit'
      ? () => updateProvider(editingName, providerInputFromEditor(editor))
      : () => createProvider(providerInputFromEditor(editor));

    return runProviderMutation(action, copy.system.failedToSaveProvider, () => {
      resetEditor();
    });
  }, [copy.system.failedToSaveProvider, editor, editorMode, editingName, resetEditor, runProviderMutation]);

  const activateProvider = useCallback(async (name: string) => {
    setProviderSaving(true);
    setProviderError('');

    try {
      const nextProvider = providers.find((provider) => stringsEqualIgnoreCase(provider.name, name));
      const firstModel = nextProvider ? firstAvailableProviderModel(nextProvider) : null;
      const update: ConfigUpdate = { provider: name };
      if (modelSelectionEnabled && firstModel) {
        update.model = firstModel;
      }

      const saved = await onActivateRuntimeConfig(update);
      if (!saved) {
        return;
      }

      await onReloadConfig();
      applyProviderList(await getProviders());
    } catch (error) {
      setProviderError(toErrorMessage(error, copy.system.failedToSwitchProvider));
    } finally {
      setProviderSaving(false);
    }
  }, [
    applyProviderList,
    copy.system.failedToSwitchProvider,
    modelSelectionEnabled,
    onActivateRuntimeConfig,
    onReloadConfig,
    providers,
  ]);

  const deleteProviderByName = useCallback(async (name: string) => {
    await runProviderMutation(() => deleteProvider(name), copy.system.failedToDeleteProvider, () => {
      if (stringsEqualIgnoreCase(editingName, name)) {
        resetEditor();
      }
    });
  }, [copy.system.failedToDeleteProvider, editingName, resetEditor, runProviderMutation]);

  const editProvider = useCallback((provider: ProviderConfig) => {
    setEditorMode('edit');
    setEditingName(provider.name);
    setEditor(editorStateFromProvider(provider));
    setProviderError('');
  }, []);

  return {
    providers,
    activeProvider,
    providersLoading,
    providerSaving,
    providerError,
    editorMode,
    editor,
    refreshProviders,
    beginCreateProvider: resetEditor,
    editProvider,
    updateEditor,
    selectProviderType,
    submitProvider,
    activateProvider,
    deleteProviderByName,
    cancelEditing: resetEditor,
  };
}

function firstAvailableProviderModel(provider: ProviderConfig): string | null {
  for (const model of provider.models ?? []) {
    const trimmed = model.trim();
    if (trimmed.length > 0) {
      return trimmed;
    }
  }

  return null;
}
