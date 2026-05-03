'use client';

import {
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
  createEmptyWorkflowDraft,
  enabledWorkflowAgentToolNames,
  normalizeWorkflowDraftAgentNodes,
  type AutosaveState,
  type WorkflowAgentRuntimeCatalog,
  type WorkflowCanvasDraft,
  type WorkflowCanvasNodeDraft,
  type WorkflowCanvasPosition,
  type WorkflowNodeType,
  type WorkflowUpdatePayload,
  validateWorkflowDraft,
  withWorkflowContent,
} from '@/lib/workflow-editor';
import { buildHomeSettingsURL } from '@/lib/settingsQuery';
import type { WebLocale } from '@/lib/i18n/locale';
import type { WebCopy } from '@/lib/i18n/messages';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import { useWorkflowAgentRuntimeCatalog } from '@/components/workflow/useWorkflowAgentRuntimeCatalog';
import {
  useOrchestrationAutosaveController,
  useOrchestrationAutosaveSchedule,
  useOrchestrationBootstrap,
  useOrchestrationImportAction,
  useOrchestrationSaveAction,
} from './orchestrationEditorControllerInternals';
import {
  buildOrchestrationWorkflowCopy,
  localizeOrchestrationValidationError,
} from './orchestrationEditorCopy';
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
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  agentRuntimeLoading: boolean;
  agentRuntimeError: string;
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
  const [agentNormalizationEnabled, setAgentNormalizationEnabled] = useState(false);
  const draftRef = useRef(draft);
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
  const workflowCopy = useMemo(() => buildOrchestrationWorkflowCopy(copy.workflow, locale), [copy.workflow, locale]);
  const localizeValidationError = useCallback(localizeOrchestrationValidationError, []);
  const autosaveController = useOrchestrationAutosaveController({ orchestrationID, draftRef, setAutosaveState });
  const handleImport = useOrchestrationImportAction({
    agentRuntimeReady,
    enabledToolNames,
    importSessionID,
    importErrorMessage: copy.system.failedToImportSessionTasks,
    setActionError,
    setAgentNormalizationEnabled,
    setDraft,
    setImportLoading,
  });
  const handleSave = useOrchestrationSaveAction({
    agentRuntimeReady,
    autosaveController,
    currentDraft: draft,
    enabledToolNames,
    locale,
    localizeValidationError,
    saveErrorMessage: copy.system.failedToSaveOrchestration,
    setActionError,
    setAgentNormalizationEnabled,
    validationErrors: validation.errors,
  });
  const handleBack = useCallback(() => router.push(ORCHESTRATION_SETTINGS_URL), [router]);

  useEffect(() => {
    draftRef.current = draft;
  }, [draft]);

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
    agentRuntimeCatalog: agentRuntimeState.catalog,
    agentRuntimeLoading: agentRuntimeState.loading,
    agentRuntimeError: agentRuntimeState.error,
    workflowCopy,
    localizeValidationError,
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
