import type {
  WorkflowDefinition,
  WorkflowInputVariable,
  WorkflowNode,
  WorkflowTaskPayload,
} from '@/lib/types';
import { cloneWorkflowTaskRuntimeOverrides } from '@/lib/workflow-editor/agentRuntime';
import {
  DECIMAL_RADIX,
  DEFAULT_LOOP_MAX_ITERATIONS,
  DEFAULT_END_NODE_ID,
  DEFAULT_INTERVAL_SECONDS,
  DEFAULT_START_NODE_ID,
  MIN_INTERVAL_SECONDS,
} from '@/lib/workflow-editor/constants';
import { buildCanvasGraphFromWorkflowDefinition } from '@/lib/workflow-editor/draftExpand';
import { compileLoopPairsForTransport } from '@/lib/workflow-editor/loopCompile';
import type {
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowCreatePayload,
  WorkflowDefinitionImport,
  WorkflowUpdatePayload,
} from '@/lib/workflow-editor/types';

export function createEmptyWorkflowDraft(mode: 'create' | 'edit' = 'create'): WorkflowCanvasDraft {
  const graph = buildCanvasGraphFromWorkflowDefinition({
    nodes: [
      { id: DEFAULT_START_NODE_ID, type: 'start' },
      { id: DEFAULT_END_NODE_ID, type: 'end' },
    ],
    edges: [{ from_node_id: DEFAULT_START_NODE_ID, to_node_id: DEFAULT_END_NODE_ID }],
  });

  return {
    mode,
    schedule: {
      mode: 'interval',
      intervalSeconds: DEFAULT_INTERVAL_SECONDS,
      cronExpr: '',
    },
    nodes: graph.nodes,
    edges: graph.edges,
  };
}

export function workflowTaskToDraft(task: WorkflowTaskPayload): WorkflowCanvasDraft {
  const base = workflowDefinitionToDraft({
    workflow: task.workflow,
    scheduleType: task.schedule_type,
    intervalSeconds: task.interval_seconds,
    cronExpr: task.cron_expr,
  });

  return {
    ...base,
    mode: 'edit',
    taskId: task.id,
  };
}

export function workflowDefinitionToDraft(input: WorkflowDefinitionImport): WorkflowCanvasDraft {
  const graph = buildCanvasGraphFromWorkflowDefinition(input.workflow);
  return {
    mode: 'create',
    schedule: scheduleDraftFromTask(input.scheduleType, input.intervalSeconds, input.cronExpr),
    nodes: graph.nodes,
    edges: graph.edges,
  };
}

export function draftToWorkflowDefinition(draft: WorkflowCanvasDraft): WorkflowDefinition {
  const transport = compileLoopPairsForTransport(draft.nodes, draft.edges);
  return {
    nodes: transport.nodes.map((node) => buildWorkflowNode(node)),
    edges: transport.edges.map((edge) => ({
      from_node_id: edge.from_node_id,
      to_node_id: edge.to_node_id,
    })),
  };
}

export function draftToWorkflowCreatePayload(draft: WorkflowCanvasDraft): WorkflowCreatePayload {
  return {
    task_kind: 'workflow',
    workflow: draftToWorkflowDefinition(draft),
    ...scheduleFieldsFromDraft(draft),
  };
}

export function draftToWorkflowUpdatePayload(draft: WorkflowCanvasDraft): WorkflowUpdatePayload {
  return {
    task_kind: 'workflow',
    workflow: draftToWorkflowDefinition(draft),
    ...scheduleFieldsFromDraft(draft),
  };
}

export function withWorkflowContent(
  draft: WorkflowCanvasDraft,
  nodes: WorkflowCanvasDraft['nodes'],
  edges: WorkflowCanvasDraft['edges'],
): WorkflowCanvasDraft {
  return {
    ...draft,
    nodes,
    edges,
    selectedNodeId: undefined,
  };
}

function scheduleDraftFromTask(
  scheduleType: WorkflowTaskPayload['schedule_type'],
  intervalSeconds?: number,
  cronExpr?: string,
): WorkflowCanvasDraft['schedule'] {
  if (scheduleType === 'interval') {
    return {
      mode: 'interval',
      intervalSeconds: String(intervalSeconds ?? DEFAULT_INTERVAL_SECONDS),
      cronExpr: '',
    };
  }

  return {
    mode: 'cron',
    intervalSeconds: DEFAULT_INTERVAL_SECONDS,
    cronExpr: cronExpr ?? '',
  };
}

function buildWorkflowNode(node: WorkflowCanvasNodeDraft): WorkflowNode {
  if (node.type === 'group') {
    throw new Error('group nodes are not supported in workflow definitions');
  }
  const loopIterations = node.loop?.max_iterations ?? DEFAULT_LOOP_MAX_ITERATIONS;
  return {
    id: node.id,
    type: node.type,
    start: buildStartWorkflowNodePayload(node),
    tool: buildToolWorkflowNodePayload(node),
    llm: buildLLMWorkflowNodePayload(node),
    agent: buildAgentWorkflowNodePayload(node),
    if: buildIfWorkflowNodePayload(node),
    loop: buildLoopWorkflowNodePayload(node, loopIterations),
  };
}

