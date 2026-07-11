import { validateOrchestrationMembers } from '@/lib/orchestration-editor/validationMembers';
import { validateOrchestrationNode } from '@/lib/orchestration-editor/validationNodes';
import type {
  WorkflowCanvasDraft,
  WorkflowCanvasEdgeDraft,
  WorkflowCanvasNodeDraft,
} from '@/lib/workflow-editor/types';

const GROUP_REQUIRED_ERROR = 'orchestration requires at least 1 group node';

interface EdgeEndpoints {
  source: WorkflowCanvasNodeDraft;
  target: WorkflowCanvasNodeDraft;
}

interface EntryExitCounts {
  entryCount: number;
  exitCount: number;
}

interface ControlDegree {
  inDegree: number;
  outDegree: number;
}

interface OrchestrationGraphValidationState {
  errors: string[];
  nodeMap: Map<string, WorkflowCanvasNodeDraft>;
  controlIn: Map<string, number>;
  controlOut: Map<string, number>;
  controlNext: Map<string, string>;
  groupMembers: Map<string, string[]>;
}

interface ControlFlowWalk {
  visited: Set<string>;
  hasCycle: boolean;
}

export function validateOrchestrationGraph(draft: WorkflowCanvasDraft): string[] {
  const state = createGraphValidationState();
  for (const node of draft.nodes) {
    registerOrchestrationNode(node, state);
  }
  for (const edge of draft.edges) {
    validateOrchestrationEdge(edge, state);
  }
  validateOrchestrationDegrees(draft, state);
  validateOrchestrationConnectivity(draft, state);
  state.errors.push(...validateOrchestrationMembers(groupNodes(draft.nodes), state.groupMembers));
  return state.errors;
}

function createGraphValidationState(): OrchestrationGraphValidationState {
  return {
    errors: [],
    nodeMap: new Map(),
    controlIn: new Map(),
    controlOut: new Map(),
    controlNext: new Map(),
    groupMembers: new Map(),
  };
}

function registerOrchestrationNode(
  node: WorkflowCanvasNodeDraft,
  state: OrchestrationGraphValidationState,
): void {
  if (!node.id.trim()) {
    state.errors.push('orchestration node id is required');
  } else if (state.nodeMap.has(node.id)) {
    state.errors.push(`duplicate orchestration node id "${node.id}"`);
  }
  state.nodeMap.set(node.id, node);
  validateOrchestrationNode(node, state.errors);
}

function validateOrchestrationEdge(
  edge: WorkflowCanvasEdgeDraft,
  state: OrchestrationGraphValidationState,
): void {
  const endpoints = resolveEdgeEndpoints(edge, state);
  if (!endpoints) {
    return;
  }
  if (edge.kind === 'member') {
    validateMemberEdge(endpoints, state);
    return;
  }
  if (edge.kind === 'control') {
    validateControlEdge(edge, endpoints, state);
    return;
  }
  state.errors.push(`unsupported orchestration edge kind "${String(edge.kind)}"`);
}

function resolveEdgeEndpoints(
  edge: WorkflowCanvasEdgeDraft,
  state: OrchestrationGraphValidationState,
): EdgeEndpoints | undefined {
  const source = state.nodeMap.get(edge.from_node_id);
  const target = state.nodeMap.get(edge.to_node_id);
  if (!source || !target) {
    state.errors.push('orchestration edge references unknown node id');
    return undefined;
  }
  return { source, target };
}

function validateMemberEdge(
  endpoints: EdgeEndpoints,
  state: OrchestrationGraphValidationState,
): void {
  if (endpoints.source.type !== 'agent' || endpoints.target.type !== 'group') {
    state.errors.push('orchestration member edge must be agent -> group');
    return;
  }
  appendGroupMember(state.groupMembers, endpoints.target.id, endpoints.source.id);
}

function appendGroupMember(groupMembers: Map<string, string[]>, groupID: string, memberID: string): void {
  groupMembers.set(groupID, [...(groupMembers.get(groupID) ?? []), memberID]);
}

function validateControlEdge(
  edge: WorkflowCanvasEdgeDraft,
  endpoints: EdgeEndpoints,
  state: OrchestrationGraphValidationState,
): void {
  if (endpoints.source.type !== 'group' || endpoints.target.type !== 'group') {
    state.errors.push(`orchestration control edge "${edge.from_node_id}" -> "${edge.to_node_id}" is invalid`);
    return;
  }
  state.controlNext.set(endpoints.source.id, endpoints.target.id);
  incrementMapCount(state.controlOut, endpoints.source.id);
  incrementMapCount(state.controlIn, endpoints.target.id);
}

