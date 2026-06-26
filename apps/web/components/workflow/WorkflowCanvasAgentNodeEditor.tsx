'use client';

import {
  AgentEditorForm,
  type AgentEditorContext,
} from '@/components/workflow/WorkflowCanvasAgentNodeEditorFields';
import {
  buildToolOptions,
  toolOrderForEditor,
} from '@/components/workflow/workflowAgentNodeEditorState';
import type { WorkflowCopy } from '@/lib/i18n/messages/workflow';
import { useWebLocale } from '@/lib/i18n/provider';
import type { PresetPayload, TaskRuntimeOverrides, ToolPayload } from '@/lib/types';
import {
  cloneOrchestrationTaskRuntimeOverrides,
  cloneWorkflowTaskRuntimeOverrides,
  type WorkflowAgentRuntimeCatalog,
  type WorkflowCanvasNodeDraft,
  type WorkflowEditorKind,
} from '@/lib/workflow-editor';

interface WorkflowCanvasAgentNodeEditorProps {
  editorKind: WorkflowEditorKind;
  selectedNode: WorkflowCanvasNodeDraft;
  agentRuntimeCatalog?: WorkflowAgentRuntimeCatalog;
  agentRuntimeLoading: boolean;
  agentRuntimeError: string;
  presets?: PresetPayload[];
  presetLoading?: boolean;
  presetError?: string;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

export function WorkflowCanvasAgentNodeEditor(props: WorkflowCanvasAgentNodeEditorProps) {
  const { copy } = useWebLocale();
  const context = resolveAgentEditorContext(props, copy.workflow);

  return <AgentEditorForm context={context} />;
}

function resolveAgentEditorContext(
  props: WorkflowCanvasAgentNodeEditorProps,
  copy: WorkflowCopy,
): AgentEditorContext {
  const overrides = cloneAgentRuntimeOverrides(props.editorKind, props.selectedNode);
  const presets = props.presets ?? [];
  const tools = toolsForCatalog(props.agentRuntimeCatalog);
  const showAllTools = props.editorKind === 'orchestration';
  const toolOrder = toolOrderForEditor(tools, showAllTools);
  const selectedToolNames = toolAllowlist(overrides);

  return {
    agentRuntimeCatalog: props.agentRuntimeCatalog,
    agentRuntimeError: props.agentRuntimeError,
    agentRuntimeLoading: props.agentRuntimeLoading,
    copy,
    editorKind: props.editorKind,
    onUpdateNode: props.onUpdateNode,
    overrides,
    presetError: props.presetError ?? '',
    presetLoading: props.presetLoading === true,
    presets,
    selectedNode: props.selectedNode,
    selectedToolNames,
    toolOptions: buildToolOptions({
      tools,
      selectedNames: selectedToolNames,
      copy,
      showAllTools,
    }),
    toolOrder,
  };
}

function cloneAgentRuntimeOverrides(
  editorKind: WorkflowEditorKind,
  selectedNode: WorkflowCanvasNodeDraft,
): TaskRuntimeOverrides {
  const current = selectedNode.agent?.runtime_overrides;

  if (editorKind === 'orchestration') {
    return cloneOrchestrationTaskRuntimeOverrides(current) ?? {};
  }

  return cloneWorkflowTaskRuntimeOverrides(current) ?? {};
}

function toolsForCatalog(catalog: WorkflowAgentRuntimeCatalog | undefined): ToolPayload[] {
  return catalog?.tools ?? [];
}

function toolAllowlist(overrides: TaskRuntimeOverrides): string[] {
  return overrides.tool_allowlist ?? [];
}
