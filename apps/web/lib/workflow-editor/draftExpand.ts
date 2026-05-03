import type {
  WorkflowDefinition,
  WorkflowInputVariable,
  WorkflowNode,
} from '@/lib/types';
import { cloneWorkflowTaskRuntimeOverrides } from '@/lib/workflow-editor/agentRuntime';
import {
  LOOP_ROLE_END,
  LOOP_ROLE_START,
  NODE_X_GAP,
  NODE_Y_BASE,
} from '@/lib/workflow-editor/constants';
import { normalizeScreenControlComposerAction } from '@/lib/workflow-editor/screenControlComposer';
import type {
  ScreenControlComposerStep,
  WorkflowCanvasEdgeDraft,
  WorkflowCanvasNodeDraft,
} from '@/lib/workflow-editor/types';

const DEFAULT_TOOL_ARGUMENTS_MODE = 'kv';
const SCREEN_CONTROL_TOOL_NAME = 'screen_control';
const SCREEN_CONTROL_WORKFLOW_STEPS_KEY = 'workflow_steps';
const SCREEN_CONTROL_ACTION_KEY = 'action';
const SCREEN_CONTROL_PARAMS_KEY = 'params';
const SCREEN_CONTROL_COMPOSER_ACTIONS = new Set(['screenshot', 'find_text', 'find_icon', 'click']);

interface WorkflowCanvasNodeSource {
  id: string;
  type: WorkflowCanvasNodeDraft['type'];
  start?: WorkflowCanvasNodeDraft['start'];
  tool?: WorkflowCanvasNodeDraft['tool'];
  llm?: WorkflowCanvasNodeDraft['llm'];
  agent?: WorkflowCanvasNodeDraft['agent'];
  if?: WorkflowCanvasNodeDraft['if'];
  loop?: WorkflowCanvasNodeDraft['loop'];
}

