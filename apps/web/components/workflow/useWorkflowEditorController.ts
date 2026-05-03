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
  enabledWorkflowAgentToolNames,
  importWorkflowFromSessionTasks,
  normalizeWorkflowDraftAgentNodes,
  validateWorkflowDraft,
  withWorkflowContent,
  type WorkflowAgentRuntimeCatalog,
} from '@/lib/workflow-editor';
import type {
  AutosaveState,
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowCanvasPosition,
  WorkflowNodeType,
  WorkflowUpdatePayload,
} from '@/lib/workflow-editor';
import { useWorkflowAgentRuntimeCatalog } from './useWorkflowAgentRuntimeCatalog';
import type {
  UseWorkflowEditorControllerOptions,
  UseWorkflowEditorControllerResult,
  WorkflowEditorPhase,
} from './workflowEditorControllerTypes';
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
  const [agentNormalizationEnabled, setAgentNormalizationEnabled] = useState(false);
  const runtimeRef = useRef<WorkflowEditorControllerRuntime>({
    taskID,
    routeReplaced: mode === 'edit',
    baselineSeeded: false,
  });
  const agentRuntimeState = useWorkflowAgentRuntimeCatalog();
  const agentRuntimeReady = agentRuntimeState.catalog !== undefined;
  const enabledToolNames = useMemo(
    () => enabledWorkflowAgentToolNames(agentRuntimeState.catalog?.tools ?? []),
    [agentRuntimeState.catalog?.tools],
  );
  const editableDraft = useMemo(() => {
    if (!agentNormalizationEnabled || !agentRuntimeReady) {
      return draft;
    }
    return normalizeWorkflowDraftAgentNodes(draft, enabledToolNames);
  }, [agentNormalizationEnabled, agentRuntimeReady, draft, enabledToolNames]);
  const validation = useMemo(() => validateWorkflowDraft(editableDraft, {
    agentRuntimeCatalog: agentRuntimeState.catalog,
  }), [agentRuntimeState.catalog, editableDraft]);
  const snapshotBuild = useMemo(() => buildAutosaveSnapshot(editableDraft), [editableDraft]);
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

  useEffect(() => {
    if (!agentNormalizationEnabled || !agentRuntimeReady) {
      return;
    }
    const selectedNode = draft.nodes.find((node) => node.id === draft.selectedNodeId);
    if (selectedNode?.type !== 'agent' || selectedNode.agent?.runtime_overrides) {
      return;
    }
    setDraft((state) => normalizeWorkflowDraftAgentNodes(state, enabledToolNames));
  }, [agentNormalizationEnabled, agentRuntimeReady, draft.nodes, draft.selectedNodeId, enabledToolNames]);

  const handleImport = useCallback(async () => {
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
      setActionError(toErrorMessage(error, copy.system.failedToImportSessionTasks));
    } finally {
      setImportLoading(false);
    }
  }, [agentRuntimeReady, copy.system.failedToImportSessionTasks, enabledToolNames, importSessionID]);

  const handleSave = useCallback(async () => {
    setActionError('');
    setAgentNormalizationEnabled(true);
    const saveDraft = agentRuntimeReady
      ? normalizeWorkflowDraftAgentNodes(draft, enabledToolNames)
      : draft;
    const saveSnapshot = buildAutosaveSnapshot(saveDraft);
    const blockReason = resolveSaveBlockReason(validation.errors, saveSnapshot.errorMessage);
    if (blockReason || !saveSnapshot.snapshot) {
      const reason = blockReason ?? VALIDATION_BLOCKED_TEXT;
      autosaveController.markBlocked(reason);
      setActionError(reason);
      return;
    }
    try {
      await autosaveController.flush(saveSnapshot.snapshot, { force: true });
    } catch (error) {
      setActionError(toErrorMessage(error, copy.system.failedToSaveWorkflow));
    }
  }, [agentRuntimeReady, autosaveController, copy.system.failedToSaveWorkflow, draft, enabledToolNames, validation.errors]);

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
    agentRuntimeCatalog: agentRuntimeState.catalog,
    agentRuntimeLoading: agentRuntimeState.loading,
    agentRuntimeError: agentRuntimeState.error,
    validationErrors: validation.errors,
    onChangeImportSessionID: setImportSessionID,
    onImportFromSession: handleImport,
    onScheduleChange: (patch) => {
      setAgentNormalizationEnabled(true);
      setDraft((state) => agentRuntimeReady
        ? normalizeWorkflowDraftAgentNodes({ ...state, schedule: { ...state.schedule, ...patch } }, enabledToolNames)
        : { ...state, schedule: { ...state.schedule, ...patch } });
    },
    onAddNode: (type, position) => {
      setAgentNormalizationEnabled(true);
      setDraft((state) => addNode(state, type, {
        position,
        source: type === 'agent' && agentRuntimeReady
          ? {
            agent: {
              message: '',
              runtime_overrides: {
                tool_allowlist_only: true,
                tool_allowlist: [...enabledToolNames],
              },
            },
          }
          : undefined,
      }));
    },
    onSelectNode: (nodeID) => {
      const selectedNode = draft.nodes.find((node) => node.id === nodeID);
      if (selectedNode?.type === 'agent') {
        setAgentNormalizationEnabled(true);
      }
      setDraft((state) => {
        const next = { ...state, selectedNodeId: nodeID };
        if (!agentRuntimeReady || selectedNode?.type !== 'agent') {
          return next;
        }
        return normalizeWorkflowDraftAgentNodes(next, enabledToolNames);
      });
    },
    onMoveNode: (nodeID, position) => {
      setAgentNormalizationEnabled(true);
      setDraft((state) => agentRuntimeReady
        ? normalizeWorkflowDraftAgentNodes(moveNode(state, nodeID, position), enabledToolNames)
        : moveNode(state, nodeID, position));
    },
    onConnectNodes: (sourceNodeID, targetNodeID) => {
      setAgentNormalizationEnabled(true);
      setDraft((state) => agentRuntimeReady
        ? normalizeWorkflowDraftAgentNodes(connectNodesByID(state, { sourceNodeID, targetNodeID }), enabledToolNames)
        : connectNodesByID(state, { sourceNodeID, targetNodeID }));
    },
    onDeleteEdge: (edgeID) => {
      setAgentNormalizationEnabled(true);
      setDraft((state) => agentRuntimeReady
        ? normalizeWorkflowDraftAgentNodes(removeEdge(state, edgeID), enabledToolNames)
        : removeEdge(state, edgeID));
    },
    onDuplicateNode: (nodeID) => {
      setAgentNormalizationEnabled(true);
      setDraft((state) => agentRuntimeReady
        ? normalizeWorkflowDraftAgentNodes(duplicateNode(state, nodeID), enabledToolNames)
        : duplicateNode(state, nodeID));
    },
    onUpdateNode: (node) => {
      setAgentNormalizationEnabled(true);
      setDraft((state) => agentRuntimeReady
        ? normalizeWorkflowDraftAgentNodes(updateNode(state, node), enabledToolNames)
        : updateNode(state, node));
    },
    onDeleteNode: (nodeID) => {
      setAgentNormalizationEnabled(true);
      setDraft((state) => agentRuntimeReady
        ? normalizeWorkflowDraftAgentNodes(removeNode(state, nodeID), enabledToolNames)
        : removeNode(state, nodeID));
    },
    onSave: handleSave,
    onBack: handleBack,
  };
}
