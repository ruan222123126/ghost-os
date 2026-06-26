'use client';

import { useCallback, useState, type Dispatch, type SetStateAction } from 'react';
import { listTasks } from '@/lib/api/tasks/api';
import { toErrorMessage } from '@/lib/errors';
import {
  connectNodesByID,
  defaultWorkflowAgentRuntimeOverrides,
  importWorkflowFromSessionTasks,
  moveNode,
  normalizeWorkflowDraftAgentNodes,
  removeEdge,
  removeNode,
  updateNode,
  withWorkflowContent,
  type AutosaveController,
  type WorkflowCanvasDraft,
  type WorkflowCanvasNodeDraft,
  type WorkflowCanvasPosition,
  type WorkflowNodeType,
  type WorkflowUpdatePayload,
} from '@/lib/workflow-editor';
import type { WebCopy } from '@/lib/i18n/messages';
import { addWorkflowNodeWithGeneratedID, duplicateWorkflowNodeWithGeneratedID } from './workflowDraftMutations';
import { buildAutosaveSnapshot, resolveSaveBlockReason, VALIDATION_BLOCKED_TEXT } from './workflowEditorDraft';

interface WorkflowEditorImportActionOptions {
  copy: WebCopy;
  agentRuntimeReady: boolean;
  enabledToolNames: string[];
  setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>;
  setActionError: Dispatch<SetStateAction<string>>;
  setAgentNormalizationEnabled: Dispatch<SetStateAction<boolean>>;
}

interface WorkflowEditorDraftActionOptions {
  draft: WorkflowCanvasDraft;
  agentRuntimeReady: boolean;
  enabledToolNames: string[];
  setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>;
  setAgentNormalizationEnabled: Dispatch<SetStateAction<boolean>>;
}

interface WorkflowEditorSaveActionOptions {
  draft: WorkflowCanvasDraft;
  copy: WebCopy;
  agentRuntimeReady: boolean;
  enabledToolNames: string[];
  autosaveController: AutosaveController<WorkflowUpdatePayload>;
  validationErrors: string[];
  setActionError: Dispatch<SetStateAction<string>>;
  setAgentNormalizationEnabled: Dispatch<SetStateAction<boolean>>;
}

type AddWorkflowNodeAction = (type: WorkflowNodeType, position: WorkflowCanvasPosition) => void;
type SelectWorkflowNodeAction = (nodeID?: string) => void;

export function useWorkflowImportActions(options: WorkflowEditorImportActionOptions) {
  const {
    copy,
    agentRuntimeReady,
    enabledToolNames,
    setDraft,
    setActionError,
    setAgentNormalizationEnabled,
  } = options;
  const [importSessionID, setImportSessionID] = useState('');
  const [importLoading, setImportLoading] = useState(false);

  const handleImport = useCallback(async () => {
    setImportLoading(true);
    setActionError('');
    try {
      const tasks = await listTasks();
      const imported = importWorkflowFromSessionTasks(tasks, importSessionID);
      setAgentNormalizationEnabled(true);
      setDraft((state) => {
        const next = withWorkflowContent(state, imported.nodes, imported.edges);
        return agentRuntimeReady
          ? normalizeWorkflowDraftAgentNodes(next, enabledToolNames)
          : next;
      });
    } catch (error) {
      setActionError(toErrorMessage(error, copy.system.failedToImportSessionTasks));
    } finally {
      setImportLoading(false);
    }
  }, [agentRuntimeReady, copy.system.failedToImportSessionTasks, enabledToolNames, importSessionID, setActionError, setAgentNormalizationEnabled, setDraft]);

  return {
    importSessionID,
    importLoading,
    onChangeImportSessionID: setImportSessionID,
    onImportFromSession: handleImport,
  };
}

export function useWorkflowDraftActions(options: WorkflowEditorDraftActionOptions) {
  return {
    ...buildWorkflowScheduleActions(options),
    ...buildWorkflowCreationActions(options),
    ...buildWorkflowGraphActions(options),
    ...buildWorkflowNodeEditActions(options),
  };
}

function buildWorkflowScheduleActions(options: WorkflowEditorDraftActionOptions) {
  const {
    agentRuntimeReady,
    enabledToolNames,
    setDraft,
    setAgentNormalizationEnabled,
  } = options;

  return {
    onScheduleChange: (patch: Partial<WorkflowCanvasDraft['schedule']>) => {
      setAgentNormalizationEnabled(true);
      setDraft((state) => {
        const next = { ...state, schedule: { ...state.schedule, ...patch } };
        return normalizeDraftIfReady(next, agentRuntimeReady, enabledToolNames);
      });
    },
  };
}

function buildWorkflowCreationActions(options: WorkflowEditorDraftActionOptions) {
  return {
    onAddNode: buildAddWorkflowNodeAction(options),
    onSelectNode: buildSelectWorkflowNodeAction(options),
  };
}

