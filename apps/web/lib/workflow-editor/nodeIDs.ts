import type { WorkflowNodeType } from '@/lib/workflow-editor/types';

const LOOP_NODE_ID_PREFIX = 'loop';
const LOOP_START_NODE_SUFFIX = 'start';
const LOOP_END_NODE_SUFFIX = 'end';

export interface WorkflowLoopNodeIDs {
  loopId: string;
  startNodeId: string;
  endNodeId: string;
}

export function createWorkflowNodeID(type: WorkflowNodeType, index: number): string {
  return `${type}-${Date.now()}-${index}`;
}

export function createWorkflowLoopNodeIDs(index: number): WorkflowLoopNodeIDs {
  const loopId = `${LOOP_NODE_ID_PREFIX}-${Date.now()}-${index}`;
  return {
    loopId,
    startNodeId: `${loopId}-${LOOP_START_NODE_SUFFIX}`,
    endNodeId: `${loopId}-${LOOP_END_NODE_SUFFIX}`,
  };
}

export function assertWorkflowNodeID(value: string, label: string): string {
  if (!value.trim()) {
    throw new Error(`${label} is required`);
  }
  return value;
}

export function assertWorkflowLoopID(value: string): string {
  if (!value.trim()) {
    throw new Error('workflow loop id is required');
  }
  return value;
}

export function assertDistinctWorkflowNodeIDs(values: string[]): void {
  const seen = new Set<string>();
  for (const value of values) {
    if (seen.has(value)) {
      throw new Error(`workflow node id must be unique: ${value}`);
    }
    seen.add(value);
  }
}