export function buildCanvasGraphFromWorkflowDefinition(workflow: WorkflowDefinition): {
  nodes: WorkflowCanvasNodeDraft[];
  edges: WorkflowCanvasEdgeDraft[];
} {
  const expanded = expandLoopPairsForDraft(workflow);
  return {
    nodes: expanded.nodes.map((node, index) => createCanvasNode(node, index)),
    edges: expanded.edges.map((edge, index) => buildEdgeDraft(edge.from_node_id, edge.to_node_id, index)),
  };
}
function createCanvasNode(node: WorkflowCanvasNodeSource, index: number): WorkflowCanvasNodeDraft {
  const screenControlComposer = extractScreenControlComposer(node.tool);
  return {
    id: node.id,
    type: node.type,
    position: {
      x: index * NODE_X_GAP,
      y: NODE_Y_BASE,
    },
    ui: {
      toolArgumentsMode: DEFAULT_TOOL_ARGUMENTS_MODE,
      screenControlComposer,
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
    agent: node.agent
      ? {
        message: node.agent.message,
        runtime_overrides: cloneWorkflowTaskRuntimeOverrides(node.agent.runtime_overrides),
      }
      : undefined,
    if: node.if
      ? {
        source_node_id: node.if.source_node_id,
        operator: node.if.operator,
        value: node.if.value,
        true_node_id: node.if.true_node_id,
        false_node_id: node.if.false_node_id,
      }
      : undefined,
    loop: node.loop
      ? {
        role: node.loop.role,
        loop_id: node.loop.loop_id,
        max_iterations: node.loop.max_iterations,
        body_node_id: node.loop.body_node_id,
        exit_node_id: node.loop.exit_node_id,
      }
      : undefined,
  };
}

function extractScreenControlComposer(
  tool: WorkflowCanvasNodeSource['tool'],
): WorkflowCanvasNodeDraft['ui']['screenControlComposer'] {
  if (!tool || tool.tool_name.trim() !== SCREEN_CONTROL_TOOL_NAME) {
    return undefined;
  }
  const args = asRecord(tool.arguments);
  const workflowSteps = readScreenControlWorkflowSteps(args[SCREEN_CONTROL_WORKFLOW_STEPS_KEY]);
  if (workflowSteps.length > 0) {
    return { steps: workflowSteps };
  }
  const singleStep = readScreenControlSingleStep(args);
  if (!singleStep) {
    return undefined;
  }
  return { steps: [singleStep] };
}

function readScreenControlWorkflowSteps(input: unknown): ScreenControlComposerStep[] {
  if (!Array.isArray(input)) {
    return [];
  }
  const steps = input
    .map((item) => readScreenControlComposerStep(asRecord(item)))
    .filter((step): step is ScreenControlComposerStep => !!step);
  return steps;
}

function readScreenControlSingleStep(args: Record<string, unknown>): ScreenControlComposerStep | undefined {
  return readScreenControlComposerStep({
    action: args[SCREEN_CONTROL_ACTION_KEY],
    params: args[SCREEN_CONTROL_PARAMS_KEY],
  });
}

function readScreenControlComposerStep(input: Record<string, unknown>): ScreenControlComposerStep | undefined {
  const action = readScreenControlComposerAction(input.action);
  if (!action) {
    return undefined;
  }
  const params = cloneObject(asRecord(input.params));
  if (!params || Object.keys(params).length === 0) {
    return { action };
  }
  return { action, params };
}

function readScreenControlComposerAction(input: unknown): ScreenControlComposerStep['action'] | undefined {
  if (typeof input !== 'string') {
    return undefined;
  }
  const normalized = normalizeScreenControlComposerAction(input.trim() as ScreenControlComposerStep['action']);
  if (!SCREEN_CONTROL_COMPOSER_ACTIONS.has(normalized)) {
    return undefined;
  }
  return normalized;
}
function expandLoopPairsForDraft(workflow: WorkflowDefinition): {
  nodes: WorkflowCanvasNodeSource[];
  edges: WorkflowDefinition['edges'];
} {
  const loopNodes = new Map<string, NonNullable<WorkflowNode['loop']>>();
  const outgoing = buildOutgoingMap(workflow.edges);
  const nodes: WorkflowCanvasNodeSource[] = [];
  for (const node of workflow.nodes) {
    if (node.type !== 'loop' || !node.loop) {
      nodes.push(workflowNodeToCanvasSource(node));
      continue;
    }
    loopNodes.set(node.id, node.loop);
    nodes.push(buildLoopStartNodeSource(node.id, node.loop));
    nodes.push(buildLoopEndNodeSource(node.id));
  }
  return {
    nodes,
    edges: rewriteLoopEdgesForDraft(workflow.edges, loopNodes, outgoing),
  };
}
function workflowNodeToCanvasSource(node: WorkflowNode): WorkflowCanvasNodeSource {
  return {
    id: node.id,
    type: node.type,
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
    agent: node.agent
      ? {
        message: node.agent.message,
        runtime_overrides: cloneWorkflowTaskRuntimeOverrides(node.agent.runtime_overrides),
      }
      : undefined,
    if: node.if
      ? {
        source_node_id: node.if.source_node_id,
        operator: node.if.operator,
        value: node.if.value,
        true_node_id: node.if.true_node_id,
        false_node_id: node.if.false_node_id,
      }
      : undefined,
    loop: node.loop
      ? {
        role: LOOP_ROLE_START,
        loop_id: node.id,
        max_iterations: node.loop.max_iterations,
        body_node_id: node.loop.body_node_id,
        exit_node_id: node.loop.exit_node_id,
      }
      : undefined,
  };
}
function buildLoopStartNodeSource(
  loopID: string,
  loop: NonNullable<WorkflowNode['loop']>,
): WorkflowCanvasNodeSource {
  return {
    id: loopStartNodeID(loopID),
    type: 'loop',
    loop: {
      role: LOOP_ROLE_START,
      loop_id: loopID,
      max_iterations: loop.max_iterations,
    },
  };
}
function buildLoopEndNodeSource(loopID: string): WorkflowCanvasNodeSource {
  return {
    id: loopEndNodeID(loopID),
    type: 'loop',
    loop: {
      role: LOOP_ROLE_END,
      loop_id: loopID,
    },
  };
}
function rewriteLoopEdgesForDraft(
  edges: WorkflowDefinition['edges'],
  loopNodes: Map<string, NonNullable<WorkflowNode['loop']>>,
  outgoing: Map<string, string[]>,
): WorkflowDefinition['edges'] {
  const loopBackSources = collectLoopBackSources(loopNodes, edges, outgoing);
  const rewritten: WorkflowDefinition['edges'] = [];
  for (const edge of edges) {
    const fromLoop = loopNodes.get(edge.from_node_id);
    if (fromLoop) {
      if (edge.to_node_id === fromLoop.body_node_id) {
        rewritten.push({
          from_node_id: loopStartNodeID(edge.from_node_id),
          to_node_id: fromLoop.body_node_id,
        });
      }
      continue;
    }
    const toLoop = loopNodes.get(edge.to_node_id);
    if (!toLoop) {
      rewritten.push(edge);
      continue;
    }
    const loopID = edge.to_node_id;
    const isLoopBack = loopBackSources.get(loopID)?.has(edge.from_node_id) ?? false;
    rewritten.push({
      from_node_id: edge.from_node_id,
      to_node_id: isLoopBack ? loopEndNodeID(loopID) : loopStartNodeID(loopID),
    });
  }
  for (const [loopID, loop] of loopNodes.entries()) {
    rewritten.push({ from_node_id: loopEndNodeID(loopID), to_node_id: loop.exit_node_id });
  }
  return dedupeWorkflowEdges(rewritten);
}
function collectLoopBackSources(
  loopNodes: Map<string, NonNullable<WorkflowNode['loop']>>,
  edges: WorkflowDefinition['edges'],
  outgoing: Map<string, string[]>,
): Map<string, Set<string>> {
  const loopBackSources = new Map<string, Set<string>>();
  for (const [loopID, loop] of loopNodes.entries()) {
    const incoming = edges
      .filter((edge) => edge.to_node_id === loopID)
      .map((edge) => edge.from_node_id)
      .filter((sourceID) => sourceID !== loopID);
    const reachableFromBody = collectReachableNodes(outgoing, loop.body_node_id, loopID);
    loopBackSources.set(
      loopID,
      new Set(incoming.filter((sourceID) => reachableFromBody.has(sourceID))),
    );
  }
  return loopBackSources;
}
function collectReachableNodes(
  outgoing: Map<string, string[]>,
  startID: string,
  blockedID: string,
): Set<string> {
  const reachable = new Set<string>();
  const queue = [startID];
  while (queue.length > 0) {
    const nodeID = queue.shift();
    if (!nodeID || nodeID === blockedID || reachable.has(nodeID)) {
      continue;
    }
    reachable.add(nodeID);
    for (const nextID of outgoing.get(nodeID) ?? []) {
      if (nextID !== blockedID && !reachable.has(nextID)) {
        queue.push(nextID);
      }
    }
  }
  return reachable;
}
function buildOutgoingMap(edges: WorkflowDefinition['edges']): Map<string, string[]> {
  const outgoing = new Map<string, string[]>();
  for (const edge of edges) {
    outgoing.set(edge.from_node_id, [...(outgoing.get(edge.from_node_id) ?? []), edge.to_node_id]);
  }
  return outgoing;
}
function dedupeWorkflowEdges(edges: WorkflowDefinition['edges']): WorkflowDefinition['edges'] {
  const seen = new Set<string>();
  const unique: WorkflowDefinition['edges'] = [];
  for (const edge of edges) {
    const key = `${edge.from_node_id}->${edge.to_node_id}`;
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    unique.push(edge);
  }
  return unique;
}
function loopStartNodeID(loopID: string): string {
  return `${loopID}-${LOOP_ROLE_START}`;
}

function loopEndNodeID(loopID: string): string {
  return `${loopID}-${LOOP_ROLE_END}`;
}
function buildEdgeDraft(fromNodeID: string, toNodeID: string, index: number): WorkflowCanvasEdgeDraft {
  return {
    id: `edge-${index}-${fromNodeID}-${toNodeID}`,
    from_node_id: fromNodeID,
    to_node_id: toNodeID,
  };
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

function asRecord(value: unknown): Record<string, unknown> {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return {};
  }
  return value as Record<string, unknown>;
}
