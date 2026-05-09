'use client';

import { useCallback, useEffect, useState } from 'react';
import {
  createOrchestration,
  deleteOrchestration,
  listOrchestrations,
  runOrchestrationNow,
  updateOrchestration,
} from '@/lib/api/orchestrations/api';
import { createEmptyOrchestrationDraft, draftToOrchestrationCreatePayload } from '@/lib/orchestration-editor/draft';
import { migrateLegacyOrchestrations } from '@/lib/orchestration-editor/legacyMigration';
import { toErrorMessage } from '@/lib/errors';
import type { WebCopy } from '@/lib/i18n/messages';
import type { OrchestrationTaskPayload } from '@/lib/types';

export interface OrchestrationSectionState {
  orchestrations: OrchestrationTaskPayload[];
  loading: boolean;
  error: string;
  creating: boolean;
  name: string;
  submitting: boolean;
  runningOrchestrationID: string;
  controlsDisabled: boolean;
  refresh: () => Promise<void>;
  startCreate: () => void;
  setName: (name: string) => void;
  submitCreate: () => Promise<void>;
  cancelCreate: () => void;
  runByID: (id: string) => Promise<void>;
  setEnabledByID: (id: string, enabled: boolean) => Promise<void>;
  deleteByID: (id: string) => Promise<void>;
}

export function useOrchestrationSectionState(copy: WebCopy): OrchestrationSectionState {
  const data = useOrchestrationData(copy);
  const createState = useOrchestrationCreateState(copy, data.refresh, data.setError);
  const runByID = useRunOrchestration(copy, data.refresh, data.setError);
  const setEnabledByID = useSetOrchestrationEnabled(copy, data.replaceOrchestration, data.setError);
  const deleteByID = useDeleteOrchestration(copy, data.removeOrchestration, data.setError);

  return {
    orchestrations: data.orchestrations,
    loading: data.loading,
    error: data.error,
    runningOrchestrationID: runByID.runningOrchestrationID,
    controlsDisabled: data.loading || createState.submitting,
    refresh: data.refresh,
    runByID: runByID.runByID,
    setEnabledByID,
    deleteByID,
    ...createState,
  };
}

function useOrchestrationData(copy: WebCopy) {
  const [orchestrations, setOrchestrations] = useState<OrchestrationTaskPayload[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      await migrateLegacyOrchestrations();
      setOrchestrations(await listOrchestrations());
      setError('');
    } catch (nextError) {
      setError(toErrorMessage(nextError, copy.system.failedToLoadOrchestration));
    } finally {
      setLoading(false);
    }
  }, [copy.system.failedToLoadOrchestration]);

  const replaceOrchestration = useCallback((nextTask: OrchestrationTaskPayload) => {
    setOrchestrations((state) => state.map((task) => task.id === nextTask.id ? nextTask : task));
  }, []);

  const removeOrchestration = useCallback((id: string) => {
    setOrchestrations((state) => state.filter((task) => task.id !== id));
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return { orchestrations, loading, error, setError, refresh, replaceOrchestration, removeOrchestration };
}

function useOrchestrationCreateState(
  copy: WebCopy,
  refresh: () => Promise<void>,
  setError: (message: string) => void,
) {
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const startCreate = useCallback(() => {
    setCreating(true);
    setError('');
  }, [setError]);

  const cancelCreate = useCallback(() => {
    setCreating(false);
    setName('');
    setError('');
  }, [setError]);

  const submitCreate = useCallback(async () => {
    if (!name.trim()) {
      setError(copy.settings.orchestrationNameRequired);
      return;
    }
    setSubmitting(true);
    setError('');
    try {
      await createOrchestration(draftToOrchestrationCreatePayload(createEmptyOrchestrationDraft('edit'), name));
      await refresh();
      setCreating(false);
      setName('');
    } catch (nextError) {
      setError(toErrorMessage(nextError, copy.system.failedToCreateOrchestration));
    } finally {
      setSubmitting(false);
    }
  }, [copy.settings.orchestrationNameRequired, copy.system.failedToCreateOrchestration, name, refresh, setError]);

  return { creating, name, setName, submitting, startCreate, cancelCreate, submitCreate };
}

function useRunOrchestration(
  copy: WebCopy,
  refresh: () => Promise<void>,
  setError: (message: string) => void,
) {
  const [runningOrchestrationID, setRunningOrchestrationID] = useState('');

  const runByID = useCallback(async (id: string) => {
    if (runningOrchestrationID) {
      return;
    }
    setRunningOrchestrationID(id);
    try {
      await runOrchestrationNow(id);
      await refresh();
    } catch (nextError) {
      setError(toErrorMessage(nextError, copy.system.failedToRunTask));
    } finally {
      setRunningOrchestrationID('');
    }
  }, [copy.system.failedToRunTask, refresh, runningOrchestrationID, setError]);

  return { runByID, runningOrchestrationID };
}

function useSetOrchestrationEnabled(
  copy: WebCopy,
  replaceOrchestration: (task: OrchestrationTaskPayload) => void,
  setError: (message: string) => void,
) {
  return useCallback(async (id: string, enabled: boolean) => {
    try {
      const task = await updateOrchestration(id, { enabled });
      replaceOrchestration(task);
      setError('');
    } catch (nextError) {
      setError(toErrorMessage(nextError, copy.system.failedToUpdateTask));
    }
  }, [copy.system.failedToUpdateTask, replaceOrchestration, setError]);
}

function useDeleteOrchestration(
  copy: WebCopy,
  removeOrchestration: (id: string) => void,
  setError: (message: string) => void,
) {
  return useCallback(async (id: string) => {
    try {
      await deleteOrchestration(id);
      removeOrchestration(id);
      setError('');
    } catch (nextError) {
      setError(toErrorMessage(nextError, copy.system.failedToDeleteTask));
    }
  }, [copy.system.failedToDeleteTask, removeOrchestration, setError]);
}
