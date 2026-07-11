'use client';

import { useCallback, useEffect, useReducer } from 'react';
import { listTools, updateTool } from '@/lib/api/tools/api';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ToolPayload, ToolUpdateRequest } from '@/lib/types';

interface UseConfigToolsOptions {
  open: boolean;
}

interface UseConfigToolsResult {
  tools: ToolPayload[];
  toolsLoading: boolean;
  toolSaving: boolean;
  toolError: string;
  refreshTools: () => Promise<void>;
  updateToolByName: (name: string, input: ToolUpdateRequest) => Promise<void>;
}

export interface ConfigToolsState {
  tools: ToolPayload[];
  loading: boolean;
  saving: boolean;
  error: string;
}

type ConfigToolsAction =
  | { type: 'load_start' }
  | { type: 'load_success'; tools: ToolPayload[] }
  | { type: 'load_error'; error: string }
  | { type: 'update_start' }
  | { type: 'update_success'; tool: ToolPayload }
  | { type: 'update_error'; error: string };

export const initialConfigToolsState: ConfigToolsState = {
  tools: [],
  loading: false,
  saving: false,
  error: '',
};

function replaceTool(tools: ToolPayload[], nextTool: ToolPayload): ToolPayload[] {
  const index = tools.findIndex((item) => item.name === nextTool.name);
  if (index < 0) {
    return [nextTool, ...tools];
  }
  const next = [...tools];
  next[index] = nextTool;
  return next;
}

export function configToolsReducer(state: ConfigToolsState, action: ConfigToolsAction): ConfigToolsState {
  switch (action.type) {
    case 'load_start':
      return { ...state, loading: true };
    case 'load_success':
      return {
        ...state,
        loading: false,
        error: '',
        tools: action.tools,
      };
    case 'load_error':
      return { ...state, loading: false, error: action.error };
    case 'update_start':
      return { ...state, saving: true, error: '' };
    case 'update_success':
      return {
        ...state,
        saving: false,
        tools: replaceTool(state.tools, action.tool),
      };
    case 'update_error':
      return { ...state, saving: false, error: action.error };
    default:
      return state;
  }
}

export function useConfigTools(options: UseConfigToolsOptions): UseConfigToolsResult {
  const { copy } = useWebLocale();
  const { open } = options;
  const [state, dispatch] = useReducer(configToolsReducer, initialConfigToolsState);

  const refreshTools = useCallback(async () => {
    dispatch({ type: 'load_start' });
    try {
      const payload = await listTools();
      dispatch({ type: 'load_success', tools: payload });
    } catch (error) {
      dispatch({ type: 'load_error', error: toErrorMessage(error, copy.system.failedToLoadTools) });
    }
  }, [copy.system.failedToLoadTools]);

  useEffect(() => {
    if (open) {
      ignorePromise(refreshTools());
    }
  }, [open, refreshTools]);

  const updateToolByName = useCallback(async (name: string, input: ToolUpdateRequest) => {
    dispatch({ type: 'update_start' });
    try {
      const updated = await updateTool(name, input);
      dispatch({ type: 'update_success', tool: updated });
    } catch (error) {
      dispatch({ type: 'update_error', error: toErrorMessage(error, copy.system.failedToUpdateTool) });
    }
  }, [copy.system.failedToUpdateTool]);

  return {
    tools: state.tools,
    toolsLoading: state.loading,
    toolSaving: state.saving,
    toolError: state.error,
    refreshTools,
    updateToolByName,
  };
}
