import type {
  OrchestrationDefinition,
  OrchestrationTaskPayload,
} from '@/lib/types';
import { cloneOrchestrationTaskRuntimeOverrides } from '@/lib/workflow-editor/agentRuntime';
import {
  DECIMAL_RADIX,
  DEFAULT_INTERVAL_SECONDS,
  DEFAULT_ORCHESTRATION_GROUP_MAX_ROUNDS,
  MIN_INTERVAL_SECONDS,
  NODE_X_GAP,
  NODE_Y_BASE,
} from '@/lib/workflow-editor/constants';
import type {
  OrchestrationCreatePayload,
  OrchestrationDefinitionImport,
  WorkflowCanvasDraft,
  WorkflowCanvasEdgeDraft,
  WorkflowCanvasNodeDraft,
  WorkflowUpdatePayload,
} from '@/lib/workflow-editor/types';

const GROUP_MEMBER_Y_GAP = 150;
const GROUP_LANE_Y = NODE_Y_BASE + 140;
const ORCHESTRATION_GROUP_TYPES = ['group', 'agent'] as const;

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
  const layout = buildNodePositionMap(normalized);
  return {
    mode: 'create',
    schedule: scheduleDraftFromTask(input.scheduleType, input.intervalSeconds, input.cronExpr),
    nodes: normalized.nodes.map((node) => buildDraftNode(node, layout.get(node.id) ?? fallbackPosition(node.id))),
    edges: normalized.edges.map((edge, index) => buildDraftEdge(edge.from_node_id, edge.to_node_id, edge.kind, index + 1)),
  };
}

export function draftToOrchestrationDefinition(draft: WorkflowCanvasDraft): OrchestrationDefinition {
  const nodeMap = new Map(draft.nodes.map((node) => [node.id, node] as const));
  return {
    nodes: draft.nodes
      .filter((node) => node.type === 'group' || node.type === 'agent')
      .map((node) => buildOrchestrationNode(node)),
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

function buildDraftNode(node: OrchestrationDefinition['nodes'][number], position: { x: number; y: number }): WorkflowCanvasNodeDraft {
  return {
    id: node.id,
    type: node.type,
    position,
    ui: { toolArgumentsMode: 'kv' },
    agent: node.type === 'agent'
      ? {
        title: node.agent?.title ?? '',
        message: node.agent?.message ?? '',
        runtime_overrides: cloneOrchestrationTaskRuntimeOverrides(node.agent?.runtime_overrides),
      }
      : undefined,
    group: node.type === 'group'
      ? {
        title: node.group?.title ?? '',
        shared_context: node.group?.shared_context ?? '',
        speaking_mode: node.group?.speaking_mode ?? 'sequential',
        owner_agent_id: node.group?.owner_agent_id ?? '',
        max_rounds: node.group?.max_rounds ?? DEFAULT_ORCHESTRATION_GROUP_MAX_ROUNDS,
      }
      : undefined,
  };
}

function buildOrchestrationNode(node: WorkflowCanvasNodeDraft): OrchestrationDefinition['nodes'][number] {
  return {
    id: node.id,
    type: node.type as 'group' | 'agent',
    group: node.type === 'group'
      ? {
        title: node.group?.title ?? '',
        shared_context: node.group?.shared_context ?? '',
        speaking_mode: node.group?.speaking_mode ?? 'sequential',
        owner_agent_id: node.group?.owner_agent_id ?? '',
        max_rounds: node.group?.max_rounds ?? DEFAULT_ORCHESTRATION_GROUP_MAX_ROUNDS,
      }
      : undefined,
    agent: node.type === 'agent'
      ? {
        title: node.agent?.title ?? '',
        message: node.agent?.message ?? '',
        runtime_overrides: cloneOrchestrationTaskRuntimeOverrides(node.agent?.runtime_overrides),
      }
      : undefined,
  };
}

function buildOrchestrationEdge(
  edge: WorkflowCanvasEdgeDraft,
  nodeMap: Map<string, WorkflowCanvasNodeDraft>,
) {
  const sourceNode = nodeMap.get(edge.from_node_id);
  const targetNode = nodeMap.get(edge.to_node_id);
  const resolvedKind = resolveEdgeKind(sourceNode, targetNode);
  const kind = edge.kind ?? resolvedKind;
  if (edge.kind && edge.kind !== resolvedKind) {
    return undefined;
  }
  if (!kind) {
    return undefined;
  }
  return {
    from_node_id: edge.from_node_id,
    to_node_id: edge.to_node_id,
    kind,
  };
}

function buildDraftEdge(fromNodeID: string, toNodeID: string, kind: 'control' | 'member', index: number): WorkflowCanvasEdgeDraft {
  return { id: `edge-${index}-${fromNodeID}-${toNodeID}-${kind}`, from_node_id: fromNodeID, to_node_id: toNodeID, kind };
}

function buildNodePositionMap(definition: OrchestrationDefinition): Map<string, { x: number; y: number }> {
  const positions = new Map<string, { x: number; y: number }>();
  const controlNext = new Map<string, string>();
  for (const edge of definition.edges) {
    if (edge.kind === 'control') {
      controlNext.set(edge.from_node_id, edge.to_node_id);
    }
  }
  let currentID = findEntryGroupID(definition);
  let index = 0;
  while (currentID && !positions.has(currentID)) {
    positions.set(currentID, { x: index * NODE_X_GAP, y: GROUP_LANE_Y });
    currentID = controlNext.get(currentID) ?? '';
    index += 1;
  }
  for (const node of definition.nodes.filter((item) => item.type === 'group')) {
    const base = positions.get(node.id) ?? fallbackPosition(node.id);
    const members = definition.edges.filter((edge) => edge.kind === 'member' && edge.to_node_id === node.id);
    members.forEach((edge, memberIndex) => {
      if (!positions.has(edge.from_node_id)) {
        positions.set(edge.from_node_id, {
          x: base.x,
          y: base.y - GROUP_MEMBER_Y_GAP * (memberIndex + 1),
        });
      }
    });
  }
  return positions;
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

function fallbackPosition(nodeID: string): { x: number; y: number } {
  const seed = Math.max(nodeID.length, 1);
  return { x: seed * NODE_X_GAP, y: GROUP_LANE_Y };
}

function normalizeOrchestrationDefinition(definition: OrchestrationDefinition): OrchestrationDefinition {
  const nodes = definition.nodes.filter((node) =>
    ORCHESTRATION_GROUP_TYPES.includes(node.type as (typeof ORCHESTRATION_GROUP_TYPES)[number]));
  const nodeMap = new Map(nodes.map((node) => [node.id, node] as const));
  return {
    nodes,
    edges: definition.edges.filter((edge) => {
      const source = nodeMap.get(edge.from_node_id);
      const target = nodeMap.get(edge.to_node_id);
      if (!source || !target) {
        return false;
      }
      if (edge.kind === 'member') {
        return source.type === 'agent' && target.type === 'group';
      }
      return source.type === 'group' && target.type === 'group';
    }),
  };
}

function findEntryGroupID(definition: OrchestrationDefinition): string {
  const groupIDs = new Set(
    definition.nodes
      .filter((node) => node.type === 'group')
      .map((node) => node.id),
  );
  const controlTargets = new Set(
    definition.edges
      .filter((edge) => edge.kind === 'control' && groupIDs.has(edge.to_node_id))
      .map((edge) => edge.to_node_id),
  );
  for (const node of definition.nodes) {
    if (node.type === 'group' && !controlTargets.has(node.id)) {
      return node.id;
    }
  }
  return definition.nodes.find((node) => node.type === 'group')?.id ?? '';
}
