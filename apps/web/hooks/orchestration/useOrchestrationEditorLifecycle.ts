'use client';

import { useEffect, type Dispatch, type MutableRefObject, type SetStateAction } from 'react';
import type {
  AutosaveController,
  AutosaveSnapshot,
  WorkflowCanvasDraft,
  WorkflowUpdatePayload,
} from '@/lib/workflow-editor';
import {
  useOrchestrationAutosaveSchedule,
  useOrchestrationBootstrap,
  useOrchestrationDraftPersistence,
} from './orchestrationEditorControllerInternals';
import type { OrchestrationEditorPhase } from './orchestrationEditorControllerTypes';

interface OrchestrationEditorLifecycleOptions {
  orchestrationID: string;
  loadErrorMessage: string;
  autosaveController: AutosaveController<WorkflowUpdatePayload>;
  draftRef: MutableRefObject<WorkflowCanvasDraft>;
  draft: WorkflowCanvasDraft;
  phase: OrchestrationEditorPhase;
  currentSnapshot?: AutosaveSnapshot<WorkflowUpdatePayload>;
  snapshotErrorMessage?: string;
  validationErrors: string[];
  setActionError: Dispatch<SetStateAction<string>>;
  setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>;
  setPhase: Dispatch<SetStateAction<OrchestrationEditorPhase>>;
}

export function useOrchestrationEditorLifecycle(options: OrchestrationEditorLifecycleOptions): void {
  const {
    orchestrationID,
    loadErrorMessage,
    autosaveController,
    draftRef,
    draft,
    phase,
    currentSnapshot,
    snapshotErrorMessage,
    validationErrors,
    setActionError,
    setDraft,
    setPhase,
  } = options;

  useEffect(() => {
    draftRef.current = draft;
  }, [draft, draftRef]);
  useOrchestrationBootstrap({
    orchestrationID,
    loadErrorMessage,
    autosaveController,
    draftRef,
    setActionError,
    setDraft,
    setPhase,
  });
  useOrchestrationDraftPersistence({
    draft,
    isLoading: phase === 'loading',
    orchestrationID,
  });
  useOrchestrationAutosaveSchedule({
    autosaveController,
    currentSnapshot,
    phase,
    snapshotErrorMessage,
    validationErrors,
  });
}
