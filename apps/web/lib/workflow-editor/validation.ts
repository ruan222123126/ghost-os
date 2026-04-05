import { END_NODE_TYPE, INPUT_NAME_PATTERN, START_NODE_TYPE } from '@/lib/workflow-editor/constants';
import type {
  WorkflowCanvasDraft,
  WorkflowCanvasNodeDraft,
  WorkflowValidationResult,
} from '@/lib/workflow-editor/types';

const SUPPORTED_INPUT_TYPES = ['string', 'number', 'boolean', 'object', 'array'] as const;

export function validateWorkflowDraft(draft: WorkflowCanvasDraft): WorkflowValidationResult {
  const errors: string[] = [];
  const nodeMap = buildNodeMap(draft.nodes, errors);

  validateNodeKinds(draft.nodes, errors);
  validateNodePayloads(draft.nodes, errors);
  validateGraphRules(draft, nodeMap, errors);

  return {
    valid: errors.length === 0,
    errors,
  };
}

function buildNodeMap(nodes: WorkflowCanvasNodeDraft[], errors: string[]): Map<string, WorkflowCanvasNodeDraft> {
  const nodeMap = new Map<string, WorkflowCanvasNodeDraft>();
  for (const node of nodes) {
    if (node.id.trim().length === 0) {
      errors.push('workflow node id is required');
      continue;
    }
    if (nodeMap.has(node.id)) {
      errors.push(`duplicate workflow node id "${node.id}"`);
      continue;
    }
    nodeMap.set(node.id, node);
  }
  return nodeMap;
}

function validateNodeKinds(nodes: WorkflowCanvasNodeDraft[], errors: string[]): void {
  const startCount = nodes.filter((node) => node.type === START_NODE_TYPE).length;
  const endCount = nodes.filter((node) => node.type === END_NODE_TYPE).length;

  if (startCount !== 1) {
    errors.push('workflow requires exactly 1 start node');
  }
  if (endCount !== 1) {
    errors.push('workflow requires exactly 1 end node');
  }
}

function validateNodePayloads(nodes: WorkflowCanvasNodeDraft[], errors: string[]): void {
  for (const node of nodes) {
    validateNodePayload(node, errors);
  }
}

function validateNodePayload(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  switch (node.type) {
    case 'start':
      if (node.tool || node.llm || node.agent) {
        errors.push(`workflow node "${node.id}" payload does not match type "start"`);
      }
      validateStartInputs(node, errors);
      return;
    case 'tool':
      if (!node.tool || node.start || node.llm || node.agent) {
        errors.push(`workflow node "${node.id}" payload does not match type "tool"`);
        return;
      }
      if (node.tool.tool_name.trim().length === 0) {
        errors.push(`workflow tool node "${node.id}" requires tool_name`);
      }
      return;
    case 'llm':
      if (!node.llm || node.start || node.tool || node.agent) {
        errors.push(`workflow node "${node.id}" payload does not match type "llm"`);
        return;
      }
      if (node.llm.prompt.trim().length === 0) {
        errors.push(`workflow llm node "${node.id}" requires prompt`);
      }
      return;
    case 'agent':
      if (!node.agent || node.start || node.tool || node.llm) {
        errors.push(`workflow node "${node.id}" payload does not match type "agent"`);
        return;
      }
      if (node.agent.message.trim().length === 0) {
        errors.push(`workflow agent node "${node.id}" requires message`);
      }
      return;
    case 'end':
      if (node.start || node.tool || node.llm || node.agent) {
        errors.push(`workflow node "${node.id}" payload does not match type "end"`);
      }
      return;
    default:
      errors.push(`unsupported workflow node type "${String(node.type)}"`);
  }
}

function validateStartInputs(node: WorkflowCanvasNodeDraft, errors: string[]): void {
  const inputs = node.start?.inputs ?? [];
  const names = new Set<string>();

  for (let index = 0; index < inputs.length; index += 1) {
    const input = inputs[index];
    const context = `workflow start node "${node.id}" input[${index}]`;

    if (!INPUT_NAME_PATTERN.test(input.name)) {
      errors.push(`${context} name "${input.name}" must match ^[A-Za-z_][A-Za-z0-9_]*$`);
    }
    if (names.has(input.name)) {
      errors.push(`${context} duplicate name "${input.name}"`);
    }
    names.add(input.name);

    if (!isSupportedInputType(input.type)) {
      errors.push(`${context} uses unsupported type "${input.type}"`);
      continue;
    }

    if (!Object.prototype.hasOwnProperty.call(input, 'default')) {
      continue;
    }
    if (!inputDefaultMatchesType(input.type, input.default)) {
      errors.push(`${context} default must match declared type "${input.type}"`);
    }
  }
}

