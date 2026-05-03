'use client';

import {
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
  useCallback,
  useEffect,
  useRef,
} from 'react';
import { listTasks } from '@/lib/api/tasks/api';
import { toErrorMessage } from '@/lib/errors';
import { getOrchestrationByID, saveOrchestrationDraft } from '@/lib/orchestrationStore';
import {
  createAutosaveController,
  importWorkflowFromSessionTasks,
  normalizeWorkflowDraftAgentNodes,
  type AutosaveController,
  type AutosaveState,
  type WorkflowCanvasDraft,
  type WorkflowUpdatePayload,
  withWorkflowContent,
} from '@/lib/workflow-editor';
import type { WebLocale } from '@/lib/i18n/locale';
import {
  buildAutosaveSnapshot,
  resolveSaveBlockReason,
  VALIDATION_BLOCKED_TEXT,
} from '@/components/workflow/workflowEditorDraft';

const AUTOSAVE_DEBOUNCE_MS = 800;
const AUTOSAVE_RETRY_MS = 2000;

type EditorPhase = 'loading' | 'ready' | 'missing';

export function useOrchestrationBootstrap(options: {
  orchestrationID: string;
  loadErrorMessage: string;
  draftRef: MutableRefObject<WorkflowCanvasDraft>;
  setActionError: Dispatch<SetStateAction<string>>;
  setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>;
  setPhase: Dispatch<SetStateAction<EditorPhase>>;
}) {
  const { orchestrationID, loadErrorMessage, draftRef, setActionError, setDraft, setPhase } = options;
  useEffect(() => {
    try {
      const orchestration = getOrchestrationByID(orchestrationID);
      if (!orchestration) {
        setActionError(loadErrorMessage);
        setPhase('missing');
        return;
      }
      draftRef.current = orchestration.draft;
      setDraft(orchestration.draft);
      setActionError('');
      setPhase('ready');
    } catch (error) {
      setActionError(toErrorMessage(error, loadErrorMessage));
      setPhase('missing');
    }
  }, [draftRef, loadErrorMessage, orchestrationID, setActionError, setDraft, setPhase]);
}

export function useOrchestrationAutosaveSchedule(options: {
  autosaveController: AutosaveController<WorkflowUpdatePayload>;
  currentSnapshot?: { fingerprint: string; payload: WorkflowUpdatePayload };
  locale: WebLocale;
  localizeValidationError: (message: string, locale: WebLocale) => string;
  phase: EditorPhase;
  snapshotErrorMessage?: string;
  validationErrors: string[];
}) {
  const { autosaveController, currentSnapshot, locale, localizeValidationError, phase, snapshotErrorMessage, validationErrors } = options;
  useEffect(() => {
    if (phase !== 'ready') {
      return;
    }
    const blockReason = resolveSaveBlockReason(validationErrors, snapshotErrorMessage);
    if (blockReason) {
      autosaveController.markBlocked(localizeValidationError(blockReason, locale));
      return;
    }
    if (!currentSnapshot) {
      autosaveController.markBlocked(VALIDATION_BLOCKED_TEXT);
      return;
    }
    autosaveController.schedule(currentSnapshot);
  }, [autosaveController, currentSnapshot, locale, localizeValidationError, phase, snapshotErrorMessage, validationErrors]);
}

export function useOrchestrationImportAction(options: {
  agentRuntimeReady: boolean;
  enabledToolNames: string[];
  importSessionID: string;
  importErrorMessage: string;
  setActionError: Dispatch<SetStateAction<string>>;
  setAgentNormalizationEnabled: Dispatch<SetStateAction<boolean>>;
  setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>;
  setImportLoading: Dispatch<SetStateAction<boolean>>;
}) {
  const {
    agentRuntimeReady,
    enabledToolNames,
    importSessionID,
    importErrorMessage,
    setActionError,
    setAgentNormalizationEnabled,
    setDraft,
    setImportLoading,
  } = options;
  return useCallback(async () => {
    setImportLoading(true);
    setActionError('');
    try {
      const tasks = await listTasks();
      const imported = importWorkflowFromSessionTasks(tasks, importSessionID);
      setAgentNormalizationEnabled(true);
      setDraft((state) => agentRuntimeReady
        ? normalizeWorkflowDraftAgentNodes(withWorkflowContent(state, imported.nodes, imported.edges), enabledToolNames)
        : withWorkflowContent(state, imported.nodes, imported.edges));
    } catch (error) {
      setActionError(toErrorMessage(error, importErrorMessage));
    } finally {
      setImportLoading(false);
    }
  }, [agentRuntimeReady, enabledToolNames, importErrorMessage, importSessionID, setActionError, setAgentNormalizationEnabled, setDraft, setImportLoading]);
}

export function useOrchestrationSaveAction(options: {
  agentRuntimeReady: boolean;
  autosaveController: AutosaveController<WorkflowUpdatePayload>;
  currentDraft: WorkflowCanvasDraft;
  enabledToolNames: string[];
  locale: WebLocale;
  localizeValidationError: (message: string, locale: WebLocale) => string;
  saveErrorMessage: string;
  setActionError: Dispatch<SetStateAction<string>>;
  setAgentNormalizationEnabled: Dispatch<SetStateAction<boolean>>;
  validationErrors: string[];
}) {
  const {
    agentRuntimeReady,
    autosaveController,
    currentDraft,
    enabledToolNames,
    locale,
    localizeValidationError,
    saveErrorMessage,
    setActionError,
    setAgentNormalizationEnabled,
    validationErrors,
  } = options;
  return useCallback(async () => {
    setActionError('');
    setAgentNormalizationEnabled(true);
    const saveDraft = agentRuntimeReady
      ? normalizeWorkflowDraftAgentNodes(currentDraft, enabledToolNames)
      : currentDraft;
    const saveSnapshot = buildAutosaveSnapshot(saveDraft);
    const blockReason = resolveSaveBlockReason(validationErrors, saveSnapshot.errorMessage);
    if (blockReason || !saveSnapshot.snapshot) {
      const reason = blockReason ?? VALIDATION_BLOCKED_TEXT;
      const localizedReason = localizeValidationError(reason, locale);
      autosaveController.markBlocked(localizedReason);
      setActionError(localizedReason);
      return;
    }
    try {
      await autosaveController.flush(saveSnapshot.snapshot, { force: true });
    } catch (error) {
      setActionError(toErrorMessage(error, saveErrorMessage));
    }
  }, [agentRuntimeReady, autosaveController, currentDraft, enabledToolNames, locale, localizeValidationError, saveErrorMessage, setActionError, setAgentNormalizationEnabled, validationErrors]);
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
      persist: async () => {
        saveOrchestrationDraft(orchestrationID, draftRef.current);
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

