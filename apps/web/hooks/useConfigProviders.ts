'use client';

import { useCallback, useEffect, useState } from 'react';
import {
  createProvider,
  deleteProvider,
  getProviders,
  setActiveProvider,
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
import type { ProviderConfig, ProviderListResponse } from '@/lib/types';

interface UseConfigProvidersOptions {
  open: boolean;
  onReloadConfig: () => Promise<void>;
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
  submitProvider: () => Promise<void>;
  activateProvider: (name: string) => Promise<void>;
  deleteProviderByName: (name: string) => Promise<void>;
  cancelEditing: () => void;
}

export function useConfigProviders(options: UseConfigProvidersOptions): UseConfigProvidersResult {
  const { open, onReloadConfig } = options;
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
      setProviderError(toErrorMessage(error, 'failed to load providers'));
    } finally {
      setProvidersLoading(false);
    }
  }, [applyProviderList]);

  useEffect(() => {
    if (open) {
      ignorePromise(refreshProviders());
    }
  }, [open, refreshProviders]);

  const runProviderMutation = useCallback(async (
    action: () => Promise<ProviderListResponse>,
    errorMessage: string,
    onSuccess?: () => void | Promise<void>,
  ) => {
    setProviderSaving(true);
    setProviderError('');
    try {
      applyProviderList(await action());
      await onReloadConfig();
      await onSuccess?.();
    } catch (error) {
      setProviderError(toErrorMessage(error, errorMessage));
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

  const submitProvider = useCallback(async () => {
    const action = editorMode === 'edit'
      ? () => updateProvider(editingName, providerInputFromEditor(editor))
      : () => createProvider(providerInputFromEditor(editor));

    await runProviderMutation(action, 'failed to save provider', () => {
      resetEditor();
    });
  }, [editor, editorMode, editingName, resetEditor, runProviderMutation]);

  const activateProvider = useCallback(async (name: string) => {
    await runProviderMutation(() => setActiveProvider(name), 'failed to switch provider');
  }, [runProviderMutation]);

  const deleteProviderByName = useCallback(async (name: string) => {
    await runProviderMutation(() => deleteProvider(name), 'failed to delete provider', () => {
      if (stringsEqualIgnoreCase(editingName, name)) {
        resetEditor();
      }
    });
  }, [editingName, resetEditor, runProviderMutation]);

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
