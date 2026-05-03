'use client';

import {
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useRouter } from 'next/navigation';
import {
  addNode,
  connectNodesByID,
  duplicateNode,
  moveNode,
  removeEdge,
  removeNode,
  updateNode,
} from '@/components/workflow/workflowCanvasState';
import {
  buildAutosaveSnapshot,
  resolveSaveBlockReason,
  VALIDATION_BLOCKED_TEXT,
} from '@/components/workflow/workflowEditorDraft';
import { listTasks } from '@/lib/api/tasks/api';
import { toErrorMessage } from '@/lib/errors';
import { getOrchestrationByID, saveOrchestrationDraft } from '@/lib/orchestrationStore';
import { buildHomeSettingsURL } from '@/lib/settingsQuery';
import {
  createAutosaveController,
  createEmptyWorkflowDraft,
  importWorkflowFromSessionTasks,
  type AutosaveController,
  type AutosaveState,
  type WorkflowCanvasDraft,
  type WorkflowCanvasNodeDraft,
  type WorkflowCanvasPosition,
  type WorkflowNodeType,
  type WorkflowUpdatePayload,
  validateWorkflowDraft,
  withWorkflowContent,
} from '@/lib/workflow-editor';
import type { WebLocale } from '@/lib/i18n/locale';
import type { WebCopy } from '@/lib/i18n/messages';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import {
  buildOrchestrationWorkflowCopy,
  localizeOrchestrationValidationError,
} from './orchestrationEditorCopy';

const AUTOSAVE_DEBOUNCE_MS = 800;
const AUTOSAVE_RETRY_MS = 2000;
const ORCHESTRATION_SETTINGS_URL = buildHomeSettingsURL('orchestration');

type EditorPhase = 'loading' | 'ready' | 'missing';

interface UseOrchestrationEditorControllerOptions {
  orchestrationID: string;
  copy: WebCopy;
  locale: WebLocale;
}

interface UseOrchestrationEditorControllerResult {
  phase: EditorPhase;
  draft: WorkflowCanvasDraft;
  autosaveState: AutosaveState;
  actionError: string;
  validationErrors: string[];
  importSessionID: string;
  importLoading: boolean;
  workflowCopy: WorkflowCopy;
  localizeValidationError: (message: string, locale: WebLocale) => string;
  onChangeImportSessionID: (value: string) => void;
  onImportFromSession: () => Promise<void>;
  onScheduleChange: (patch: Partial<WorkflowCanvasDraft['schedule']>) => void;
  onAddNode: (type: WorkflowNodeType, position: WorkflowCanvasPosition) => void;
  onSelectNode: (nodeID?: string) => void;
  onMoveNode: (nodeID: string, position: WorkflowCanvasPosition) => void;
  onConnectNodes: (sourceNodeID: string, targetNodeID: string) => void;
  onDeleteEdge: (edgeID: string) => void;
  onDuplicateNode: (nodeID: string) => void;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
  onDeleteNode: (nodeID: string) => void;
  onSave: () => Promise<void>;
  onBack: () => void;
}

