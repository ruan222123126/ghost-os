'use client';

import { useEffect, type Dispatch, type MutableRefObject, type SetStateAction } from 'react';
import type { WebCopy } from '@/lib/i18n/messages';
import {
  type AutosaveController,
  type AutosaveSnapshot,
  type WorkflowCanvasDraft,
  type WorkflowUpdatePayload,
  normalizeWorkflowDraftAgentNodes,
} from '@/lib/workflow-editor';
import {
  useAutosaveSchedule,
  useDraftPersistence,
  useSeedAutosaveBaseline,
  useWorkflowBootstrap,
  type WorkflowEditorControllerRuntime,
} from './workflowEditorControllerInternals';

interface WorkflowEditorLifecycleOptions {
  mode: 'create' | 'edit';
  taskID?: string;
  copy: WebCopy;
  autosaveController: AutosaveController<WorkflowUpdatePayload>;
  runtimeRef: MutableRefObject<WorkflowEditorControllerRuntime>;
  draft: WorkflowCanvasDraft;
  currentSnapshot?: AutosaveSnapshot<WorkflowUpdatePayload>;
  isLoading: boolean;
  snapshotErrorMessage?: string;
  validationErrors: string[];
  agentNormalizationEnabled: boolean;
  agentRuntimeReady: boolean;
  enabledToolNames: string[];
  setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>;
  setActionError: Dispatch<SetStateAction<string>>;
  setPhase: Dispatch<SetStateAction<'loading' | 'ready'>>;
}

export function useWorkflowEditorLifecycle(options: WorkflowEditorLifecycleOptions): void {
  const {
    mode,
    taskID,
    copy,
    autosaveController,
    runtimeRef,
    draft,
    currentSnapshot,
    isLoading,
    snapshotErrorMessage,
    validationErrors,
    agentNormalizationEnabled,
    agentRuntimeReady,
    enabledToolNames,
    setDraft,
    setActionError,
    setPhase,
  } = options;

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
    snapshotErrorMessage,
    validationErrors,
  });
  useWorkflowAgentNormalization({
    draft,
    agentNormalizationEnabled,
    agentRuntimeReady,
    enabledToolNames,
    setDraft,
  });
}

function useWorkflowAgentNormalization(options: {
  draft: WorkflowCanvasDraft;
  agentNormalizationEnabled: boolean;
  agentRuntimeReady: boolean;
  enabledToolNames: string[];
  setDraft: Dispatch<SetStateAction<WorkflowCanvasDraft>>;
}): void {
  const {
    draft,
    agentNormalizationEnabled,
    agentRuntimeReady,
    enabledToolNames,
    setDraft,
  } = options;

  useEffect(() => {
    if (!agentNormalizationEnabled || !agentRuntimeReady) {
      return;
    }
    const selectedNode = draft.nodes.find((node) => node.id === draft.selectedNodeId);
    if (selectedNode?.type !== 'agent' || selectedNode.agent?.runtime_overrides) {
      return;
    }
    setDraft((state) => {
      const normalized = normalizeWorkflowDraftAgentNodes(state, enabledToolNames);
      return normalized === state ? state : normalized;
    });
  }, [agentNormalizationEnabled, agentRuntimeReady, draft.nodes, draft.selectedNodeId, enabledToolNames, setDraft]);
}