function buildStartWorkflowNodePayload(node: WorkflowCanvasNodeDraft): WorkflowNode['start'] {
  return node.type === 'start' ? { inputs: cloneInputs(node.start?.inputs) } : undefined;
}

function buildToolWorkflowNodePayload(node: WorkflowCanvasNodeDraft): WorkflowNode['tool'] {
  if (node.type !== 'tool') {
    return undefined;
  }
  return {
    tool_name: node.tool?.tool_name ?? '',
    arguments: cloneObject(node.tool?.arguments),
  };
}

function buildLLMWorkflowNodePayload(node: WorkflowCanvasNodeDraft): WorkflowNode['llm'] {
  if (node.type !== 'llm') {
    return undefined;
  }
  return {
    prompt: node.llm?.prompt ?? '',
    system_prompt: node.llm?.system_prompt,
  };
}

function buildAgentWorkflowNodePayload(node: WorkflowCanvasNodeDraft): WorkflowNode['agent'] {
  if (node.type !== 'agent') {
    return undefined;
  }
  return {
    message: node.agent?.message ?? '',
    runtime_overrides: cloneWorkflowTaskRuntimeOverrides(node.agent?.runtime_overrides),
  };
}

function buildIfWorkflowNodePayload(node: WorkflowCanvasNodeDraft): WorkflowNode['if'] {
  if (node.type !== 'if') {
    return undefined;
  }
  const ifConfig = node.if;
  return {
    source_node_id: workflowIfSourceNodeID(ifConfig),
    operator: workflowIfOperator(ifConfig),
    value: workflowIfValue(ifConfig),
    true_node_id: workflowIfTrueNodeID(ifConfig),
    false_node_id: workflowIfFalseNodeID(ifConfig),
  };
}

function workflowIfSourceNodeID(ifConfig: WorkflowCanvasNodeDraft['if']): string | undefined {
  const trimmed = ifConfig?.source_node_id?.trim() ?? '';
  return trimmed.length > 0 ? trimmed : undefined;
}

function workflowIfOperator(ifConfig: WorkflowCanvasNodeDraft['if']): NonNullable<WorkflowNode['if']>['operator'] {
  return ifConfig?.operator ?? 'equals';
}

function workflowIfValue(ifConfig: WorkflowCanvasNodeDraft['if']): string {
  return optionalText(ifConfig?.value);
}

function workflowIfTrueNodeID(ifConfig: WorkflowCanvasNodeDraft['if']): string {
  return optionalText(ifConfig?.true_node_id);
}

function workflowIfFalseNodeID(ifConfig: WorkflowCanvasNodeDraft['if']): string {
  return optionalText(ifConfig?.false_node_id);
}

function optionalText(value: string | undefined): string {
  return value ?? '';
}

function buildLoopWorkflowNodePayload(
  node: WorkflowCanvasNodeDraft,
  loopIterations: number,
): WorkflowNode['loop'] {
  if (node.type !== 'loop') {
    return undefined;
  }
  return {
    max_iterations: normalizeLoopIterations(loopIterations),
    body_node_id: node.loop?.body_node_id ?? '',
    exit_node_id: node.loop?.exit_node_id ?? '',
  };
}

function scheduleFieldsFromDraft(draft: WorkflowCanvasDraft): {
  interval_seconds?: number;
  cron_expr?: string;
} {
  if (draft.schedule.mode === 'interval') {
    return {
      interval_seconds: parseIntervalSeconds(draft.schedule.intervalSeconds),
    };
  }

  const cronExpr = draft.schedule.cronExpr.trim();
  if (cronExpr.length === 0) {
    throw new Error('cron expression is required');
  }

  return {
    cron_expr: cronExpr,
  };
}

function parseIntervalSeconds(raw: string): number {
  const parsed = Number.parseInt(raw.trim(), DECIMAL_RADIX);
  if (!Number.isFinite(parsed) || parsed < MIN_INTERVAL_SECONDS) {
    throw new Error('interval seconds must be a positive integer');
  }

  return parsed;
}

function cloneInputs(inputs?: WorkflowInputVariable[]): WorkflowInputVariable[] | undefined {
  if (!inputs) {
    return undefined;
  }

  return inputs.map((input) => ({ ...input }));
}

function cloneObject(input?: Record<string, unknown>): Record<string, unknown> | undefined {
  if (!input) {
    return undefined;
  }

  return { ...input };
}

function normalizeLoopIterations(value: number): number {
  if (!Number.isFinite(value) || value < 1) {
    return DEFAULT_LOOP_MAX_ITERATIONS;
  }
  return Math.floor(value);
}
