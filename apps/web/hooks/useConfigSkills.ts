'use client';

import { useCallback, useEffect, useReducer } from 'react';
import { deleteSkill, listSkills, updateSkill } from '@/lib/api/skills/api';
import { ignorePromise, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { SkillPayload } from '@/lib/types';

interface UseConfigSkillsOptions {
  open: boolean;
}

interface UseConfigSkillsResult {
  skills: SkillPayload[];
  skillsLoading: boolean;
  skillSaving: boolean;
  skillError: string;
  refreshSkills: () => Promise<void>;
  updateSkillByID: (id: string, enabled: boolean) => Promise<void>;
  deleteSkillByID: (id: string) => Promise<void>;
}

export interface ConfigSkillsState {
  skills: SkillPayload[];
  loading: boolean;
  saving: boolean;
  error: string;
}

type ConfigSkillsAction =
  | { type: 'load_start' }
  | { type: 'load_success'; skills: SkillPayload[] }
  | { type: 'load_error'; error: string }
  | { type: 'save_start' }
  | { type: 'update_success'; skill: SkillPayload }
  | { type: 'save_error'; error: string }
  | { type: 'delete_start' }
  | { type: 'delete_success'; id: string }
  | { type: 'delete_error'; error: string };

export const initialConfigSkillsState: ConfigSkillsState = {
  skills: [],
  loading: false,
  saving: false,
  error: '',
};

function removeSkillFromList(skills: SkillPayload[], id: string): SkillPayload[] {
  return skills.filter((item) => item.id !== id);
}

function replaceSkillInList(skills: SkillPayload[], updated: SkillPayload): SkillPayload[] {
  return skills.map((item) => (item.id === updated.id ? updated : item));
}

export function configSkillsReducer(state: ConfigSkillsState, action: ConfigSkillsAction): ConfigSkillsState {
  switch (action.type) {
    case 'load_start':
      return { ...state, loading: true };
    case 'load_success':
      return {
        ...state,
        loading: false,
        error: '',
        skills: action.skills,
      };
    case 'load_error':
      return { ...state, loading: false, error: action.error };
    case 'save_start':
      return { ...state, saving: true, error: '' };
    case 'update_success':
      return {
        ...state,
        saving: false,
        skills: replaceSkillInList(state.skills, action.skill),
      };
    case 'save_error':
      return { ...state, saving: false, error: action.error };
    case 'delete_start':
      return { ...state, saving: true, error: '' };
    case 'delete_success':
      return {
        ...state,
        saving: false,
        skills: removeSkillFromList(state.skills, action.id),
      };
    case 'delete_error':
      return { ...state, saving: false, error: action.error };
    default:
      return state;
  }
}

export function useConfigSkills(options: UseConfigSkillsOptions): UseConfigSkillsResult {
  const { copy } = useWebLocale();
  const { open } = options;
  const [state, dispatch] = useReducer(configSkillsReducer, initialConfigSkillsState);

  const refreshSkills = useCallback(async () => {
    dispatch({ type: 'load_start' });
    try {
      const payload = await listSkills();
      dispatch({ type: 'load_success', skills: payload });
    } catch (error) {
      dispatch({ type: 'load_error', error: toErrorMessage(error, copy.system.failedToLoadSkills) });
    }
  }, [copy.system.failedToLoadSkills]);

  useEffect(() => {
    if (open) {
      ignorePromise(refreshSkills());
    }
  }, [open, refreshSkills]);

  const updateSkillByID = useCallback(async (id: string, enabled: boolean) => {
    dispatch({ type: 'save_start' });
    try {
      const payload = await updateSkill(id, { enabled });
      dispatch({ type: 'update_success', skill: payload });
      await refreshSkills();
    } catch (error) {
      dispatch({ type: 'save_error', error: toErrorMessage(error, copy.system.failedToUpdateSkill) });
    }
  }, [copy.system.failedToUpdateSkill, refreshSkills]);

  const deleteSkillByID = useCallback(async (id: string) => {
    dispatch({ type: 'delete_start' });
    try {
      await deleteSkill(id);
      dispatch({ type: 'delete_success', id });
      await refreshSkills();
    } catch (error) {
      dispatch({ type: 'delete_error', error: toErrorMessage(error, copy.system.failedToDeleteSkill) });
    }
  }, [copy.system.failedToDeleteSkill, refreshSkills]);

  return {
    skills: state.skills,
    skillsLoading: state.loading,
    skillSaving: state.saving,
    skillError: state.error,
    refreshSkills,
    updateSkillByID,
    deleteSkillByID,
  };
}