function isSupportedInputType(inputType: string): inputType is (typeof SUPPORTED_INPUT_TYPES)[number] {
  return SUPPORTED_INPUT_TYPES.includes(inputType as (typeof SUPPORTED_INPUT_TYPES)[number]);
}

function inputDefaultMatchesType(inputType: string, value: unknown): boolean {
  switch (inputType) {
    case 'string':
      return typeof value === 'string';
    case 'number':
      return typeof value === 'number' && Number.isFinite(value);
    case 'boolean':
      return typeof value === 'boolean';
    case 'object':
      return typeof value === 'object' && value !== null && !Array.isArray(value);
    case 'array':
      return Array.isArray(value);
    default:
      return false;
  }
}

function validateGraphRules(
  draft: WorkflowCanvasDraft,
  nodeMap: Map<string, WorkflowCanvasNodeDraft>,
  errors: string[],
): void {
  if (draft.edges.length !== draft.nodes.length - 1) {
    errors.push('workflow requires exactly len(nodes)-1 edges');
  }

  const indegree = new Map<string, number>();
  const outdegree = new Map<string, number>();
  const nextByNode = new Map<string, string>();

  for (const edge of draft.edges) {
    validateEdge(edge.from_node_id, edge.to_node_id, nodeMap, nextByNode, indegree, outdegree, errors);
  }

  validateDegreeRules(draft.nodes, indegree, outdegree, errors);
  validateConnectivity(draft.nodes, nextByNode, errors);
}

function validateEdge(
  fromNodeID: string,
  toNodeID: string,
  nodeMap: Map<string, WorkflowCanvasNodeDraft>,
  nextByNode: Map<string, string>,
  indegree: Map<string, number>,
  outdegree: Map<string, number>,
  errors: string[],
): void {
  if (fromNodeID.length === 0 || toNodeID.length === 0) {
    errors.push('workflow edge endpoints are required');
    return;
  }
  if (fromNodeID === toNodeID) {
    errors.push(`workflow does not allow self-loop edge "${fromNodeID}"`);
    return;
  }
  if (!nodeMap.has(fromNodeID)) {
    errors.push(`workflow edge references unknown from_node_id "${fromNodeID}"`);
    return;
  }
  if (!nodeMap.has(toNodeID)) {
    errors.push(`workflow edge references unknown to_node_id "${toNodeID}"`);
    return;
  }

  const existingNext = nextByNode.get(fromNodeID);
  if (existingNext && existingNext !== toNodeID) {
    errors.push('workflow branching is not supported');
  }

  nextByNode.set(fromNodeID, toNodeID);
  outdegree.set(fromNodeID, (outdegree.get(fromNodeID) ?? 0) + 1);
  indegree.set(toNodeID, (indegree.get(toNodeID) ?? 0) + 1);
}

function validateDegreeRules(
  nodes: WorkflowCanvasNodeDraft[],
  indegree: Map<string, number>,
  outdegree: Map<string, number>,
  errors: string[],
): void {
  for (const node of nodes) {
    const inDegree = indegree.get(node.id) ?? 0;
    const outDegree = outdegree.get(node.id) ?? 0;

    if (node.type === START_NODE_TYPE && (inDegree !== 0 || outDegree !== 1)) {
      errors.push('start node must have in=0 and out=1');
      continue;
    }
    if (node.type === END_NODE_TYPE && (inDegree !== 1 || outDegree !== 0)) {
      errors.push('end node must have in=1 and out=0');
      continue;
    }
    if (node.type !== START_NODE_TYPE && node.type !== END_NODE_TYPE && (inDegree !== 1 || outDegree !== 1)) {
      errors.push(`workflow node "${node.id}" must have in=1 and out=1`);
    }
  }
}

function validateConnectivity(
  nodes: WorkflowCanvasNodeDraft[],
  nextByNode: Map<string, string>,
  errors: string[],
): void {
  const startNode = nodes.find((node) => node.type === START_NODE_TYPE);
  const endNode = nodes.find((node) => node.type === END_NODE_TYPE);
  if (!startNode || !endNode) {
    return;
  }

  const seen = new Set<string>();
  let current = startNode.id;

  while (true) {
    if (seen.has(current)) {
      errors.push('workflow must not contain cycles');
      return;
    }
    seen.add(current);

    const next = nextByNode.get(current);
    if (!next) {
      break;
    }
    current = next;
  }

  if (current !== endNode.id) {
    errors.push('workflow does not reach end node');
  }
  if (seen.size !== nodes.length) {
    errors.push('workflow must be fully connected');
  }
}
