import type {
  OrchestrationDefinition,
  OrchestrationEdge,
  OrchestrationNode,
} from '@/lib/types';
import {
  NODE_X_GAP,
  NODE_Y_BASE,
} from '@/lib/workflow-editor/constants';

const GROUP_MEMBER_Y_GAP = 150;
const GROUP_LANE_Y = NODE_Y_BASE + 140;

export interface OrchestrationNodePosition {
  x: number;
  y: number;
}

interface MemberPositionOptions {
  positions: Map<string, OrchestrationNodePosition>;
  memberID: string;
  base: OrchestrationNodePosition;
  memberIndex: number;
}

export function buildOrchestrationNodePositionMap(
  definition: OrchestrationDefinition,
): Map<string, OrchestrationNodePosition> {
  const positions = new Map<string, OrchestrationNodePosition>();
  positionControlLane(positions, buildControlNextMap(definition.edges), findEntryGroupID(definition));
  positionGroupMembers(positions, definition);
  return positions;
}

export function fallbackOrchestrationNodePosition(nodeID: string): OrchestrationNodePosition {
  const seed = Math.max(nodeID.length, 1);
  return { x: seed * NODE_X_GAP, y: GROUP_LANE_Y };
}

function positionControlLane(
  positions: Map<string, OrchestrationNodePosition>,
  controlNext: Map<string, string>,
  entryGroupID: string,
): void {
  let currentID = entryGroupID;
  let index = 0;
  while (shouldPositionControlGroup(currentID, positions)) {
    positions.set(currentID, controlGroupPosition(index));
    currentID = controlNext.get(currentID) ?? '';
    index += 1;
  }
}

function shouldPositionControlGroup(
  groupID: string,
  positions: Map<string, OrchestrationNodePosition>,
): boolean {
  return groupID.length > 0 && !positions.has(groupID);
}

function controlGroupPosition(index: number): OrchestrationNodePosition {
  return { x: index * NODE_X_GAP, y: GROUP_LANE_Y };
}

function positionGroupMembers(
  positions: Map<string, OrchestrationNodePosition>,
  definition: OrchestrationDefinition,
): void {
  for (const group of groupNodes(definition.nodes)) {
    positionMembersForGroup(positions, group.id, definition.edges);
  }
}

function positionMembersForGroup(
  positions: Map<string, OrchestrationNodePosition>,
  groupID: string,
  edges: OrchestrationEdge[],
): void {
  const base = positions.get(groupID) ?? fallbackOrchestrationNodePosition(groupID);
  memberEdgesForGroup(edges, groupID).forEach((edge, memberIndex) => {
    positionMemberIfNeeded({
      positions,
      memberID: edge.from_node_id,
      base,
      memberIndex,
    });
  });
}

function positionMemberIfNeeded(options: MemberPositionOptions): void {
  const { positions, memberID, base, memberIndex } = options;
  if (positions.has(memberID)) {
    return;
  }
  positions.set(memberID, {
    x: base.x,
    y: base.y - GROUP_MEMBER_Y_GAP * (memberIndex + 1),
  });
}

function memberEdgesForGroup(edges: OrchestrationEdge[], groupID: string): OrchestrationEdge[] {
  return edges.filter((edge) => edge.kind === 'member' && edge.to_node_id === groupID);
}

function groupNodes(nodes: OrchestrationNode[]): OrchestrationNode[] {
  return nodes.filter((node) => node.type === 'group');
}

function buildControlNextMap(edges: OrchestrationEdge[]): Map<string, string> {
  const controlNext = new Map<string, string>();
  for (const edge of edges) {
    if (edge.kind === 'control') {
      controlNext.set(edge.from_node_id, edge.to_node_id);
    }
  }
  return controlNext;
}

function findEntryGroupID(definition: OrchestrationDefinition): string {
  const controlTargets = collectControlTargetGroupIDs(definition);
  return (
    findFirstGroupWithoutIncomingControl(definition.nodes, controlTargets)
    ?? findFirstGroupID(definition.nodes)
    ?? ''
  );
}

function collectControlTargetGroupIDs(definition: OrchestrationDefinition): Set<string> {
  const groupIDs = collectGroupIDs(definition.nodes);
  const controlTargets = new Set<string>();
  for (const edge of definition.edges) {
    if (edge.kind === 'control' && groupIDs.has(edge.to_node_id)) {
      controlTargets.add(edge.to_node_id);
    }
  }
  return controlTargets;
}

function collectGroupIDs(nodes: OrchestrationNode[]): Set<string> {
  const groupIDs = new Set<string>();
  for (const node of nodes) {
    if (node.type === 'group') {
      groupIDs.add(node.id);
    }
  }
  return groupIDs;
}

function findFirstGroupWithoutIncomingControl(
  nodes: OrchestrationNode[],
  controlTargets: Set<string>,
): string | undefined {
  for (const node of nodes) {
    if (node.type === 'group' && !controlTargets.has(node.id)) {
      return node.id;
    }
  }
  return undefined;
}

function findFirstGroupID(nodes: OrchestrationNode[]): string | undefined {
  return nodes.find((node) => node.type === 'group')?.id;
}
