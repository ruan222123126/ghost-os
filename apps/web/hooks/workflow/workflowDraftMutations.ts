import {
  addNode,
  createWorkflowLoopNodeIDs,
  createWorkflowNodeID,
  duplicateNode,
} from '@/lib/workflow-editor';
import type {
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowCanvasPosition,
  WorkflowNodeType,
} from '@/lib/workflow-editor';

interface AddWorkflowNodeOptions {
  state: WorkflowCanvasDraft;
  type: WorkflowNodeType;
  position: WorkflowCanvasPosition;
  source?: Partial<WorkflowCanvasNodeDraft>;
}

export function addWorkflowNodeWithGeneratedID(options: AddWorkflowNodeOptions): WorkflowCanvasDraft {
  const { state, type, position, source } = options;
  if (type === 'loop') {
    return addNode(state, {
      type,
      ...createWorkflowLoopNodeIDs(state.nodes.length),
      position,
    });
  }
  return addNode(state, {
    type,
    id: createWorkflowNodeID(type, state.nodes.length),
    position,
    source,
  });
}

export function duplicateWorkflowNodeWithGeneratedID(
  state: WorkflowCanvasDraft,
  nodeID: string,
): WorkflowCanvasDraft {
  const source = state.nodes.find((node) => node.id === nodeID);
  if (!source) {
    return state;
  }
  if (source.type === 'loop') {
    return duplicateNode(state, {
      nodeID,
      ...createWorkflowLoopNodeIDs(state.nodes.length),
    });
  }
  return duplicateNode(state, {
    nodeID,
    newNodeId: createWorkflowNodeID(source.type, state.nodes.length),
  });
}
