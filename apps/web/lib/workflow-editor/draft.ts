import type {
  WorkflowDefinition,
  WorkflowInputVariable,
  WorkflowNode,
  WorkflowTaskPayload,
} from '@/lib/types';
import {
  DECIMAL_RADIX,
  DEFAULT_END_NODE_ID,
  DEFAULT_INTERVAL_SECONDS,
  DEFAULT_START_NODE_ID,
  MIN_INTERVAL_SECONDS,
  NODE_X_GAP,
  NODE_Y_BASE,
} from '@/lib/workflow-editor/constants';
import type {
  WorkflowCanvasDraft,
  WorkflowCanvasEdgeDraft,
  WorkflowCanvasNodeDraft,
  WorkflowCreatePayload,
  WorkflowDefinitionImport,
  WorkflowUpdatePayload,
} from '@/lib/workflow-editor/types';

const DEFAULT_TOOL_ARGUMENTS_MODE = 'kv';

export function createEmptyWorkflowDraft(mode: 'create' | 'edit' = 'create'): WorkflowCanvasDraft {
  const nodes = [
    createCanvasNode({ id: DEFAULT_START_NODE_ID, type: 'start' }, 0),
    createCanvasNode({ id: DEFAULT_END_NODE_ID, type: 'end' }, 1),
  ];

  return {
    mode,
    schedule: {
      mode: 'interval',
      intervalSeconds: DEFAULT_INTERVAL_SECONDS,
      cronExpr: '',
    },
    nodes,
    edges: [buildEdgeDraft(DEFAULT_START_NODE_ID, DEFAULT_END_NODE_ID, 0)],
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
  const nodes = input.workflow.nodes.map((node, index) => createCanvasNode(node, index));
  const edges = input.workflow.edges.map((edge, index) => buildEdgeDraft(edge.from_node_id, edge.to_node_id, index));

  return {
    mode: 'create',
    schedule: scheduleDraftFromTask(input.scheduleType, input.intervalSeconds, input.cronExpr),
    nodes,
    edges,
  };
}

export function draftToWorkflowDefinition(draft: WorkflowCanvasDraft): WorkflowDefinition {
  return {
    nodes: draft.nodes.map((node) => buildWorkflowNode(node)),
    edges: draft.edges.map((edge) => ({
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
  nodes: WorkflowCanvasNodeDraft[],
  edges: WorkflowCanvasEdgeDraft[],
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

function createCanvasNode(node: WorkflowNode, index: number): WorkflowCanvasNodeDraft {
  return {
    id: node.id,
    type: node.type,
    position: {
      x: index * NODE_X_GAP,
      y: NODE_Y_BASE,
    },
    ui: {
      toolArgumentsMode: DEFAULT_TOOL_ARGUMENTS_MODE,
    },
    start: node.start ? { inputs: cloneInputs(node.start.inputs) } : undefined,
    tool: node.tool
      ? {
        tool_name: node.tool.tool_name,
        arguments: cloneObject(node.tool.arguments),
      }
      : undefined,
    llm: node.llm
      ? {
        prompt: node.llm.prompt,
        system_prompt: node.llm.system_prompt,
      }
      : undefined,
    agent: node.agent ? { message: node.agent.message } : undefined,
  };
}

function buildEdgeDraft(fromNodeID: string, toNodeID: string, index: number): WorkflowCanvasEdgeDraft {
  return {
    id: `edge-${index}-${fromNodeID}-${toNodeID}`,
    from_node_id: fromNodeID,
    to_node_id: toNodeID,
  };
}

function buildWorkflowNode(node: WorkflowCanvasNodeDraft): WorkflowNode {
  return {
    id: node.id,
    type: node.type,
    start: node.type === 'start' ? { inputs: cloneInputs(node.start?.inputs) } : undefined,
    tool: node.type === 'tool'
      ? {
        tool_name: node.tool?.tool_name ?? '',
        arguments: cloneObject(node.tool?.arguments),
      }
      : undefined,
    llm: node.type === 'llm'
      ? {
        prompt: node.llm?.prompt ?? '',
        system_prompt: node.llm?.system_prompt,
      }
      : undefined,
    agent: node.type === 'agent' ? { message: node.agent?.message ?? '' } : undefined,
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

  return inputs.map((input) => ({
    ...input,
  }));
}

function cloneObject(input?: Record<string, unknown>): Record<string, unknown> | undefined {
  if (!input) {
    return undefined;
  }

  return { ...input };
}