function incrementMapCount(counts: Map<string, number>, key: string): void {
  counts.set(key, (counts.get(key) ?? 0) + 1);
}

function validateOrchestrationDegrees(
  draft: WorkflowCanvasDraft,
  state: OrchestrationGraphValidationState,
): void {
  const groups = groupNodes(draft.nodes);
  if (!shouldValidateGroupDegrees(draft, groups, state.errors)) {
    return;
  }
  const counts = countEntryExitGroups(groups, state);
  validateEntryExitCounts(counts, state.errors);
}

function shouldValidateGroupDegrees(
  draft: WorkflowCanvasDraft,
  groups: WorkflowCanvasNodeDraft[],
  errors: string[],
): boolean {
  if (groups.length > 0) {
    return true;
  }
  if (hasVisibleOrchestrationConfig(draft)) {
    errors.push(GROUP_REQUIRED_ERROR);
  }
  return false;
}

function hasVisibleOrchestrationConfig(draft: WorkflowCanvasDraft): boolean {
  return draft.nodes.some((node) => node.type === 'agent') || draft.edges.length > 0;
}

function countEntryExitGroups(
  groups: WorkflowCanvasNodeDraft[],
  state: OrchestrationGraphValidationState,
): EntryExitCounts {
  const counts = { entryCount: 0, exitCount: 0 };
  for (const group of groups) {
    const degree = groupControlDegree(group.id, state);
    validateGroupControlDegree(group.id, degree, state.errors);
    if (degree.inDegree === 0) {
      counts.entryCount += 1;
    }
    if (degree.outDegree === 0) {
      counts.exitCount += 1;
    }
  }
  return counts;
}

function groupControlDegree(groupID: string, state: OrchestrationGraphValidationState): ControlDegree {
  return {
    inDegree: state.controlIn.get(groupID) ?? 0,
    outDegree: state.controlOut.get(groupID) ?? 0,
  };
}

function validateGroupControlDegree(groupID: string, degree: ControlDegree, errors: string[]): void {
  if (degree.inDegree > 1 || degree.outDegree > 1) {
    errors.push(`orchestration group node "${groupID}" must have in<=1 and out<=1`);
  }
}

function validateEntryExitCounts(counts: EntryExitCounts, errors: string[]): void {
  if (counts.entryCount !== 1) {
    errors.push('orchestration requires exactly 1 entry group');
  }
  if (counts.exitCount !== 1) {
    errors.push('orchestration requires exactly 1 exit group');
  }
}

function validateOrchestrationConnectivity(
  draft: WorkflowCanvasDraft,
  state: OrchestrationGraphValidationState,
): void {
  const groups = groupNodes(draft.nodes);
  if (groups.length === 0) {
    return;
  }
  const startNode = findEntryGroup(groups, state.controlIn);
  if (!startNode) {
    return;
  }
  const walk = walkControlFlow(startNode.id, state.controlNext);
  if (walk.hasCycle) {
    state.errors.push('orchestration control flow contains a cycle');
  }
  validateControlFlowCoverage(groups, walk.visited, state.errors);
}

function findEntryGroup(
  groups: WorkflowCanvasNodeDraft[],
  controlIn: Map<string, number>,
): WorkflowCanvasNodeDraft | undefined {
  return groups.find((node) => (controlIn.get(node.id) ?? 0) === 0);
}

function walkControlFlow(startID: string, controlNext: Map<string, string>): ControlFlowWalk {
  const visited = new Set<string>();
  let currentID = startID;
  while (currentID) {
    if (visited.has(currentID)) {
      return { visited, hasCycle: true };
    }
    visited.add(currentID);
    currentID = controlNext.get(currentID) ?? '';
  }
  return { visited, hasCycle: false };
}

function validateControlFlowCoverage(
  groups: WorkflowCanvasNodeDraft[],
  visited: Set<string>,
  errors: string[],
): void {
  for (const node of groups) {
    if (!visited.has(node.id)) {
      errors.push(`orchestration node "${node.id}" is disconnected from control flow`);
    }
  }
}

function groupNodes(nodes: WorkflowCanvasNodeDraft[]): WorkflowCanvasNodeDraft[] {
  return nodes.filter((node) => node.type === 'group');
}
