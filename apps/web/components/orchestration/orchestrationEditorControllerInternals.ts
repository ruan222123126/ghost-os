'use client';

import {
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
  useEffect,
  useRef,
} from 'react';
import { getOrchestration, updateOrchestration } from '@/lib/api/orchestrations/api';
import { toErrorMessage } from '@/lib/errors';
import { orchestrationTaskToDraft } from '@/lib/orchestration-editor/draft';
import {
  createAutosaveController,
  type AutosaveController,
  type AutosaveSnapshot,
  type AutosaveState,
  type WorkflowCanvasDraft,
  type WorkflowUpdatePayload,
} from '@/lib/workflow-editor';
import {
  buildOrchestrationAutosaveSnapshot,
  loadStoredOrchestrationDraft,
  mergeOrchestrationDraftWithCachedCanvas,
  ORCHESTRATION_VALIDATION_BLOCKED_TEXT,
  storeOrchestrationDraft,
} from './orchestrationEditorDraft';

const AUTOSAVE_DEBOUNCE_MS = 800;
const AUTOSAVE_RETRY_MS = 2000;
const DRAFT_PERSIST_DEBOUNCE_MS = 220;

type EditorPhase = 'loading' | 'ready' | 'missing';

export function useOrchestrationBootstrap(options: {
  orchestrationID: string;
  loadErrorMessage: string;
  autosaveController: AutosaveController<WorkflowUpdatePayload>;
  draftRef: MutableRefObject<WorkflowCanvasDraft>;
  setActionError: Dispatch<SetStateAction<string>>;
  setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>;
  setPhase: Dispatch<SetStateAction<EditorPhase>>;
}) {
  const { orchestrationID, loadErrorMessage, autosaveController, draftRef, setActionError, setDraft, setPhase } = options;
  useEffect(() => {
    let cancelled = false;
    setPhase('loading');
    setActionError('');
    void (async () => {
      try {
        const task = await getOrchestration(orchestrationID);
        const draft = mergeOrchestrationDraftWithCachedCanvas(
          orchestrationTaskToDraft(task),
          loadStoredOrchestrationDraft(task.id),
        );
        const snapshot = buildOrchestrationAutosaveSnapshot(draft);
        if (snapshot.snapshot) {
          autosaveController.markSaved(snapshot.snapshot.fingerprint);
        }
        if (!cancelled) {
          draftRef.current = draft;
          setDraft(draft);
          setPhase('ready');
        }
      } catch (error) {
        if (cancelled) {
          return;
        }
        setActionError(toErrorMessage(error, loadErrorMessage));
        setPhase('missing');
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [autosaveController, draftRef, loadErrorMessage, orchestrationID, setActionError, setDraft, setPhase]);
}

export function useOrchestrationDraftPersistence(options: {
  draft: WorkflowCanvasDraft;
  isLoading: boolean;
  orchestrationID: string;
}) {
  const { draft, isLoading, orchestrationID } = options;
  useEffect(() => {
    if (isLoading) {
      return;
    }
    const timer = window.setTimeout(() => {
      storeOrchestrationDraft(draft, orchestrationID);
    }, DRAFT_PERSIST_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [draft, isLoading, orchestrationID]);
}

export function useOrchestrationAutosaveSchedule(options: {
  autosaveController: AutosaveController<WorkflowUpdatePayload>;
  currentSnapshot?: AutosaveSnapshot<WorkflowUpdatePayload>;
  phase: EditorPhase;
  snapshotErrorMessage?: string;
  validationErrors: string[];
}) {
  const { autosaveController, currentSnapshot, phase, snapshotErrorMessage, validationErrors } = options;
  useEffect(() => {
    if (phase !== 'ready') {
      return;
    }
    const blockReason = validationErrors[0]?.trim() || snapshotErrorMessage?.trim();
    if (blockReason) {
      autosaveController.markBlocked(blockReason);
      return;
    }
    if (!currentSnapshot) {
      autosaveController.markBlocked(ORCHESTRATION_VALIDATION_BLOCKED_TEXT);
      return;
    }
    autosaveController.schedule(currentSnapshot);
  }, [autosaveController, currentSnapshot, phase, snapshotErrorMessage, validationErrors]);
}

export function useOrchestrationAutosaveController(options: {
  orchestrationID: string;
  draftRef: MutableRefObject<WorkflowCanvasDraft>;
  setAutosaveState: Dispatch<SetStateAction<AutosaveState>>;
}): AutosaveController<WorkflowUpdatePayload> {
  const { orchestrationID, draftRef, setAutosaveState } = options;
  const autosaveControllerRef = useRef<AutosaveController<WorkflowUpdatePayload> | null>(null);
  if (autosaveControllerRef.current === null) {
    autosaveControllerRef.current = createAutosaveController({
      debounceMs: AUTOSAVE_DEBOUNCE_MS,
      retryMs: AUTOSAVE_RETRY_MS,
      persist: async (snapshot) => {
        draftRef.current = { ...draftRef.current, taskId: orchestrationID, mode: 'edit' };
        await updateOrchestration(orchestrationID, snapshot.payload);
      },
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
