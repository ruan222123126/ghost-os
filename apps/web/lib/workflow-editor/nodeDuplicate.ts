import { cloneWorkflowTaskRuntimeOverrides } from '@/lib/workflow-editor/agentRuntime';
import { LOOP_ROLE_END, LOOP_ROLE_START } from '@/lib/workflow-editor/constants';
import {
  assertDistinctWorkflowNodeIDs,
  assertWorkflowLoopID,
  assertWorkflowNodeID,
} from '@/lib/workflow-editor/nodeIDs';
import { normalizeScreenControlComposerAction } from '@/lib/workflow-editor/screenControlComposer';
import type {
  ScreenControlComposerStep,
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowCanvasPosition,
} from '@/lib/workflow-editor/types';

const DUPLICATE_NODE_OFFSET_X = 36;
const DUPLICATE_NODE_OFFSET_Y = 36;

export interface DuplicateNodeOptions {
  nodeID: string;
  newNodeId?: string;
  loopId?: string;
  startNodeId?: string;
  endNodeId?: string;
}

interface LoopNodePair {
  startNode: WorkflowCanvasNodeDraft;
  endNode: WorkflowCanvasNodeDraft;
}

export function duplicateWorkflowNode(draft: WorkflowCanvasDraft, options: DuplicateNodeOptions): WorkflowCanvasDraft {
  const source = draft.nodes.find((node) => node.id === options.nodeID);
  if (!source) {
    return draft;
  }
  if (source.type === 'loop') {
    return duplicateLoopNodePair(draft, source, options);
  }
  return duplicateSingleNode(draft, source, options.newNodeId);
}

function duplicateSingleNode(
  draft: WorkflowCanvasDraft,
  sourceNode: WorkflowCanvasNodeDraft,
  newNodeId?: string,
): WorkflowCanvasDraft {
  const nodeID = assertNewNodeIDAvailable(draft, newNodeId, 'duplicate workflow node id');
  const copiedNode = cloneNodeForDuplicate(sourceNode, {
    id: nodeID,
    position: offsetPosition(sourceNode.position),
  });
  return {
    ...draft,
    nodes: [...draft.nodes, copiedNode],
    selectedNodeId: copiedNode.id,
  };
}

function duplicateLoopNodePair(
  draft: WorkflowCanvasDraft,
  sourceNode: WorkflowCanvasNodeDraft,
  options: DuplicateNodeOptions,
): WorkflowCanvasDraft {
  const loopPair = findLoopNodePair(draft.nodes, sourceNode.loop?.loop_id);
  if (!loopPair) {
    return draft;
  }

  const loopIDs = assertDuplicateLoopIDsAvailable(draft, options);
  const copiedPair = cloneLoopPair(loopPair, loopIDs);
  const selectedNodeId = sourceNode.loop?.role === LOOP_ROLE_END
    ? copiedPair.endNode.id
    : copiedPair.startNode.id;

  return {
    ...draft,
    nodes: [...draft.nodes, copiedPair.startNode, copiedPair.endNode],
    selectedNodeId,
  };
}

function findLoopNodePair(
  nodes: WorkflowCanvasNodeDraft[],
  loopID?: string,
): LoopNodePair | undefined {
  const normalizedLoopID = (loopID ?? '').trim();
  if (!normalizedLoopID) {
    return undefined;
  }

  const startNode = findLoopRoleNode(nodes, normalizedLoopID, LOOP_ROLE_START);
  const endNode = findLoopRoleNode(nodes, normalizedLoopID, LOOP_ROLE_END);
  if (!startNode || !endNode) {
    return undefined;
  }
  return { startNode, endNode };
}

function findLoopRoleNode(
  nodes: WorkflowCanvasNodeDraft[],
  loopID: string,
  role: typeof LOOP_ROLE_START | typeof LOOP_ROLE_END,
): WorkflowCanvasNodeDraft | undefined {
  return nodes.find((node) => isLoopRoleNode(node, loopID, role));
}

function isLoopRoleNode(
  node: WorkflowCanvasNodeDraft,
  loopID: string,
  role: typeof LOOP_ROLE_START | typeof LOOP_ROLE_END,
): boolean {
  return node.type === 'loop' && node.loop?.loop_id === loopID && node.loop.role === role;
}

function cloneLoopPair(loopPair: LoopNodePair, loopIDs: RequiredLoopIDs): LoopNodePair {
  const startNode = cloneNodeForDuplicate(loopPair.startNode, {
    id: loopIDs.startNodeId,
    position: offsetPosition(loopPair.startNode.position),
    loop: {
      ...loopPair.startNode.loop,
      role: LOOP_ROLE_START,
      loop_id: loopIDs.loopId,
    },
  });
  const endNode = cloneNodeForDuplicate(loopPair.endNode, {
    id: loopIDs.endNodeId,
    position: offsetPosition(loopPair.endNode.position),
    loop: {
      ...loopPair.endNode.loop,
      role: LOOP_ROLE_END,
      loop_id: loopIDs.loopId,
    },
  });
  return { startNode, endNode };
}

