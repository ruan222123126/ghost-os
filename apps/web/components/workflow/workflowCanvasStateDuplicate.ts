import type {
  ScreenControlComposerStep,
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowCanvasPosition,
  WorkflowNodeType,
} from '@/lib/workflow-editor';
import { normalizeScreenControlComposerAction } from '@/lib/workflow-editor';
import { LOOP_ROLE_END, LOOP_ROLE_START } from '@/lib/workflow-editor/constants';

const LOOP_NODE_ID_PREFIX = 'loop';
const LOOP_START_NODE_SUFFIX = 'start';
const LOOP_END_NODE_SUFFIX = 'end';
const DUPLICATE_NODE_OFFSET_X = 36;
const DUPLICATE_NODE_OFFSET_Y = 36;

export function duplicateWorkflowNode(draft: WorkflowCanvasDraft, nodeID: string): WorkflowCanvasDraft {
  const source = draft.nodes.find((node) => node.id === nodeID);
  if (!source) {
    return draft;
  }
  if (source.type === 'loop') {
    return duplicateLoopNodePair(draft, source);
  }
  return duplicateSingleNode(draft, source);
}

function duplicateSingleNode(
  draft: WorkflowCanvasDraft,
  sourceNode: WorkflowCanvasNodeDraft,
): WorkflowCanvasDraft {
  const copiedNode = cloneNodeForDuplicate(sourceNode, {
    id: buildNodeID(sourceNode.type, draft.nodes.length),
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
): WorkflowCanvasDraft {
  const loopPair = findLoopNodePair(draft.nodes, sourceNode.loop?.loop_id);
  if (!loopPair) {
    return draft;
  }

  const nextLoopID = `${LOOP_NODE_ID_PREFIX}-${Date.now()}-${draft.nodes.length}`;
  const copiedPair = cloneLoopPair(loopPair, nextLoopID);
  const selectedNodeId = sourceNode.loop?.role === LOOP_ROLE_END
    ? copiedPair.endNode.id
    : copiedPair.startNode.id;

  return {
    ...draft,
    nodes: [...draft.nodes, copiedPair.startNode, copiedPair.endNode],
    selectedNodeId,
  };
}

interface LoopNodePair {
  startNode: WorkflowCanvasNodeDraft;
  endNode: WorkflowCanvasNodeDraft;
}

function findLoopNodePair(
  nodes: WorkflowCanvasNodeDraft[],
  loopID?: string,
): LoopNodePair | undefined {
  const normalizedLoopID = (loopID ?? '').trim();
  if (!normalizedLoopID) {
    return undefined;
  }

  const startNode = nodes.find(
    (node) => node.type === 'loop'
      && node.loop?.loop_id === normalizedLoopID
      && node.loop?.role === LOOP_ROLE_START,
  );
  const endNode = nodes.find(
    (node) => node.type === 'loop'
      && node.loop?.loop_id === normalizedLoopID
      && node.loop?.role === LOOP_ROLE_END,
  );
  if (!startNode || !endNode) {
    return undefined;
  }
  return { startNode, endNode };
}

function cloneLoopPair(loopPair: LoopNodePair, nextLoopID: string): LoopNodePair {
  const startNode = cloneNodeForDuplicate(loopPair.startNode, {
    id: `${nextLoopID}-${LOOP_START_NODE_SUFFIX}`,
    position: offsetPosition(loopPair.startNode.position),
    loop: {
      ...loopPair.startNode.loop,
      role: LOOP_ROLE_START,
      loop_id: nextLoopID,
    },
  });
  const endNode = cloneNodeForDuplicate(loopPair.endNode, {
    id: `${nextLoopID}-${LOOP_END_NODE_SUFFIX}`,
    position: offsetPosition(loopPair.endNode.position),
    loop: {
      ...loopPair.endNode.loop,
      role: LOOP_ROLE_END,
      loop_id: nextLoopID,
    },
  });
  return { startNode, endNode };
}

function buildNodeID(type: WorkflowNodeType, index: number): string {
  return `${type}-${Date.now()}-${index}`;
}

function cloneNodeForDuplicate(
  node: WorkflowCanvasNodeDraft,
  patch: Partial<WorkflowCanvasNodeDraft>,
): WorkflowCanvasNodeDraft {
  return {
    ...node,
    ...patch,
    ui: cloneNodeUI(node.ui),
    start: node.start ? { inputs: cloneInputs(node.start.inputs) } : undefined,
    tool: node.tool
      ? {
        ...node.tool,
        arguments: cloneRecord(node.tool.arguments),
      }
      : undefined,
    llm: node.llm ? { ...node.llm } : undefined,
    agent: node.agent ? { ...node.agent } : undefined,
    if: node.if ? { ...node.if } : undefined,
    loop: patch.loop ?? (node.loop ? { ...node.loop } : undefined),
  };
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
