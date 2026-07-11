import type { WebLocale } from '@/lib/i18n/locale';
import {
  addNode,
  createWorkflowNodeID,
  defaultOrchestrationAgentRuntimeOverrides,
  duplicateNode,
  moveNode,
  removeEdge,
  removeNode,
  updateNode,
} from '@/lib/workflow-editor';
import { DEFAULT_ORCHESTRATION_AGENT_TITLE_PREFIX } from '@/lib/workflow-editor/constants';
import type {
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowCanvasPosition,
  WorkflowNodeType,
} from '@/lib/workflow-editor/types';
import { connectOrchestrationNodes } from '@/lib/orchestration-editor/graph';
import { buildDefaultOrchestrationGroupNode } from '@/lib/orchestration-editor/groupDefaults';

type OrchestrationNodeType = Extract<WorkflowNodeType, 'agent' | 'group'>;

interface AddOrchestrationNodeOptions {
  type: WorkflowNodeType;
  position: WorkflowCanvasPosition;
  source?: Partial<WorkflowCanvasNodeDraft>;
}

interface DefaultOrchestrationNodeSourceOptions {
  draft: WorkflowCanvasDraft;
  type: WorkflowNodeType;
  toolNames: string[];
  locale: WebLocale;
}

export function buildDefaultOrchestrationNodeSource(
  options: DefaultOrchestrationNodeSourceOptions,
): Partial<WorkflowCanvasNodeDraft> {
  const type = assertOrchestrationNodeType(options.type);
  if (type === 'group') {
    return {
      group: buildDefaultOrchestrationGroupNode(countNodesByType(options.draft, type) + 1, options.locale),
    };
  }
  return {
    agent: {
      title: `${DEFAULT_ORCHESTRATION_AGENT_TITLE_PREFIX} ${countNodesByType(options.draft, type) + 1}`,
      message: '',
      runtime_overrides: defaultOrchestrationAgentRuntimeOverrides(options.toolNames),
    },
  };
}

export function addOrchestrationNode(
  draft: WorkflowCanvasDraft,
  options: AddOrchestrationNodeOptions,
): WorkflowCanvasDraft {
  const type = assertOrchestrationNodeType(options.type);
  return addNode(draft, {
    type,
    id: createWorkflowNodeID(type, draft.nodes.length),
    position: options.position,
    source: options.source,
  });
}

export function moveOrchestrationNode(
  draft: WorkflowCanvasDraft,
  nodeID: string,
  position: WorkflowCanvasPosition,
): WorkflowCanvasDraft {
  assertKnownOrchestrationNode(draft, nodeID);
  return moveNode(draft, nodeID, position);
}

export function duplicateOrchestrationNode(draft: WorkflowCanvasDraft, nodeID: string): WorkflowCanvasDraft {
  const source = assertKnownOrchestrationNode(draft, nodeID);
  return duplicateNode(draft, {
    nodeID,
    newNodeId: createWorkflowNodeID(source.type, draft.nodes.length),
  });
}

export function removeOrchestrationNode(draft: WorkflowCanvasDraft, nodeID: string): WorkflowCanvasDraft {
  assertKnownOrchestrationNode(draft, nodeID);
  return removeNode(draft, nodeID);
}

export function updateOrchestrationNode(
  draft: WorkflowCanvasDraft,
  node: WorkflowCanvasNodeDraft,
): WorkflowCanvasDraft {
  assertOrchestrationNodeType(node.type);
  return updateNode(draft, node);
}

export function connectOrchestrationDraftNodes(
  draft: WorkflowCanvasDraft,
  sourceNodeID: string,
  targetNodeID: string,
): WorkflowCanvasDraft {
  return connectOrchestrationNodes(draft, sourceNodeID, targetNodeID);
}

export function removeOrchestrationEdge(draft: WorkflowCanvasDraft, edgeID: string): WorkflowCanvasDraft {
  return removeEdge(draft, edgeID);
}

function assertKnownOrchestrationNode(
  draft: WorkflowCanvasDraft,
  nodeID: string,
): WorkflowCanvasNodeDraft {
  const node = draft.nodes.find((item) => item.id === nodeID);
  if (!node) {
    throw new Error(`orchestration node not found: ${nodeID}`);
  }
  assertOrchestrationNodeType(node.type);
  return node;
}

function assertOrchestrationNodeType(type: WorkflowNodeType): OrchestrationNodeType {
  if (type === 'agent' || type === 'group') {
    return type;
  }
  throw new Error(`unsupported orchestration node type "${type}"`);
}

function countNodesByType(draft: WorkflowCanvasDraft, type: OrchestrationNodeType): number {
  return draft.nodes.filter((node) => node.type === type).length;
}
