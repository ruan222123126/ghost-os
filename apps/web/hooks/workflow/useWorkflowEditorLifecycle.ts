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
  useWorkflowBootstrap(options);
  useSeedAutosaveBaseline(options);
  useDraftPersistence(options);
  useAutosaveSchedule(options);
  useWorkflowAgentNormalization(options);
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
    if (!shouldNormalizeSelectedAgent({ draft, agentNormalizationEnabled, agentRuntimeReady })) {
      return;
    }
    setDraft((state) => {
      const normalized = normalizeWorkflowDraftAgentNodes(state, enabledToolNames);
      return normalized === state ? state : normalized;
    });
  }, [agentNormalizationEnabled, agentRuntimeReady, draft.nodes, draft.selectedNodeId, enabledToolNames, setDraft]);
}

function shouldNormalizeSelectedAgent(options: {
  draft: WorkflowCanvasDraft;
  agentNormalizationEnabled: boolean;
  agentRuntimeReady: boolean;
}): boolean {
  const { draft, agentNormalizationEnabled, agentRuntimeReady } = options;
  if (!agentNormalizationEnabled || !agentRuntimeReady) {
    return false;
  }
  const selectedNode = draft.nodes.find((node) => node.id === draft.selectedNodeId);
  return selectedNode?.type === 'agent' && !selectedNode.agent?.runtime_overrides;
}
