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
import {
  countLegacyOrchestrations,
  migrateLegacyOrchestrations,
} from '@/lib/orchestration-editor/legacyMigration';
import { toErrorMessage } from '@/lib/errors';
import type { WebCopy } from '@/lib/i18n/messages';
import type { OrchestrationTaskPayload } from '@/lib/types';

export interface OrchestrationSectionState {
  orchestrations: OrchestrationTaskPayload[];
  loading: boolean;
  error: string;
  success: string;
  creating: boolean;
  name: string;
  submitting: boolean;
  runningOrchestrationID: string;
  controlsDisabled: boolean;
  legacyMigrationCount: number;
  legacyMigrationRunning: boolean;
  refresh: () => Promise<void>;
  runLegacyMigration: () => Promise<void>;
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
  const createState = useOrchestrationCreateState(copy, data.refresh, data.showError, data.clearFeedback);
  const runByID = useRunOrchestration(copy, data.showError, data.showSuccess, data.clearFeedback);
  const setEnabledByID = useSetOrchestrationEnabled(copy, data.replaceOrchestration, data.showError, data.clearFeedback);
  const deleteByID = useDeleteOrchestration(copy, data.removeOrchestration, data.showError, data.clearFeedback);

  return {
    orchestrations: data.orchestrations,
    loading: data.loading,
    error: data.error,
    success: data.success,
    runningOrchestrationID: runByID.runningOrchestrationID,
    controlsDisabled: data.loading || createState.submitting || data.legacyMigrationRunning,
    legacyMigrationCount: data.legacyMigrationCount,
    legacyMigrationRunning: data.legacyMigrationRunning,
    refresh: data.refresh,
    runLegacyMigration: data.runLegacyMigration,
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
  const [success, setSuccess] = useState('');
  const [legacyMigrationCount, setLegacyMigrationCount] = useState(() => countLegacyOrchestrations());
  const [legacyMigrationRunning, setLegacyMigrationRunning] = useState(false);
  const clearFeedback = useCallback(() => {
    setError('');
    setSuccess('');
  }, []);
  const showError = useCallback((message: string) => {
    setSuccess('');
    setError(message);
  }, []);
  const showSuccess = useCallback((message: string) => {
    setError('');
    setSuccess(message);
  }, []);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      setOrchestrations(await listOrchestrations());
      setLegacyMigrationCount(countLegacyOrchestrations());
      setError('');
    } catch (nextError) {
      showError(toErrorMessage(nextError, copy.system.failedToLoadOrchestration));
    } finally {
      setLoading(false);
    }
  }, [copy.system.failedToLoadOrchestration, showError]);

  const replaceOrchestration = useCallback((nextTask: OrchestrationTaskPayload) => {
    setOrchestrations((state) => state.map((task) => task.id === nextTask.id ? nextTask : task));
  }, []);

  const removeOrchestration = useCallback((id: string) => {
    setOrchestrations((state) => state.filter((task) => task.id !== id));
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const runLegacyMigration = useCallback(async () => {
    setLegacyMigrationRunning(true);
    clearFeedback();
    try {
      await migrateLegacyOrchestrations();
      setLegacyMigrationCount(countLegacyOrchestrations());
      await refresh();
    } catch (nextError) {
      showError(toErrorMessage(nextError, copy.system.failedToCreateOrchestration));
    } finally {
      setLegacyMigrationRunning(false);
    }
  }, [clearFeedback, copy.system.failedToCreateOrchestration, refresh, showError]);

  return {
    orchestrations,
    loading,
    error,
    success,
    legacyMigrationCount,
    legacyMigrationRunning,
    showError,
    showSuccess,
    clearFeedback,
    refresh,
    runLegacyMigration,
    replaceOrchestration,
    removeOrchestration,
  };
}

function useOrchestrationCreateState(
  copy: WebCopy,
  refresh: () => Promise<void>,
  showError: (message: string) => void,
  clearFeedback: () => void,
) {
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const startCreate = useCallback(() => {
    setCreating(true);
    clearFeedback();
  }, [clearFeedback]);

  const cancelCreate = useCallback(() => {
    setCreating(false);
    setName('');
    clearFeedback();
  }, [clearFeedback]);

  const submitCreate = useCallback(async () => {
    if (!name.trim()) {
      showError(copy.settings.orchestrationNameRequired);
      return;
    }
    setSubmitting(true);
    clearFeedback();
    try {
      await createOrchestration(draftToOrchestrationCreatePayload(createEmptyOrchestrationDraft('edit'), name));
      await refresh();
      setCreating(false);
      setName('');
    } catch (nextError) {
      showError(toErrorMessage(nextError, copy.system.failedToCreateOrchestration));
    } finally {
      setSubmitting(false);
    }
  }, [clearFeedback, copy.settings.orchestrationNameRequired, copy.system.failedToCreateOrchestration, name, refresh, showError]);

  return { creating, name, setName, submitting, startCreate, cancelCreate, submitCreate };
}

function useRunOrchestration(
  copy: WebCopy,
  showError: (message: string) => void,
  showSuccess: (message: string) => void,
  clearFeedback: () => void,
) {
  const [runningOrchestrationID, setRunningOrchestrationID] = useState('');

  const runByID = useCallback(async (id: string) => {
    if (runningOrchestrationID) {
      return;
    }
    clearFeedback();
    setRunningOrchestrationID(id);
    try {
      await runOrchestrationNow(id);
      showSuccess(copy.settings.tasksRunStarted);
    } catch (nextError) {
      showError(toErrorMessage(nextError, copy.system.failedToRunTask));
    } finally {
      setRunningOrchestrationID('');
    }
  }, [clearFeedback, copy.settings.tasksRunStarted, copy.system.failedToRunTask, runningOrchestrationID, showError, showSuccess]);

  return { runByID, runningOrchestrationID };
}

function useSetOrchestrationEnabled(
  copy: WebCopy,
  replaceOrchestration: (task: OrchestrationTaskPayload) => void,
  showError: (message: string) => void,
  clearFeedback: () => void,
) {
  return useCallback(async (id: string, enabled: boolean) => {
    clearFeedback();
    try {
      const task = await updateOrchestration(id, { enabled });
      replaceOrchestration(task);
    } catch (nextError) {
      showError(toErrorMessage(nextError, copy.system.failedToUpdateTask));
    }
  }, [clearFeedback, copy.system.failedToUpdateTask, replaceOrchestration, showError]);
}

function useDeleteOrchestration(
  copy: WebCopy,
  removeOrchestration: (id: string) => void,
  showError: (message: string) => void,
  clearFeedback: () => void,
) {
  return useCallback(async (id: string) => {
    clearFeedback();
    try {
      await deleteOrchestration(id);
      removeOrchestration(id);
    } catch (nextError) {
      showError(toErrorMessage(nextError, copy.system.failedToDeleteTask));
    }
  }, [clearFeedback, copy.system.failedToDeleteTask, removeOrchestration, showError]);
}
