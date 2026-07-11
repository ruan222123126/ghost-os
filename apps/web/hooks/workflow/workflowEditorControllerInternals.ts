'use client';

import {
  useEffect,
  useRef,
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
} from 'react';
import { useRouter } from 'next/navigation';
import { createTask, getTask, updateTask } from '@/lib/api/tasks/api';
import { toErrorMessage } from '@/lib/errors';
import {
  type AutosaveController,
  type AutosaveSnapshot,
  type AutosaveState,
  type WorkflowCanvasDraft,
  type WorkflowCreatePayload,
  type WorkflowUpdatePayload,
  createAutosaveController,
  workflowTaskToDraft,
} from '@/lib/workflow-editor';
import type { WebCopy } from '@/lib/i18n/messages';
import {
  buildAutosaveSnapshot,
  loadStoredDraft,
  mergeServerDraftWithCachedCanvas,
  migrateStoredDraft,
  resolveSaveBlockReason,
  storeDraft,
  VALIDATION_BLOCKED_TEXT,
} from './workflowEditorDraft';

interface WorkflowBootstrapOptions {
  mode: 'create' | 'edit';
  taskID?: string;
  copy: WebCopy;
  autosaveController: AutosaveController<WorkflowUpdatePayload>;
  runtimeRef: MutableRefObject<WorkflowEditorControllerRuntime>;
  setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>;
  setActionError: Dispatch<SetStateAction<string>>;
  setPhase: Dispatch<SetStateAction<'loading' | 'ready'>>;
}

interface PersistWorkflowSnapshotOptions {
  snapshot: AutosaveSnapshot<WorkflowUpdatePayload>;
  runtimeRef: MutableRefObject<WorkflowEditorControllerRuntime>;
  setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>;
  router: ReturnType<typeof useRouter>;
}

export interface WorkflowEditorControllerRuntime {
  taskID?: string;
  routeReplaced: boolean;
  baselineSeeded: boolean;
}

const AUTOSAVE_DEBOUNCE_MS = 800;
const AUTOSAVE_RETRY_MS = 2000;
const DRAFT_PERSIST_DEBOUNCE_MS = 220;

export function useAutosaveController(
  persistSnapshotRef: MutableRefObject<(snapshot: AutosaveSnapshot<WorkflowUpdatePayload>) => Promise<void>>,
  setAutosaveState: Dispatch<SetStateAction<AutosaveState>>,
): AutosaveController<WorkflowUpdatePayload> {
  const autosaveControllerRef = useRef<AutosaveController<WorkflowUpdatePayload> | null>(null);
  if (autosaveControllerRef.current === null) {
    autosaveControllerRef.current = createAutosaveController<WorkflowUpdatePayload>({
      debounceMs: AUTOSAVE_DEBOUNCE_MS,
      retryMs: AUTOSAVE_RETRY_MS,
      persist: (snapshot) => persistSnapshotRef.current(snapshot),
      onStateChange: setAutosaveState,
    });
  }
  const autosaveController = autosaveControllerRef.current;
  useEffect(() => () => {
    autosaveController.dispose();
    autosaveControllerRef.current = null;
  }, [autosaveController]);
  return autosaveController;
}

export function useWorkflowBootstrap(options: WorkflowBootstrapOptions) {
  useCreateWorkflowBootstrap(options);
  useEditWorkflowBootstrap(options);
}

function useCreateWorkflowBootstrap(options: WorkflowBootstrapOptions) {
  const { mode, setDraft, setPhase } = options;
  useEffect(() => {
    if (mode !== 'create') {
      return;
    }
    hydrateCreateDraft(setDraft);
    setPhase('ready');
  }, [mode, setDraft, setPhase]);
}

function useEditWorkflowBootstrap(options: WorkflowBootstrapOptions) {
  const { mode, taskID, copy, autosaveController, runtimeRef, setDraft, setActionError, setPhase } = options;
  const failureMessage = copy.system.failedToLoadWorkflowTask;
  useEffect(() => {
    if (mode !== 'edit') {
      return;
    }
    if (!taskID) {
      handleMissingEditTaskID({ failureMessage, setActionError, setPhase });
      return;
    }

    return startEditDraftHydration({
      taskID,
      autosaveController,
      runtimeRef,
      setDraft,
      setActionError,
      setPhase,
      failureMessage,
    });
  }, [autosaveController, failureMessage, mode, runtimeRef, setActionError, setDraft, setPhase, taskID]);
}

function handleMissingEditTaskID(options: {
  failureMessage: string;
  setActionError: Dispatch<SetStateAction<string>>;
  setPhase: Dispatch<SetStateAction<'loading' | 'ready'>>;
}) {
  const { failureMessage, setActionError, setPhase } = options;
  setActionError(failureMessage);
  setPhase('ready');
}

function startEditDraftHydration(options: {
  taskID: string;
  autosaveController: AutosaveController<WorkflowUpdatePayload>;
  runtimeRef: MutableRefObject<WorkflowEditorControllerRuntime>;
  setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>;
  setActionError: Dispatch<SetStateAction<string>>;
  setPhase: Dispatch<SetStateAction<'loading' | 'ready'>>;
  failureMessage: string;
}) {
  const { taskID, autosaveController, runtimeRef, setDraft, setActionError, setPhase, failureMessage } = options;
  let cancelled = false;
  const isCancelled = () => cancelled;
  setPhase('loading');
  setActionError('');
  void hydrateEditDraft({
    taskID,
    autosaveController,
    runtimeRef,
    setDraft,
    onError: (error) => handleEditDraftHydrationError({
      error,
      failureMessage,
      isCancelled,
      setActionError,
    }),
    onFinally: () => handleEditDraftHydrationFinally({ isCancelled, setPhase }),
  });
  return () => {
    cancelled = true;
  };
}