export function useOrchestrationEditorController(
  options: UseOrchestrationEditorControllerOptions,
): UseOrchestrationEditorControllerResult {
  const { orchestrationID, copy, locale } = options;
  const router = useRouter();
  const [draft, setDraft] = useState(() => createEmptyWorkflowDraft('edit'));
  const [phase, setPhase] = useState<EditorPhase>('loading');
  const [autosaveState, setAutosaveState] = useState<AutosaveState>(() => ({
    phase: 'idle',
    message: 'Autosave idle',
    updatedAt: Date.now(),
  }));
  const [actionError, setActionError] = useState('');
  const [importSessionID, setImportSessionID] = useState('');
  const [importLoading, setImportLoading] = useState(false);
  const draftRef = useRef(draft);
  const validation = useMemo(() => validateWorkflowDraft(draft), [draft]);
  const snapshotBuild = useMemo(() => buildAutosaveSnapshot(draft), [draft]);
  const workflowCopy = useMemo(() => buildOrchestrationWorkflowCopy(copy.workflow, locale), [copy.workflow, locale]);
  const localizeValidationError = useCallback(localizeOrchestrationValidationError, []);
  const autosaveController = useOrchestrationAutosaveController({ orchestrationID, draftRef, setAutosaveState });
  const handleImport = useOrchestrationImportAction({ importSessionID, importErrorMessage: copy.system.failedToImportSessionTasks, setActionError, setDraft, setImportLoading });
  const handleSave = useOrchestrationSaveAction({ autosaveController, currentSnapshot: snapshotBuild.snapshot, locale, localizeValidationError, saveErrorMessage: copy.system.failedToSaveOrchestration, setActionError, snapshotErrorMessage: snapshotBuild.errorMessage, validationErrors: validation.errors });
  const handleBack = useCallback(() => router.push(ORCHESTRATION_SETTINGS_URL), [router]);

  useEffect(() => {
    draftRef.current = draft;
  }, [draft]);

  useOrchestrationBootstrap({ orchestrationID, loadErrorMessage: copy.system.failedToLoadOrchestration, draftRef, setActionError, setDraft, setPhase });
  useOrchestrationAutosaveSchedule({ autosaveController, currentSnapshot: snapshotBuild.snapshot, locale, localizeValidationError, phase, snapshotErrorMessage: snapshotBuild.errorMessage, validationErrors: validation.errors });

  return {
    phase,
    draft,
    autosaveState,
    actionError,
    validationErrors: validation.errors,
    importSessionID,
    importLoading,
    workflowCopy,
    localizeValidationError,
    onChangeImportSessionID: setImportSessionID,
    onImportFromSession: handleImport,
    onScheduleChange: (patch) => setDraft((state) => ({ ...state, schedule: { ...state.schedule, ...patch } })),
    onAddNode: (type, position) => setDraft((state) => addNode(state, type, { position })),
    onSelectNode: (nodeID) => setDraft((state) => ({ ...state, selectedNodeId: nodeID })),
    onMoveNode: (nodeID, position) => setDraft((state) => moveNode(state, nodeID, position)),
    onConnectNodes: (sourceNodeID, targetNodeID) => setDraft((state) => connectNodesByID(state, { sourceNodeID, targetNodeID })),
    onDeleteEdge: (edgeID) => setDraft((state) => removeEdge(state, edgeID)),
    onDuplicateNode: (nodeID) => setDraft((state) => duplicateNode(state, nodeID)),
    onUpdateNode: (node) => setDraft((state) => updateNode(state, node)),
    onDeleteNode: (nodeID) => setDraft((state) => removeNode(state, nodeID)),
    onSave: handleSave,
    onBack: handleBack,
  };
}

function useOrchestrationBootstrap(options: {
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

function useOrchestrationAutosaveSchedule(options: {
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

function useOrchestrationImportAction(options: {
  importSessionID: string;
  importErrorMessage: string;
  setActionError: Dispatch<SetStateAction<string>>;
  setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>;
  setImportLoading: Dispatch<SetStateAction<boolean>>;
}) {
  const { importSessionID, importErrorMessage, setActionError, setDraft, setImportLoading } = options;

  return useCallback(async () => {
    setImportLoading(true);
    setActionError('');
    try {
      const tasks = await listTasks();
      const imported = importWorkflowFromSessionTasks(tasks, importSessionID);
      setDraft((state) => withWorkflowContent(state, imported.nodes, imported.edges));
    } catch (error) {
      setActionError(toErrorMessage(error, importErrorMessage));
    } finally {
      setImportLoading(false);
    }
  }, [importErrorMessage, importSessionID, setActionError, setDraft, setImportLoading]);
}

function useOrchestrationSaveAction(options: {
  autosaveController: AutosaveController<WorkflowUpdatePayload>;
  currentSnapshot?: { fingerprint: string; payload: WorkflowUpdatePayload };
  locale: WebLocale;
  localizeValidationError: (message: string, locale: WebLocale) => string;
  saveErrorMessage: string;
  setActionError: Dispatch<SetStateAction<string>>;
  snapshotErrorMessage?: string;
  validationErrors: string[];
}) {
  const { autosaveController, currentSnapshot, locale, localizeValidationError, saveErrorMessage, setActionError, snapshotErrorMessage, validationErrors } = options;

  return useCallback(async () => {
    setActionError('');
    const blockReason = resolveSaveBlockReason(validationErrors, snapshotErrorMessage);
    if (blockReason || !currentSnapshot) {
      const reason = blockReason ?? VALIDATION_BLOCKED_TEXT;
      const localizedReason = localizeValidationError(reason, locale);
      autosaveController.markBlocked(localizedReason);
      setActionError(localizedReason);
      return;
    }
    try {
      await autosaveController.flush(currentSnapshot, { force: true });
    } catch (error) {
      setActionError(toErrorMessage(error, saveErrorMessage));
    }
  }, [autosaveController, currentSnapshot, locale, localizeValidationError, saveErrorMessage, setActionError, snapshotErrorMessage, validationErrors]);
}

function useOrchestrationAutosaveController(options: {
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
