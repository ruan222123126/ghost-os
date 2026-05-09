import { draftToOrchestrationUpdatePayload } from '@/lib/orchestration-editor/draft';
import type { WorkflowCanvasDraft, WorkflowUpdatePayload } from '@/lib/workflow-editor';
import type { AutosaveSnapshot } from '@/lib/workflow-editor';

const ORCHESTRATION_DRAFT_STORAGE_PREFIX = 'ghost.orchestration.canvas.v2';
export const ORCHESTRATION_VALIDATION_BLOCKED_TEXT = 'Validation failed, not saved';

export interface OrchestrationAutosaveSnapshotBuildResult {
  snapshot?: AutosaveSnapshot<WorkflowUpdatePayload>;
  errorMessage?: string;
}

export function buildOrchestrationAutosaveSnapshot(draft: WorkflowCanvasDraft): OrchestrationAutosaveSnapshotBuildResult {
  try {
    const payload = draftToOrchestrationUpdatePayload(draft);
    return {
      snapshot: {
        payload,
        fingerprint: JSON.stringify({ schedule: draft.schedule, nodes: draft.nodes, edges: draft.edges }),
      },
    };
  } catch (error) {
    return {
      errorMessage: error instanceof Error && error.message.trim() ? error.message.trim() : 'failed to build orchestration payload',
    };
  }
}

export function loadStoredOrchestrationDraft(taskID?: string): WorkflowCanvasDraft | undefined {
  if (typeof window === 'undefined') {
    return undefined;
  }
  const raw = window.localStorage.getItem(storageKey(taskID));
  if (!raw) {
    return undefined;
  }
  try {
    return JSON.parse(raw) as WorkflowCanvasDraft;
  } catch {
    return undefined;
  }
}

export function storeOrchestrationDraft(draft: WorkflowCanvasDraft, taskID?: string): void {
  if (typeof window === 'undefined') {
    return;
  }
  window.localStorage.setItem(storageKey(taskID), JSON.stringify(draft));
}

export function mergeOrchestrationDraftWithCachedCanvas(
  serverDraft: WorkflowCanvasDraft,
  cachedDraft?: WorkflowCanvasDraft,
): WorkflowCanvasDraft {
  if (!cachedDraft) {
    return serverDraft;
  }
  const nodeState = new Map(cachedDraft.nodes.map((node) => [node.id, { position: node.position, ui: node.ui }] as const));
  return {
    ...serverDraft,
    nodes: serverDraft.nodes.map((node) => {
      const cached = nodeState.get(node.id);
      if (!cached) {
        return node;
      }
      return { ...node, position: cached.position, ui: cached.ui };
    }),
  };
}

function storageKey(taskID?: string): string {
  return `${ORCHESTRATION_DRAFT_STORAGE_PREFIX}:${taskID ?? 'new'}`;
}
