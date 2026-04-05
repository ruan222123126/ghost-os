import type {
  Connection,
  Edge,
  EdgeChange,
  Node,
  NodeChange,
} from '@xyflow/react';
import type {
  WorkflowCanvasDraft,
  WorkflowCanvasEdgeDraft,
  WorkflowCanvasNodeDraft,
  WorkflowNodeType,
} from '@/lib/workflow-editor';

const NEW_NODE_X_GAP = 260;
const NEW_NODE_Y = 80;

export function toFlowNode(node: WorkflowCanvasNodeDraft, selected: boolean): Node {
  return {
    id: node.id,
    position: node.position,
    selected,
    data: { label: `${node.type.toUpperCase()}\n${node.id}` },
    style: {
      borderRadius: 10,
      border: '1px solid #525252',
      background: '#171717',
      color: '#fafafa',
      whiteSpace: 'pre',
    },
  };
}

export function toFlowEdge(edge: WorkflowCanvasEdgeDraft): Edge {
  return {
    id: edge.id,
    source: edge.from_node_id,
    target: edge.to_node_id,
    animated: false,
  };
}

export function applyNodeChanges(
  nodes: WorkflowCanvasNodeDraft[],
  changes: NodeChange[],
): WorkflowCanvasNodeDraft[] {
  const removedIDs = new Set<string>();
  for (const change of changes) {
    if (change.type === 'remove' && 'id' in change) {
      removedIDs.add(change.id);
    }
  }

  return nodes
    .filter((node) => !removedIDs.has(node.id))
    .map((node) => {
      let nextPosition: WorkflowCanvasNodeDraft['position'] | undefined;
      for (const change of changes) {
        if (change.type !== 'position' || !('id' in change) || change.id !== node.id || !change.position) {
          continue;
        }
        nextPosition = change.position;
        break;
      }
      if (!nextPosition) {
        return node;
      }

      return { ...node, position: nextPosition };
    });
}

export function applyEdgeChanges(
  edges: WorkflowCanvasEdgeDraft[],
  changes: EdgeChange[],
): WorkflowCanvasEdgeDraft[] {
  const removedIDs = new Set<string>();
  for (const change of changes) {
    if (change.type === 'remove' && 'id' in change) {
      removedIDs.add(change.id);
    }
  }

  return edges.filter((edge) => !removedIDs.has(edge.id));
}

export function connectNodes(draft: WorkflowCanvasDraft, connection: Connection): WorkflowCanvasDraft {
  if (!connection.source || !connection.target) {
    return draft;
  }
  const edge: WorkflowCanvasEdgeDraft = {
    id: `edge-${draft.edges.length + 1}-${connection.source}-${connection.target}`,
    from_node_id: connection.source,
    to_node_id: connection.target,
  };

  return {
    ...draft,
    edges: [...draft.edges, edge],
  };
}

export function addNode(draft: WorkflowCanvasDraft, type: WorkflowNodeType): WorkflowCanvasDraft {
  const index = draft.nodes.length;
  const node = createDraftNode(`${type}-${Date.now()}-${index}`, type, index);

  return {
    ...draft,
    nodes: [...draft.nodes, node],
    selectedNodeId: node.id,
  };
}

export function removeNode(draft: WorkflowCanvasDraft, nodeID: string): WorkflowCanvasDraft {
  return {
    ...draft,
    nodes: draft.nodes.filter((node) => node.id !== nodeID),
    edges: draft.edges.filter((edge) => edge.from_node_id !== nodeID && edge.to_node_id !== nodeID),
    selectedNodeId: draft.selectedNodeId === nodeID ? undefined : draft.selectedNodeId,
  };
}

export function updateNode(draft: WorkflowCanvasDraft, node: WorkflowCanvasNodeDraft): WorkflowCanvasDraft {
  const currentID = draft.selectedNodeId ?? node.id;
  const nodes = draft.nodes.map((item) => (item.id === currentID ? node : item));
  if (currentID === node.id) {
    return { ...draft, nodes };
  }

  return {
    ...draft,
    nodes,
    edges: draft.edges.map((edge) => ({
      ...edge,
      from_node_id: edge.from_node_id === currentID ? node.id : edge.from_node_id,
      to_node_id: edge.to_node_id === currentID ? node.id : edge.to_node_id,
    })),
    selectedNodeId: node.id,
  };
}

export function createDraftNode(
  id: string,
  type: WorkflowNodeType,
  index: number,
  source?: Partial<WorkflowCanvasNodeDraft>,
): WorkflowCanvasNodeDraft {
  return {
    id,
    type,
    position: source?.position ?? { x: index * NEW_NODE_X_GAP, y: NEW_NODE_Y },
    ui: { toolArgumentsMode: source?.ui?.toolArgumentsMode ?? 'kv' },
    start: type === 'start' ? source?.start ?? { inputs: [] } : undefined,
    tool: type === 'tool' ? source?.tool ?? { tool_name: '', arguments: {} } : undefined,
    llm: type === 'llm' ? source?.llm ?? { prompt: '', system_prompt: '' } : undefined,
    agent: type === 'agent' ? source?.agent ?? { message: '' } : undefined,
  };
}