function handleEditDraftHydrationError(options: {
  error: unknown;
  failureMessage: string;
  isCancelled: () => boolean;
  setActionError: Dispatch<SetStateAction<string>>;
}) {
  const { error, failureMessage, isCancelled, setActionError } = options;
  if (isCancelled()) {
    return;
  }
  setActionError(toErrorMessage(error, failureMessage));
}

function handleEditDraftHydrationFinally(options: {
  isCancelled: () => boolean;
  setPhase: Dispatch<SetStateAction<'loading' | 'ready'>>;
}) {
  const { isCancelled, setPhase } = options;
  if (isCancelled()) {
    return;
  }
  setPhase('ready');
}

export function useSeedAutosaveBaseline(options: {
  currentSnapshot?: AutosaveSnapshot<WorkflowUpdatePayload>;
  autosaveController: AutosaveController<WorkflowUpdatePayload>;
  runtimeRef: MutableRefObject<WorkflowEditorControllerRuntime>;
}) {
  const { currentSnapshot, autosaveController, runtimeRef } = options;
  useEffect(() => {
    if (runtimeRef.current.baselineSeeded || !currentSnapshot) {
      return;
    }
    autosaveController.markSaved(currentSnapshot.fingerprint);
    runtimeRef.current.baselineSeeded = true;
  }, [autosaveController, currentSnapshot, runtimeRef]);
}

export function useDraftPersistence(options: {
  draft: WorkflowCanvasDraft;
  isLoading: boolean;
  runtimeRef: MutableRefObject<WorkflowEditorControllerRuntime>;
}) {
  const { draft, isLoading, runtimeRef } = options;
  useEffect(() => {
    if (isLoading) {
      return;
    }
    const timer = window.setTimeout(() => {
      storeDraft(draft, runtimeRef.current.taskID);
    }, DRAFT_PERSIST_DEBOUNCE_MS);
    return () => {
      window.clearTimeout(timer);
    };
  }, [draft, isLoading, runtimeRef]);
}

export function useAutosaveSchedule(options: {
  autosaveController: AutosaveController<WorkflowUpdatePayload>;
  currentSnapshot?: AutosaveSnapshot<WorkflowUpdatePayload>;
  isLoading: boolean;
  snapshotErrorMessage?: string;
  validationErrors: string[];
}) {
  const { autosaveController, currentSnapshot, isLoading, snapshotErrorMessage, validationErrors } = options;
  useEffect(() => {
    if (isLoading) {
      return;
    }
    const blockReason = resolveSaveBlockReason(validationErrors, snapshotErrorMessage);
    if (blockReason) {
      autosaveController.markBlocked(blockReason);
      return;
    }
    if (!currentSnapshot) {
      autosaveController.markBlocked(VALIDATION_BLOCKED_TEXT);
      return;
    }
    autosaveController.schedule(currentSnapshot);
  }, [autosaveController, currentSnapshot, isLoading, snapshotErrorMessage, validationErrors]);
}

export async function persistWorkflowSnapshot(
  options: PersistWorkflowSnapshotOptions,
) {
  const { snapshot, runtimeRef, setDraft, router } = options;
  const existingTaskID = runtimeRef.current.taskID;
  if (existingTaskID) {
    await updateTask(existingTaskID, snapshot.payload);
    return;
  }

  const createdTask = await createTask(snapshot.payload as WorkflowCreatePayload);
  if (createdTask.task_kind !== 'workflow') {
    throw new Error('created task is not a workflow');
  }

  migrateStoredDraft(undefined, createdTask.id);
  runtimeRef.current.taskID = createdTask.id;
  setDraft((state) => ({ ...state, mode: 'edit', taskId: createdTask.id }));
  if (runtimeRef.current.routeReplaced) {
    return;
  }
  runtimeRef.current.routeReplaced = true;
  router.replace(`/workflow/${encodeURIComponent(createdTask.id)}`);
}

function hydrateCreateDraft(setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>) {
  const cachedDraft = loadStoredDraft();
  if (!cachedDraft) {
    return;
  }
  setDraft((state) => mergeServerDraftWithCachedCanvas(state, cachedDraft));
}

async function hydrateEditDraft(options: {
  taskID: string;
  autosaveController: AutosaveController<WorkflowUpdatePayload>;
  runtimeRef: MutableRefObject<WorkflowEditorControllerRuntime>;
  setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>;
  onError: (error: unknown) => void;
  onFinally: () => void;
}) {
  const { taskID, autosaveController, runtimeRef, setDraft, onError, onFinally } = options;
  try {
    const task = await getTask(taskID);
    if (task.task_kind !== 'workflow') {
      throw new Error('task is not a workflow');
    }
    const nextDraft = workflowTaskToDraft(task);
    const cachedDraft = loadStoredDraft(task.id);
    const hydratedDraft = mergeServerDraftWithCachedCanvas(nextDraft, cachedDraft);
    const snapshot = buildAutosaveSnapshot(hydratedDraft);
    if (snapshot.snapshot) {
      autosaveController.markSaved(snapshot.snapshot.fingerprint);
    }
    runtimeRef.current.taskID = task.id;
    runtimeRef.current.routeReplaced = true;
    runtimeRef.current.baselineSeeded = true;
    setDraft(hydratedDraft);
  } catch (error) {
    onError(error);
  } finally {
    onFinally();
  }
}
