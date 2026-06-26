'use client';

import { useCallback, useEffect, useReducer, type Dispatch } from 'react';
import {
  activatePreset,
  createPreset,
  deletePreset,
  listPresets,
  updatePreset,
} from '@/lib/api/presets/api';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type {
  PresetCreateRequest,
  PresetPayload,
  PresetUpdateRequest,
} from '@/lib/types';

interface UseConfigPresetsOptions {
  open: boolean;
}

interface UseConfigPresetsResult {
  presets: PresetPayload[];
  presetsLoading: boolean;
  presetSaving: boolean;
  presetError: string;
  refreshPresets: () => Promise<void>;
  createPreset: (input: PresetCreateRequest) => Promise<void>;
  updatePresetByID: (id: string, input: PresetUpdateRequest) => Promise<void>;
  deletePresetByID: (id: string) => Promise<void>;
  activatePresetByID: (id: string) => Promise<void>;
}

export interface ConfigPresetsState {
  presets: PresetPayload[];
  loading: boolean;
  saving: boolean;
  error: string;
}

type ConfigPresetsAction =
  | { type: 'load_start' }
  | { type: 'load_success'; presets: PresetPayload[] }
  | { type: 'load_error'; error: string }
  | { type: 'mutate_start' }
  | { type: 'create_success'; preset: PresetPayload }
  | { type: 'update_success'; preset: PresetPayload }
  | { type: 'delete_success'; presetID: string }
  | { type: 'activate_success' }
  | { type: 'mutate_error'; error: string };

type ConfigPresetsCopy = ReturnType<typeof useWebLocale>['copy'];
type ConfigPresetsDispatch = Dispatch<ConfigPresetsAction>;

export const initialConfigPresetsState: ConfigPresetsState = {
  presets: [],
  loading: false,
  saving: false,
  error: '',
};

function replacePreset(presets: PresetPayload[], nextPreset: PresetPayload): PresetPayload[] {
  const index = presets.findIndex((item) => item.id === nextPreset.id);
  if (index < 0) {
    return [...presets, nextPreset];
  }

  const next = [...presets];
  next[index] = nextPreset;
  return next;
}

function removePreset(presets: PresetPayload[], presetID: string): PresetPayload[] {
  return presets.filter((item) => item.id !== presetID);
}

export function configPresetsReducer(
  state: ConfigPresetsState,
  action: ConfigPresetsAction,
): ConfigPresetsState {
  switch (action.type) {
    case 'load_start':
      return { ...state, loading: true };
    case 'load_success':
      return {
        ...state,
        loading: false,
        error: '',
        presets: action.presets,
      };
    case 'load_error':
      return { ...state, loading: false, error: action.error };
    case 'mutate_start':
      return { ...state, saving: true, error: '' };
    case 'create_success':
      return {
        ...state,
        saving: false,
        presets: [...state.presets, action.preset],
      };
    case 'update_success':
      return {
        ...state,
        saving: false,
        presets: replacePreset(state.presets, action.preset),
      };
    case 'delete_success':
      return {
        ...state,
        saving: false,
        presets: removePreset(state.presets, action.presetID),
      };
    case 'activate_success':
      return {
        ...state,
        saving: false,
      };
    case 'mutate_error':
      return { ...state, saving: false, error: action.error };
    default:
      return state;
  }
}

export function useConfigPresets(options: UseConfigPresetsOptions): UseConfigPresetsResult {
  const { copy } = useWebLocale();
  const { open } = options;
  const [state, dispatch] = useReducer(configPresetsReducer, initialConfigPresetsState);
  const refreshPresets = useRefreshPresets(copy, dispatch);
  const createPresetEntry = useCreatePresetEntry(copy, dispatch);
  const updatePresetEntry = useUpdatePresetEntry(copy, dispatch);
  const deletePresetEntry = useDeletePresetEntry(copy, dispatch);
  const activatePresetEntry = useActivatePresetEntry(copy, dispatch);

  useEffect(() => {
    if (open) {
      ignorePromise(refreshPresets());
    }
  }, [open, refreshPresets]);

  return {
    presets: state.presets,
    presetsLoading: state.loading,
    presetSaving: state.saving,
    presetError: state.error,
    refreshPresets,
    createPreset: createPresetEntry,
    updatePresetByID: updatePresetEntry,
    deletePresetByID: deletePresetEntry,
    activatePresetByID: activatePresetEntry,
  };
}

function useRefreshPresets(copy: ConfigPresetsCopy, dispatch: ConfigPresetsDispatch) {
  return useCallback(async () => {
    dispatch({ type: 'load_start' });
    try {
      const payload = await listPresets();
      dispatch({ type: 'load_success', presets: payload });
    } catch (error) {
      dispatch({ type: 'load_error', error: toErrorMessage(error, copy.system.failedToLoadPresets) });
    }
  }, [copy.system.failedToLoadPresets, dispatch]);
}

function useCreatePresetEntry(copy: ConfigPresetsCopy, dispatch: ConfigPresetsDispatch) {
  return useCallback(async (input: PresetCreateRequest) => {
    dispatch({ type: 'mutate_start' });
    try {
      const preset = await createPreset(input);
      dispatch({ type: 'create_success', preset });
    } catch (error) {
      dispatch({ type: 'mutate_error', error: toErrorMessage(error, copy.system.failedToCreatePreset) });
    }
  }, [copy.system.failedToCreatePreset, dispatch]);
}

function useUpdatePresetEntry(copy: ConfigPresetsCopy, dispatch: ConfigPresetsDispatch) {
  return useCallback(async (id: string, input: PresetUpdateRequest) => {
    dispatch({ type: 'mutate_start' });
    try {
      const preset = await updatePreset(id, input);
      dispatch({ type: 'update_success', preset });
    } catch (error) {
      dispatch({ type: 'mutate_error', error: toErrorMessage(error, copy.system.failedToUpdatePreset) });
    }
  }, [copy.system.failedToUpdatePreset, dispatch]);
}

function useDeletePresetEntry(copy: ConfigPresetsCopy, dispatch: ConfigPresetsDispatch) {
  return useCallback(async (id: string) => {
    dispatch({ type: 'mutate_start' });
    try {
      await deletePreset(id);
      dispatch({ type: 'delete_success', presetID: id });
    } catch (error) {
      dispatch({ type: 'mutate_error', error: toErrorMessage(error, copy.system.failedToDeletePreset) });
    }
  }, [copy.system.failedToDeletePreset, dispatch]);
}

function useActivatePresetEntry(copy: ConfigPresetsCopy, dispatch: ConfigPresetsDispatch) {
  return useCallback(async (id: string) => {
    dispatch({ type: 'mutate_start' });
    try {
      await activatePreset(id);
      dispatch({ type: 'activate_success' });
    } catch (error) {
      dispatch({ type: 'mutate_error', error: toErrorMessage(error, copy.system.failedToActivatePreset) });
    }
  }, [copy.system.failedToActivatePreset, dispatch]);
}
