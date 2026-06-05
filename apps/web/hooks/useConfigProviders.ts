'use client';

import { useCallback, useEffect, useMemo, useReducer, type Dispatch } from 'react';
import { useProvidersApi } from '@/hooks/config-panel/useProvidersApi';
import {
  createInitialProvidersState,
  firstAvailableProviderModel,
  providersReducer,
  type ProvidersAction,
  type ProvidersState,
} from '@/lib/config-panel/providersMachine';
import { providerInputFromEditor, stringsEqualIgnoreCase } from '@/lib/configProviders';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ConfigUpdate, ProviderConfig } from '@/lib/types';

interface UseConfigProvidersOptions {
  open: boolean;
  onReloadConfig: () => Promise<void>;
  onActivateRuntimeConfig: (update: ConfigUpdate) => Promise<boolean>;
  modelSelectionEnabled: boolean;
}

interface UseConfigProvidersActions {
  refresh: () => Promise<boolean>;
  startCreate: () => void;
  startEdit: (provider: ProviderConfig) => void;
  cancelEditing: () => void;
  updateEditor: (patch: Partial<ProvidersState['editor']>) => void;
  selectProviderType: (providerType: ProviderConfig['type']) => void;
  submit: () => Promise<boolean>;
  activate: (name: string) => Promise<void>;
  delete: (name: string) => Promise<void>;
}

interface UseConfigProvidersResult {
  state: ProvidersState;
  actions: UseConfigProvidersActions;
}

type ProvidersApi = ReturnType<typeof useProvidersApi>;

export function useConfigProviders(options: UseConfigProvidersOptions): UseConfigProvidersResult {
  const { copy } = useWebLocale();
  const api = useProvidersApi(options);
  const [state, dispatch] = useReducer(providersReducer, undefined, createInitialProvidersState);
  const refresh = useRefreshProviders(api, copy, dispatch);
  const submit = useSubmitProvider(api, copy, state, dispatch);
  const activate = useActivateProvider(api, copy, state, dispatch);
  const deleteByName = useDeleteProvider(api, copy, state, dispatch);
  const actions = useProviderActionBundle(dispatch, refresh, submit, activate, deleteByName);

  useEffect(() => {
    if (options.open) {
      ignorePromise(refresh());
    }
  }, [options.open, refresh]);

  return { state, actions };
}

function useRefreshProviders(
  api: ProvidersApi,
  copy: ReturnType<typeof useWebLocale>['copy'],
  dispatch: Dispatch<ProvidersAction>,
) {
  return useCallback(async (): Promise<boolean> => {
    dispatch({ type: 'load_start' });
    try {
      dispatch({ type: 'load_success', payload: await api.getProviders() });
      return true;
    } catch (error) {
      dispatch({
        type: 'load_error',
        error: toErrorMessage(error, copy.system.failedToLoadProviders),
      });
      return false;
    }
  }, [api, copy.system.failedToLoadProviders, dispatch]);
}

function useSubmitProvider(
  api: ProvidersApi,
  copy: ReturnType<typeof useWebLocale>['copy'],
  state: ProvidersState,
  dispatch: Dispatch<ProvidersAction>,
) {
  return useCallback(async (): Promise<boolean> => {
    dispatch({ type: 'mutate_start' });
    try {
      const input = providerInputFromEditor(state.editor);
      const payload = state.editorMode === 'edit'
        ? await api.updateProvider(state.editingName, input)
        : await api.createProvider(input);
      dispatch({ type: 'load_success', payload });
      await api.reloadConfig();
      dispatch({ type: 'set_saving', saving: false });
      dispatch({ type: 'exit_editor' });
      return true;
    } catch (error) {
      dispatchMutationError(dispatch, error, copy.system.failedToSaveProvider);
      return false;
    }
  }, [api, copy.system.failedToSaveProvider, dispatch, state.editor, state.editorMode, state.editingName]);
}

function useActivateProvider(
  api: ProvidersApi,
  copy: ReturnType<typeof useWebLocale>['copy'],
  state: ProvidersState,
  dispatch: Dispatch<ProvidersAction>,
) {
  return useCallback(async (name: string): Promise<void> => {
    dispatch({ type: 'mutate_start' });
    try {
      const saved = await api.activateRuntimeConfig(buildProviderUpdate(api, state.providers, name));
      if (!saved) {
        dispatch({ type: 'set_saving', saving: false });
        return;
      }
      await api.reloadConfig();
      dispatch({ type: 'load_success', payload: await api.getProviders() });
      dispatch({ type: 'set_saving', saving: false });
    } catch (error) {
      dispatchMutationError(dispatch, error, copy.system.failedToSwitchProvider);
    }
  }, [api, copy.system.failedToSwitchProvider, dispatch, state.providers]);
}

function useDeleteProvider(
  api: ProvidersApi,
  copy: ReturnType<typeof useWebLocale>['copy'],
  state: ProvidersState,
  dispatch: Dispatch<ProvidersAction>,
) {
  return useCallback(async (name: string): Promise<void> => {
    dispatch({ type: 'mutate_start' });
    try {
      dispatch({ type: 'load_success', payload: await api.deleteProvider(name) });
      await api.reloadConfig();
      if (stringsEqualIgnoreCase(state.editingName, name)) {
        dispatch({ type: 'exit_editor' });
      }
      dispatch({ type: 'set_saving', saving: false });
    } catch (error) {
      dispatchMutationError(dispatch, error, copy.system.failedToDeleteProvider);
    }
  }, [api, copy.system.failedToDeleteProvider, dispatch, state.editingName]);
}

function useProviderActionBundle(
  dispatch: Dispatch<ProvidersAction>,
  refresh: UseConfigProvidersActions['refresh'],
  submit: UseConfigProvidersActions['submit'],
  activate: UseConfigProvidersActions['activate'],
  deleteByName: UseConfigProvidersActions['delete'],
): UseConfigProvidersActions {
  return useMemo(() => ({
    refresh,
    submit,
    activate,
    delete: deleteByName,
    startCreate: () => dispatch({ type: 'enter_create' }),
    startEdit: (provider) => dispatch({ type: 'enter_edit', provider }),
    cancelEditing: () => dispatch({ type: 'exit_editor' }),
    updateEditor: (patch) => dispatch({ type: 'patch_editor', patch }),
    selectProviderType: (providerType) => dispatch({ type: 'select_provider_type', providerType }),
  }), [activate, deleteByName, dispatch, refresh, submit]);
}

function buildProviderUpdate(
  api: ProvidersApi,
  providers: ProviderConfig[],
  name: string,
): ConfigUpdate {
  const update: ConfigUpdate = { provider: name };
  const provider = providers.find((item) => stringsEqualIgnoreCase(item.name, name));
  const firstModel = provider ? firstAvailableProviderModel(provider) : null;
  if (api.modelSelectionEnabled && firstModel) {
    update.model = firstModel;
  }
  return update;
}

function dispatchMutationError(
  dispatch: Dispatch<ProvidersAction>,
  error: unknown,
  fallback: string,
) {
  dispatch({ type: 'mutate_error', error: toErrorMessage(error, fallback) });
}