function buildAddWorkflowNodeAction(options: WorkflowEditorDraftActionOptions): AddWorkflowNodeAction {
  const {
    agentRuntimeReady,
    enabledToolNames,
    setDraft,
    setAgentNormalizationEnabled,
  } = options;

  return (type, position) => {
    setAgentNormalizationEnabled(true);
    setDraft((state) => addWorkflowNodeWithGeneratedID({
      state,
      type,
      position,
      source: buildNewWorkflowNodeSource(type, agentRuntimeReady, enabledToolNames),
    }));
  };
}

function buildNewWorkflowNodeSource(
  type: WorkflowNodeType,
  agentRuntimeReady: boolean,
  enabledToolNames: string[],
): Partial<WorkflowCanvasNodeDraft> | undefined {
  if (type !== 'agent' || !agentRuntimeReady) {
    return undefined;
  }
  return {
    agent: {
      message: '',
      runtime_overrides: defaultWorkflowAgentRuntimeOverrides(enabledToolNames),
    },
  };
}

function buildSelectWorkflowNodeAction(options: WorkflowEditorDraftActionOptions): SelectWorkflowNodeAction {
  const {
    draft,
    agentRuntimeReady,
    enabledToolNames,
    setDraft,
    setAgentNormalizationEnabled,
  } = options;

  return (nodeID) => {
    const selectedNode = draft.nodes.find((node) => node.id === nodeID);
    const shouldNormalize = selectedNode?.type === 'agent';
    if (shouldNormalize) {
      setAgentNormalizationEnabled(true);
    }
    setDraft((state) => {
      const next = { ...state, selectedNodeId: nodeID };
      return shouldNormalize
        ? normalizeDraftIfReady(next, agentRuntimeReady, enabledToolNames)
        : next;
    });
  };
}

function buildWorkflowGraphActions(options: WorkflowEditorDraftActionOptions) {
  const {
    agentRuntimeReady,
    enabledToolNames,
    setDraft,
    setAgentNormalizationEnabled,
  } = options;

  return {
    onMoveNode: (nodeID: string, position: WorkflowCanvasPosition) => {
      setAgentNormalizationEnabled(true);
      setDraft((state) => {
        const next = moveNode(state, nodeID, position);
        return normalizeDraftIfReady(next, agentRuntimeReady, enabledToolNames);
      });
    },
    onConnectNodes: (sourceNodeID: string, targetNodeID: string) => {
      setAgentNormalizationEnabled(true);
      setDraft((state) => {
        const next = connectNodesByID(state, { sourceNodeID, targetNodeID });
        return normalizeDraftIfReady(next, agentRuntimeReady, enabledToolNames);
      });
    },
    onDeleteEdge: (edgeID: string) => {
      setAgentNormalizationEnabled(true);
      setDraft((state) => {
        const next = removeEdge(state, edgeID);
        return normalizeDraftIfReady(next, agentRuntimeReady, enabledToolNames);
      });
    },
  };
}

function buildWorkflowNodeEditActions(options: WorkflowEditorDraftActionOptions) {
  const {
    agentRuntimeReady,
    enabledToolNames,
    setDraft,
    setAgentNormalizationEnabled,
  } = options;

  return {
    onDuplicateNode: (nodeID: string) => {
      setAgentNormalizationEnabled(true);
      setDraft((state) => {
        const next = duplicateWorkflowNodeWithGeneratedID(state, nodeID);
        return normalizeDraftIfReady(next, agentRuntimeReady, enabledToolNames);
      });
    },
    onUpdateNode: (node: WorkflowCanvasNodeDraft) => {
      setAgentNormalizationEnabled(true);
      setDraft((state) => {
        const next = updateNode(state, node);
        return normalizeDraftIfReady(next, agentRuntimeReady, enabledToolNames);
      });
    },
    onDeleteNode: (nodeID: string) => {
      setAgentNormalizationEnabled(true);
      setDraft((state) => {
        const next = removeNode(state, nodeID);
        return normalizeDraftIfReady(next, agentRuntimeReady, enabledToolNames);
      });
    },
  };
}

function normalizeDraftIfReady(
  draft: WorkflowCanvasDraft,
  agentRuntimeReady: boolean,
  enabledToolNames: string[],
): WorkflowCanvasDraft {
  return agentRuntimeReady
    ? normalizeWorkflowDraftAgentNodes(draft, enabledToolNames)
    : draft;
}

export function useWorkflowSaveAction(options: WorkflowEditorSaveActionOptions) {
  const {
    draft,
    copy,
    agentRuntimeReady,
    enabledToolNames,
    autosaveController,
    validationErrors,
    setActionError,
    setAgentNormalizationEnabled,
  } = options;

  return useCallback(async () => {
    setActionError('');
    setAgentNormalizationEnabled(true);
    const saveDraft = agentRuntimeReady
      ? normalizeWorkflowDraftAgentNodes(draft, enabledToolNames)
      : draft;
    const saveSnapshot = buildAutosaveSnapshot(saveDraft);
    const blockReason = resolveSaveBlockReason(validationErrors, saveSnapshot.errorMessage);
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
  }, [agentRuntimeReady, autosaveController, copy.system.failedToSaveWorkflow, draft, enabledToolNames, setActionError, setAgentNormalizationEnabled, validationErrors]);
}
