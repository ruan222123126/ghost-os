'use client';

import {
  useCallback,
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
import { listTasks } from '@/lib/api/tasks/api';
import { toErrorMessage } from '@/lib/errors';
import { buildHomeSettingsURL } from '@/lib/settingsQuery';
import {
  type AutosaveSnapshot,
  createEmptyWorkflowDraft,
  importWorkflowFromSessionTasks,
  validateWorkflowDraft,
  withWorkflowContent,
} from '@/lib/workflow-editor';
import type {
  AutosaveState,
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowCanvasPosition,
  WorkflowNodeType,
  WorkflowUpdatePayload,
} from '@/lib/workflow-editor';
import type { WebCopy } from '@/lib/i18n/messages';
import {
  buildAutosaveSnapshot,
  resolveSaveBlockReason,
  VALIDATION_BLOCKED_TEXT,
} from './workflowEditorDraft';
import {
  type WorkflowEditorControllerRuntime,
  persistWorkflowSnapshot,
  useAutosaveController,
  useAutosaveSchedule,
  useDraftPersistence,
  useSeedAutosaveBaseline,
  useWorkflowBootstrap,
} from './workflowEditorControllerInternals';

type WorkflowEditorMode = 'create' | 'edit';
type WorkflowEditorPhase = 'loading' | 'ready';

interface UseWorkflowEditorControllerOptions {
  mode: WorkflowEditorMode;
  taskID?: string;
  copy: WebCopy;
}

interface UseWorkflowEditorControllerResult {
  phase: WorkflowEditorPhase;
  draft: WorkflowCanvasDraft;
  autosaveState: AutosaveState;
  actionError: string;
  importSessionID: string;
  importLoading: boolean;
  validationErrors: string[];
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

export function useWorkflowEditorController(
  options: UseWorkflowEditorControllerOptions,
): UseWorkflowEditorControllerResult {
  const { mode, taskID, copy } = options;
  const router = useRouter();
  const [draft, setDraft] = useState<WorkflowCanvasDraft>(() => createEmptyWorkflowDraft(mode));
  const [phase, setPhase] = useState<WorkflowEditorPhase>(mode === 'edit' ? 'loading' : 'ready');
  const [autosaveState, setAutosaveState] = useState<AutosaveState>(() => ({
    phase: 'idle',
    message: 'Autosave idle',
    updatedAt: Date.now(),
  }));
  const [actionError, setActionError] = useState('');
  const [importSessionID, setImportSessionID] = useState('');
  const [importLoading, setImportLoading] = useState(false);
  const runtimeRef = useRef<WorkflowEditorControllerRuntime>({
    taskID,
    routeReplaced: mode === 'edit',
    baselineSeeded: false,
  });
  const validation = useMemo(() => validateWorkflowDraft(draft), [draft]);
  const snapshotBuild = useMemo(() => buildAutosaveSnapshot(draft), [draft]);
  const currentSnapshot = snapshotBuild.snapshot;
  const persistSnapshotRef = useRef<(snapshot: AutosaveSnapshot<WorkflowUpdatePayload>) => Promise<void>>(
    async () => undefined,
  );

  persistSnapshotRef.current = async (snapshot) =>
    persistWorkflowSnapshot(snapshot, runtimeRef, setDraft, router);

  const autosaveController = useAutosaveController(persistSnapshotRef, setAutosaveState);
  const isLoading = phase === 'loading';
  useWorkflowBootstrap({
    mode,
    taskID,
    copy,
    autosaveController,
    runtimeRef,
    setDraft,
    setActionError,
    setPhase,
  });
  useSeedAutosaveBaseline({
    currentSnapshot,
    autosaveController,
    runtimeRef,
  });
  useDraftPersistence({
    draft,
    isLoading,
    runtimeRef,
  });
  useAutosaveSchedule({
    autosaveController,
    currentSnapshot,
    isLoading,
    snapshotErrorMessage: snapshotBuild.errorMessage,
    validationErrors: validation.errors,
  });

  const handleImport = useCallback(async () => {
    setImportLoading(true);
    setActionError('');
    try {
      const tasks = await listTasks();
      const imported = importWorkflowFromSessionTasks(tasks, importSessionID);
      setDraft((state) => withWorkflowContent(state, imported.nodes, imported.edges));
    } catch (error) {
      setActionError(toErrorMessage(error, copy.system.failedToImportSessionTasks));
    } finally {
      setImportLoading(false);
    }
  }, [copy.system.failedToImportSessionTasks, importSessionID]);

  const handleSave = useCallback(async () => {
    setActionError('');
    const blockReason = resolveSaveBlockReason(validation.errors, snapshotBuild.errorMessage);
    if (blockReason || !currentSnapshot) {
      const reason = blockReason ?? VALIDATION_BLOCKED_TEXT;
      autosaveController.markBlocked(reason);
      setActionError(reason);
      return;
    }
    try {
      await autosaveController.flush(currentSnapshot, { force: true });
    } catch (error) {
      setActionError(toErrorMessage(error, copy.system.failedToSaveWorkflow));
    }
  }, [autosaveController, copy.system.failedToSaveWorkflow, currentSnapshot, snapshotBuild.errorMessage, validation.errors]);

  const handleBack = useCallback(() => {
    router.push(buildHomeSettingsURL('tasks'));
  }, [router]);

  return {
    phase,
    draft,
    autosaveState,
    actionError,
    importSessionID,
    importLoading,
    validationErrors: validation.errors,
    onChangeImportSessionID: setImportSessionID,
    onImportFromSession: handleImport,
    onScheduleChange: (patch) => setDraft((state) => ({ ...state, schedule: { ...state.schedule, ...patch } })),
    onAddNode: (type, position) => setDraft((state) => addNode(state, type, { position })),
    onSelectNode: (nodeID) => setDraft((state) => ({ ...state, selectedNodeId: nodeID })),
    onMoveNode: (nodeID, position) => setDraft((state) => moveNode(state, nodeID, position)),
    onConnectNodes: (sourceNodeID, targetNodeID) => {
      setDraft((state) => connectNodesByID(state, { sourceNodeID, targetNodeID }));
    },
    onDeleteEdge: (edgeID) => setDraft((state) => removeEdge(state, edgeID)),
    onDuplicateNode: (nodeID) => setDraft((state) => duplicateNode(state, nodeID)),
    onUpdateNode: (node) => setDraft((state) => updateNode(state, node)),
    onDeleteNode: (nodeID) => setDraft((state) => removeNode(state, nodeID)),
    onSave: handleSave,
    onBack: handleBack,
  };
}
