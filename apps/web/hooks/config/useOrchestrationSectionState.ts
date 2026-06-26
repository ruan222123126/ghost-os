'use client';

import { useCallback, useEffect, useMemo, useReducer, type Dispatch } from 'react';
import { useOrchestrationApi } from '@/hooks/config-panel/useOrchestrationApi';
import {
  createInitialOrchestrationState,
  orchestrationReducer,
  type OrchestrationAction,
  type OrchestrationState,
} from '@/lib/config-panel/orchestrationMachine';
import { toErrorMessage, ignorePromise } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import { createEmptyOrchestrationDraft, draftToOrchestrationCreatePayload } from '@/lib/orchestration-editor/draft';

export interface OrchestrationSectionActions {
  refresh: () => Promise<boolean>;
  startCreate: () => void;
  cancelCreate: () => void;
  setName: (name: string) => void;
  submitCreate: () => Promise<void>;
  runByID: (id: string) => Promise<void>;
  setEnabledByID: (id: string, enabled: boolean) => Promise<void>;
  deleteByID: (id: string) => Promise<void>;
}

export interface OrchestrationSectionMachine {
  state: OrchestrationState;
  actions: OrchestrationSectionActions;
}

type OrchestrationApi = ReturnType<typeof useOrchestrationApi>;
type OrchestrationCopy = ReturnType<typeof useWebLocale>['copy'];
type OrchestrationDispatch = Dispatch<OrchestrationAction>;

interface OrchestrationMutationOptions {
  api: OrchestrationApi;
  copy: OrchestrationCopy;
  dispatch: OrchestrationDispatch;
}

interface CreateOrchestrationOptions extends OrchestrationMutationOptions {
  refresh: () => Promise<boolean>;
  state: OrchestrationState;
}

interface OrchestrationActionBundleOptions {
  deleteByID: OrchestrationSectionActions['deleteByID'];
  dispatch: OrchestrationDispatch;
  refresh: OrchestrationSectionActions['refresh'];
  runByID: OrchestrationSectionActions['runByID'];
  setEnabledByID: OrchestrationSectionActions['setEnabledByID'];
  submitCreate: OrchestrationSectionActions['submitCreate'];
}

export function useOrchestrationSectionState(): OrchestrationSectionMachine {
  const { copy } = useWebLocale();
  const api = useOrchestrationApi();
  const [state, dispatch] = useReducer(
    orchestrationReducer,
    undefined,
    createInitialOrchestrationState,
  );
  const refresh = useRefreshOrchestrations(api, copy, dispatch);
  const submitCreate = useSubmitCreate({ api, copy, dispatch, refresh, state });
  const runByID = useRunOrchestration({ api, copy, dispatch, refresh, state });
  const setEnabledByID = useSetOrchestrationEnabled(api, copy, dispatch);
  const deleteByID = useDeleteOrchestration(api, copy, dispatch);
  const actions = useOrchestrationActionBundle({
    deleteByID,
    dispatch,
    refresh,
    runByID,
    setEnabledByID,
    submitCreate,
  });

  useEffect(() => {
    ignorePromise(refresh());
  }, [refresh]);

  return { state, actions };
}

function useRefreshOrchestrations(
  api: OrchestrationApi,
  copy: OrchestrationCopy,
  dispatch: OrchestrationDispatch,
) {
  return useCallback(async (): Promise<boolean> => {
    dispatch({ type: 'load_start' });
    try {
      await api.migrateLegacyOrchestrations();
      dispatch({ type: 'load_success', orchestrations: await api.listOrchestrations() });
      return true;
    } catch (error) {
      dispatch({
        type: 'load_error',
        error: toErrorMessage(error, copy.system.failedToLoadOrchestration),
      });
      return false;
    }
  }, [api, copy.system.failedToLoadOrchestration, dispatch]);
}

function useSubmitCreate(options: CreateOrchestrationOptions) {
  const { api, copy, dispatch, refresh, state } = options;

  return useCallback(async (): Promise<void> => {
    if (!state.name.trim()) {
      dispatch({ type: 'create_error', error: copy.settings.orchestrationNameRequired });
      return;
    }
    dispatch({ type: 'create_start' });
    try {
      await api.createOrchestration(createOrchestrationPayload(state.name));
      dispatch({ type: 'create_success' });
      await refresh();
    } catch (error) {
      dispatchCreateError(dispatch, error, copy.system.failedToCreateOrchestration);
    }
  }, [api, copy, dispatch, refresh, state.name]);
}

function useRunOrchestration(options: CreateOrchestrationOptions) {
  const { api, copy, dispatch, refresh, state } = options;

  return useCallback(async (id: string): Promise<void> => {
    if (state.runningOrchestrationID) {
      return;
    }
    dispatch({ type: 'run_start', id });
    try {
      await api.runOrchestrationNow(id);
      dispatch({ type: 'set_success', success: copy.settings.tasksRunStarted });
      await refresh();
    } catch (error) {
      dispatch({ type: 'run_error', error: toErrorMessage(error, copy.system.failedToRunTask) });
    } finally {
      dispatch({ type: 'run_finish' });
    }
  }, [api, copy, dispatch, refresh, state.runningOrchestrationID]);
}

function useSetOrchestrationEnabled(
  api: OrchestrationApi,
  copy: OrchestrationCopy,
  dispatch: OrchestrationDispatch,
) {
  return useCallback(async (id: string, enabled: boolean): Promise<void> => {
    dispatch({ type: 'clear_feedback' });
    try {
      const orchestration = await api.updateOrchestration(id, { enabled });
      dispatch({ type: 'replace_orchestration', orchestration });
    } catch (error) {
      dispatch({ type: 'set_error', error: toErrorMessage(error, copy.system.failedToUpdateTask) });
    }
  }, [api, copy.system.failedToUpdateTask, dispatch]);
}

function useDeleteOrchestration(
  api: OrchestrationApi,
  copy: OrchestrationCopy,
  dispatch: OrchestrationDispatch,
) {
  return useCallback(async (id: string): Promise<void> => {
    dispatch({ type: 'clear_feedback' });
    try {
      await api.deleteOrchestration(id);
      dispatch({ type: 'remove_orchestration', id });
    } catch (error) {
      dispatch({ type: 'set_error', error: toErrorMessage(error, copy.system.failedToDeleteTask) });
    }
  }, [api, copy.system.failedToDeleteTask, dispatch]);
}

function useOrchestrationActionBundle(
  options: OrchestrationActionBundleOptions,
): OrchestrationSectionActions {
  const { deleteByID, dispatch, refresh, runByID, setEnabledByID, submitCreate } = options;

  return useMemo(() => ({
    refresh,
    submitCreate,
    runByID,
    setEnabledByID,
    deleteByID,
    startCreate: () => dispatch({ type: 'enter_create' }),
    cancelCreate: () => dispatch({ type: 'cancel_create' }),
    setName: (name) => dispatch({ type: 'set_name', name }),
  }), [deleteByID, dispatch, refresh, runByID, setEnabledByID, submitCreate]);
}

function createOrchestrationPayload(name: string) {
  return draftToOrchestrationCreatePayload(createEmptyOrchestrationDraft('edit'), name);
}

function dispatchCreateError(
  dispatch: Dispatch<OrchestrationAction>,
  error: unknown,
  fallback: string,
) {
  dispatch({ type: 'create_error', error: toErrorMessage(error, fallback) });
}
