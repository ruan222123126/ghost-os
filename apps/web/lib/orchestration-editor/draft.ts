import type {
  OrchestrationDefinition,
  OrchestrationAgentNode,
  OrchestrationEdge,
  OrchestrationGroupNode,
  OrchestrationNode,
  OrchestrationTaskPayload,
} from '@/lib/types';
import {
  buildOrchestrationNodePositionMap,
  fallbackOrchestrationNodePosition,
  type OrchestrationNodePosition,
} from '@/lib/orchestration-editor/draftLayout';
import { cloneOrchestrationTaskRuntimeOverrides } from '@/lib/workflow-editor/agentRuntime';
import {
  DECIMAL_RADIX,
  DEFAULT_INTERVAL_SECONDS,
  DEFAULT_ORCHESTRATION_GROUP_MAX_ROUNDS,
  MIN_INTERVAL_SECONDS,
} from '@/lib/workflow-editor/constants';
import type {
  OrchestrationCreatePayload,
  OrchestrationDefinitionImport,
  WorkflowCanvasDraft,
  WorkflowCanvasEdgeDraft,
  WorkflowCanvasNodeDraft,
  WorkflowUpdatePayload,
} from '@/lib/workflow-editor/types';

export function createEmptyOrchestrationDraft(mode: 'create' | 'edit' = 'create'): WorkflowCanvasDraft {
  return {
    mode,
    schedule: {
      mode: 'interval',
      intervalSeconds: DEFAULT_INTERVAL_SECONDS,
      cronExpr: '',
    },
    nodes: [],
    edges: [],
  };
}

export function orchestrationTaskToDraft(task: OrchestrationTaskPayload): WorkflowCanvasDraft {
  return {
    ...orchestrationDefinitionToDraft({
      orchestration: task.orchestration,
      scheduleType: task.schedule_type,
      intervalSeconds: task.interval_seconds,
      cronExpr: task.cron_expr,
    }),
    mode: 'edit',
    taskId: task.id,
  };
}

export function orchestrationDefinitionToDraft(input: OrchestrationDefinitionImport): WorkflowCanvasDraft {
  const normalized = normalizeOrchestrationDefinition(input.orchestration);
  const layout = buildOrchestrationNodePositionMap(normalized);
  return {
    mode: 'create',
    schedule: scheduleDraftFromTask(input.scheduleType, input.intervalSeconds, input.cronExpr),
    nodes: normalized.nodes.map((node) => buildDraftNode(node, layout.get(node.id) ?? fallbackOrchestrationNodePosition(node.id))),
    edges: normalized.edges.map((edge, index) => buildDraftEdge(edge, index + 1)),
  };
}

export function draftToOrchestrationDefinition(draft: WorkflowCanvasDraft): OrchestrationDefinition {
  const nodeMap = new Map(draft.nodes.map((node) => [node.id, node] as const));
  return {
    nodes: draft.nodes
      .map((node) => buildOrchestrationNode(node))
      .filter((node): node is NonNullable<typeof node> => node !== undefined),
    edges: draft.edges
      .map((edge) => buildOrchestrationEdge(edge, nodeMap))
      .filter((edge): edge is NonNullable<typeof edge> => edge !== undefined),
  };
}

export function draftToOrchestrationCreatePayload(
  draft: WorkflowCanvasDraft,
  name: string,
): OrchestrationCreatePayload {
  return {
    task_kind: 'orchestration',
    name: name.trim(),
    orchestration: draftToOrchestrationDefinition(draft),
    ...scheduleFieldsFromDraft(draft),
  };
}

export function draftToOrchestrationUpdatePayload(
  draft: WorkflowCanvasDraft,
  name?: string,
): Pick<WorkflowUpdatePayload, 'task_kind' | 'name' | 'orchestration' | 'interval_seconds' | 'cron_expr'> {
  return {
    task_kind: 'orchestration',
    name: name?.trim() || undefined,
    orchestration: draftToOrchestrationDefinition(draft),
    ...scheduleFieldsFromDraft(draft),
  };
}

function buildDraftNode(node: OrchestrationNode, position: OrchestrationNodePosition): WorkflowCanvasNodeDraft {
  return {
    id: node.id,
    type: node.type,
    position,
    ui: { toolArgumentsMode: 'kv' },
    agent: buildDraftAgentPayload(node),
    group: buildDraftGroupPayload(node),
  };
}

function buildOrchestrationNode(node: WorkflowCanvasNodeDraft): OrchestrationDefinition['nodes'][number] | undefined {
  if (!isOrchestrationDraftNode(node)) {
    return undefined;
  }
  return {
    id: node.id,
    type: node.type,
    group: buildOrchestrationGroupPayload(node),
    agent: buildOrchestrationAgentPayload(node),
  };
}

function buildDraftAgentPayload(node: OrchestrationNode): WorkflowCanvasNodeDraft['agent'] {
  return node.type === 'agent' ? buildAgentPayload(node.agent) : undefined;
}