interface RequiredLoopIDs {
  loopId: string;
  startNodeId: string;
  endNodeId: string;
}

function assertDuplicateLoopIDsAvailable(
  draft: WorkflowCanvasDraft,
  options: DuplicateNodeOptions,
): RequiredLoopIDs {
  const loopId = assertWorkflowLoopID(options.loopId ?? '');
  const startNodeId = assertNewNodeIDAvailable(draft, options.startNodeId, 'duplicate loop start node id');
  const endNodeId = assertNewNodeIDAvailable(draft, options.endNodeId, 'duplicate loop end node id');
  assertDistinctWorkflowNodeIDs([startNodeId, endNodeId]);
  if (draft.nodes.some((node) => node.type === 'loop' && node.loop?.loop_id === loopId)) {
    throw new Error(`workflow loop id already exists: ${loopId}`);
  }
  return { loopId, startNodeId, endNodeId };
}

function assertNewNodeIDAvailable(
  draft: WorkflowCanvasDraft,
  nodeID: string | undefined,
  label: string,
): string {
  const nextNodeID = assertWorkflowNodeID(nodeID ?? '', label);
  if (draft.nodes.some((node) => node.id === nextNodeID)) {
    throw new Error(`workflow node id already exists: ${nextNodeID}`);
  }
  return nextNodeID;
}

function cloneNodeForDuplicate(
  node: WorkflowCanvasNodeDraft,
  patch: Partial<WorkflowCanvasNodeDraft>,
): WorkflowCanvasNodeDraft {
  return {
    ...node,
    ...patch,
    ui: cloneNodeUI(node.ui),
    start: cloneStartPayload(node),
    tool: cloneToolPayload(node),
    llm: cloneLLMPayload(node),
    agent: cloneAgentPayload(node),
    if: cloneIfPayload(node),
    loop: patch.loop ?? cloneLoopPayload(node),
  };
}

function cloneStartPayload(node: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft['start'] {
  return node.start ? { inputs: cloneInputs(node.start.inputs) } : undefined;
}

function cloneToolPayload(node: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft['tool'] {
  if (!node.tool) {
    return undefined;
  }
  return {
    ...node.tool,
    arguments: cloneRecord(node.tool.arguments),
  };
}

function cloneLLMPayload(node: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft['llm'] {
  return node.llm ? { ...node.llm } : undefined;
}

function cloneAgentPayload(node: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft['agent'] {
  if (!node.agent) {
    return undefined;
  }
  return {
    ...node.agent,
    runtime_overrides: cloneWorkflowTaskRuntimeOverrides(node.agent.runtime_overrides),
  };
}

function cloneIfPayload(node: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft['if'] {
  return node.if ? { ...node.if } : undefined;
}

function cloneLoopPayload(node: WorkflowCanvasNodeDraft): WorkflowCanvasNodeDraft['loop'] {
  return node.loop ? { ...node.loop } : undefined;
}

function offsetPosition(position: WorkflowCanvasPosition): WorkflowCanvasPosition {
  return {
    x: position.x + DUPLICATE_NODE_OFFSET_X,
    y: position.y + DUPLICATE_NODE_OFFSET_Y,
  };
}

function cloneInputs(
  inputs?: NonNullable<WorkflowCanvasNodeDraft['start']>['inputs'],
): NonNullable<WorkflowCanvasNodeDraft['start']>['inputs'] {
  if (!inputs) {
    return undefined;
  }
  return inputs.map((item) => ({
    ...item,
    default: cloneUnknown(item.default),
  }));
}

function cloneRecord(input?: Record<string, unknown>): Record<string, unknown> | undefined {
  if (!input) {
    return undefined;
  }
  return cloneUnknown(input) as Record<string, unknown>;
}

function cloneUnknown<T>(input: T): T {
  if (input === null || input === undefined) {
    return input;
  }
  if (typeof structuredClone === 'function') {
    return structuredClone(input);
  }
  return JSON.parse(JSON.stringify(input)) as T;
}

function cloneNodeUI(ui: WorkflowCanvasNodeDraft['ui']): WorkflowCanvasNodeDraft['ui'] {
  return {
    ...ui,
    screenControlComposer: cloneScreenControlComposer(ui.screenControlComposer),
  };
}

function cloneScreenControlComposer(
  composer?: WorkflowCanvasNodeDraft['ui']['screenControlComposer'],
): WorkflowCanvasNodeDraft['ui']['screenControlComposer'] {
  if (!composer || composer.steps.length === 0) {
    return undefined;
  }
  return {
    steps: cloneComposerSteps(composer.steps),
  };
}

function cloneComposerSteps(steps: ScreenControlComposerStep[]): ScreenControlComposerStep[] {
  return steps.map((step) => ({
    action: normalizeScreenControlComposerAction(step.action),
    params: step.params ? { ...step.params } : undefined,
  }));
}
