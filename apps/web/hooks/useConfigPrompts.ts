'use client';

import { useCallback, useEffect, useReducer } from 'react';
import { getSystemPrompts, updateSystemPrompts } from '@/lib/api/config/api';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { SystemPromptPayload, SystemPromptUpdateRequest } from '@/lib/types';

export type SystemPromptField = 'global_template' | 'core_prompt' | 'tool_prompt' | 'tool_key_spec';

interface UseConfigPromptsOptions {
  open: boolean;
}

interface UseConfigPromptsResult {
  prompts: SystemPromptPayload | null;
  promptsLoading: boolean;
  promptSaving: boolean;
  promptError: string;
  refreshPrompts: () => Promise<void>;
  savePromptField: (field: SystemPromptField, value: string) => Promise<void>;
}

export interface ConfigPromptsState {
  prompts: SystemPromptPayload | null;
  loading: boolean;
  saving: boolean;
  error: string;
}

type ConfigPromptsAction =
  | { type: 'load_start' }
  | { type: 'load_success'; prompts: SystemPromptPayload }
  | { type: 'load_error'; error: string }
  | { type: 'save_start' }
  | { type: 'save_success'; prompts: SystemPromptPayload }
  | { type: 'save_error'; error: string };

export const initialConfigPromptsState: ConfigPromptsState = {
  prompts: null,
  loading: false,
  saving: false,
  error: '',
};

export function configPromptsReducer(
  state: ConfigPromptsState,
  action: ConfigPromptsAction,
): ConfigPromptsState {
  switch (action.type) {
    case 'load_start':
      return { ...state, loading: true };
    case 'load_success':
      return {
        ...state,
        loading: false,
        error: '',
        prompts: action.prompts,
      };
    case 'load_error':
      return { ...state, loading: false, error: action.error };
    case 'save_start':
      return { ...state, saving: true, error: '' };
    case 'save_success':
      return {
        ...state,
        saving: false,
        prompts: action.prompts,
      };
    case 'save_error':
      return { ...state, saving: false, error: action.error };
    default:
      return state;
  }
}

export function useConfigPrompts(options: UseConfigPromptsOptions): UseConfigPromptsResult {
  const { copy } = useWebLocale();
  const { open } = options;
  const [state, dispatch] = useReducer(configPromptsReducer, initialConfigPromptsState);

  const refreshPrompts = useCallback(async () => {
    dispatch({ type: 'load_start' });
    try {
      const payload = await getSystemPrompts();
      dispatch({ type: 'load_success', prompts: payload });
    } catch (error) {
      dispatch({ type: 'load_error', error: toErrorMessage(error, copy.system.failedToLoadPrompts) });
    }
  }, [copy.system.failedToLoadPrompts]);

  useEffect(() => {
    if (open) {
      ignorePromise(refreshPrompts());
    }
  }, [open, refreshPrompts]);

  const savePromptField = useCallback(async (field: SystemPromptField, value: string) => {
    dispatch({ type: 'save_start' });
    try {
      const update: SystemPromptUpdateRequest = { [field]: value } as SystemPromptUpdateRequest;
      const payload = await updateSystemPrompts(update);
      dispatch({ type: 'save_success', prompts: payload });
    } catch (error) {
      dispatch({ type: 'save_error', error: toErrorMessage(error, copy.system.failedToUpdatePrompts) });
    }
  }, [copy.system.failedToUpdatePrompts]);

  return {
    prompts: state.prompts,
    promptsLoading: state.loading,
    promptSaving: state.saving,
    promptError: state.error,
    refreshPrompts,
    savePromptField,
  };
}