function buildDraftGroupPayload(node: OrchestrationNode): WorkflowCanvasNodeDraft['group'] {
  return node.type === 'group' ? buildGroupPayload(node.group) : undefined;
}

function buildOrchestrationAgentPayload(node: WorkflowCanvasNodeDraft): OrchestrationAgentNode | undefined {
  return node.type === 'agent' ? buildAgentPayload(node.agent) : undefined;
}

function buildOrchestrationGroupPayload(node: WorkflowCanvasNodeDraft): OrchestrationGroupNode | undefined {
  return node.type === 'group' ? buildGroupPayload(node.group) : undefined;
}

function buildAgentPayload(
  agent?: OrchestrationAgentNode | WorkflowCanvasNodeDraft['agent'],
): OrchestrationAgentNode {
  const { title = '', message = '', runtime_overrides } = agent ?? {};
  return {
    title,
    message,
    runtime_overrides: cloneOrchestrationTaskRuntimeOverrides(runtime_overrides),
  };
}

function buildGroupPayload(
  group?: OrchestrationGroupNode | WorkflowCanvasNodeDraft['group'],
): OrchestrationGroupNode {
  const {
    title = '',
    shared_context = '',
    speaking_mode = 'sequential',
    owner_agent_id = '',
    max_rounds = DEFAULT_ORCHESTRATION_GROUP_MAX_ROUNDS,
  } = group ?? {};
  return {
    title,
    shared_context,
    speaking_mode,
    owner_agent_id,
    max_rounds,
  };
}

function buildOrchestrationEdge(
  edge: WorkflowCanvasEdgeDraft,
  nodeMap: Map<string, WorkflowCanvasNodeDraft>,
) {
  const sourceNode = nodeMap.get(edge.from_node_id);
  const targetNode = nodeMap.get(edge.to_node_id);
  const resolvedKind = resolveEdgeKind(sourceNode, targetNode);
  if (!isOrchestrationDraftNode(sourceNode) || !isOrchestrationDraftNode(targetNode)) {
    return undefined;
  }
  if (edge.kind) {
    return {
      from_node_id: edge.from_node_id,
      to_node_id: edge.to_node_id,
      kind: edge.kind,
    };
  }
  if (!resolvedKind) {
    return undefined;
  }
  return {
    from_node_id: edge.from_node_id,
    to_node_id: edge.to_node_id,
    kind: resolvedKind,
  };
}

function isOrchestrationDraftNode(
  node?: WorkflowCanvasNodeDraft,
): node is WorkflowCanvasNodeDraft & { type: OrchestrationDefinition['nodes'][number]['type'] } {
  return node?.type === 'group' || node?.type === 'agent';
}

function buildDraftEdge(edge: OrchestrationEdge, index: number): WorkflowCanvasEdgeDraft {
  return {
    id: `edge-${index}-${edge.from_node_id}-${edge.to_node_id}-${edge.kind}`,
    from_node_id: edge.from_node_id,
    to_node_id: edge.to_node_id,
    kind: edge.kind,
  };
}

function resolveEdgeKind(
  sourceNode?: WorkflowCanvasNodeDraft,
  targetNode?: WorkflowCanvasNodeDraft,
): 'control' | 'member' | undefined {
  if (!sourceNode || !targetNode) {
    return undefined;
  }
  if (sourceNode.type === 'agent' && targetNode.type === 'group') {
    return 'member';
  }
  if (sourceNode.type === 'group' && targetNode.type === 'group') {
    return 'control';
  }
  return undefined;
}

function scheduleDraftFromTask(
  scheduleType: OrchestrationTaskPayload['schedule_type'],
  intervalSeconds?: number,
  cronExpr?: string,
): WorkflowCanvasDraft['schedule'] {
  if (scheduleType === 'interval') {
    return { mode: 'interval', intervalSeconds: String(intervalSeconds ?? DEFAULT_INTERVAL_SECONDS), cronExpr: '' };
  }
  return { mode: 'cron', intervalSeconds: DEFAULT_INTERVAL_SECONDS, cronExpr: cronExpr ?? '' };
}

function scheduleFieldsFromDraft(draft: WorkflowCanvasDraft): { interval_seconds?: number; cron_expr?: string } {
  if (draft.schedule.mode === 'interval') {
    return { interval_seconds: parseIntervalSeconds(draft.schedule.intervalSeconds) };
  }
  const cronExpr = draft.schedule.cronExpr.trim();
  if (!cronExpr) {
    throw new Error('cron expression is required');
  }
  return { cron_expr: cronExpr };
}

function parseIntervalSeconds(raw: string): number {
  const parsed = Number.parseInt(raw.trim(), DECIMAL_RADIX);
  if (!Number.isFinite(parsed) || parsed < MIN_INTERVAL_SECONDS) {
    throw new Error('interval seconds must be a positive integer');
  }
  return parsed;
}

function normalizeOrchestrationDefinition(definition: OrchestrationDefinition): OrchestrationDefinition {
  return {
    nodes: definition.nodes.map((node) => ({ ...node })),
    edges: definition.edges.map((edge) => ({ ...edge })),
  };
}
