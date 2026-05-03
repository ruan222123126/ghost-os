import type {
  WorkflowCanvasDraft,
  WorkflowCanvasEdgeDraft,
  WorkflowCanvasNodeDraft,
  WorkflowVariableOption,
} from '@/lib/workflow-editor/types';
import type { WorkflowInputVariable } from '@/lib/types';

const ACTION_NODE_TYPES = new Set<WorkflowCanvasNodeDraft['type']>(['tool', 'llm', 'agent']);
const BUILTIN_VARIABLE_OPTIONS: WorkflowVariableOption[] = [
  createOption('${last}', 'last', 'builtin'),
  createOption('${last_output}', 'last output', 'builtin'),
  createOption('${last_node_id}', 'last node id', 'builtin'),
];

export function buildWorkflowVariableSuggestions(
  draft: WorkflowCanvasDraft,
  selectedNode: WorkflowCanvasNodeDraft,
): WorkflowVariableOption[] {
  return [
    ...buildStartVariableOptions(draft.nodes),
    ...BUILTIN_VARIABLE_OPTIONS,
    ...buildUpstreamNodeOptions(draft, selectedNode),
  ];
}

function buildStartVariableOptions(nodes: WorkflowCanvasNodeDraft[]): WorkflowVariableOption[] {
  const startNodes = nodes.filter((node) => node.type === 'start');
  if (startNodes.length !== 1) {
    return [];
  }

  return (startNodes[0].start?.inputs ?? [])
    .map((input) => buildStartVariableOption(input))
    .filter((option): option is WorkflowVariableOption => option !== undefined);
}

function buildStartVariableOption(input: WorkflowInputVariable): WorkflowVariableOption | undefined {
  const name = input.name.trim();
  if (name.length === 0) {
    return undefined;
  }

  return createOption(`\${inputs.${name}}`, name, 'start', [
    input.description,
    input.type,
  ]);
}

function buildUpstreamNodeOptions(
  draft: WorkflowCanvasDraft,
  selectedNode: WorkflowCanvasNodeDraft,
): WorkflowVariableOption[] {
  const upstreamIDs = collectReverseReachableNodeIDs(draft.edges, selectedNode.id);
  upstreamIDs.delete(selectedNode.id);

  return draft.nodes
    .filter((node) => upstreamIDs.has(node.id) && ACTION_NODE_TYPES.has(node.type))
    .flatMap((node) => [
      createOption(`\${outputs.${node.id}}`, `${node.id} outputs`, 'upstream', [node.id]),
      createOption(`\${nodes.${node.id}.output}`, `${node.id} output`, 'upstream', [node.id]),
      createOption(`\${nodes.${node.id}.text}`, `${node.id} text`, 'upstream', [node.id]),
      createOption(`\${nodes.${node.id}.status}`, `${node.id} status`, 'upstream', [node.id]),
    ]);
}

function collectReverseReachableNodeIDs(
  edges: WorkflowCanvasEdgeDraft[],
  targetNodeID: string,
): Set<string> {
  const incoming = new Map<string, string[]>();
  for (const edge of edges) {
    incoming.set(edge.to_node_id, [...(incoming.get(edge.to_node_id) ?? []), edge.from_node_id]);
  }

  const visited = new Set<string>();
  const pending = [...(incoming.get(targetNodeID) ?? [])];
  while (pending.length > 0) {
    const nodeID = pending.pop();
    if (!nodeID || visited.has(nodeID)) {
      continue;
    }
    visited.add(nodeID);
    for (const parentNodeID of incoming.get(nodeID) ?? []) {
      if (!visited.has(parentNodeID)) {
        pending.push(parentNodeID);
      }
    }
  }

  return visited;
}

function createOption(
  token: string,
  label: string,
  section: WorkflowVariableOption['section'],
  extraSearchText: Array<string | undefined> = [],
): WorkflowVariableOption {
  return {
    token,
    label,
    section,
    searchText: [tokenPath(token), label, ...extraSearchText]
      .filter((part): part is string => typeof part === 'string' && part.trim().length > 0)
      .join(' ')
      .toLowerCase(),
  };
}

function tokenPath(token: string): string {
  return token.replace(/^\$\{/, '').replace(/\}$/, '');
}
