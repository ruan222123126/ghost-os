import {
  draftToWorkflowUpdatePayload,
} from '@/lib/workflow-editor';
import type {
  AutosaveSnapshot,
  WorkflowCanvasDraft,
  WorkflowUpdatePayload,
} from '@/lib/workflow-editor';

const WORKFLOW_DRAFT_STORAGE_PREFIX = 'ghost.workflow.draft.v1';
export const VALIDATION_BLOCKED_TEXT = 'Validation failed, not saved';

export interface AutosaveSnapshotBuildResult {
  snapshot?: AutosaveSnapshot<WorkflowUpdatePayload>;
  errorMessage?: string;
}

export function buildAutosaveSnapshot(draft: WorkflowCanvasDraft): AutosaveSnapshotBuildResult {
  try {
    const payload = draftToWorkflowUpdatePayload(draft);
    return {
      snapshot: {
        payload,
        fingerprint: buildDraftFingerprint(draft),
      },
    };
  } catch (error) {
    const message = error instanceof Error && error.message.trim().length > 0
      ? error.message.trim()
      : 'failed to build workflow payload';
    return {
      errorMessage: message,
    };
  }
}

export function resolveSaveBlockReason(
  validationErrors: string[],
  snapshotErrorMessage?: string,
): string | undefined {
  if (validationErrors.length > 0) {
    return validationErrors[0] ?? VALIDATION_BLOCKED_TEXT;
  }
  if (snapshotErrorMessage?.trim()) {
    return snapshotErrorMessage.trim();
  }
  return undefined;
}

export function loadStoredDraft(taskID?: string): WorkflowCanvasDraft | undefined {
  if (typeof window === 'undefined') {
    return undefined;
  }
  const raw = window.localStorage.getItem(storageKey(taskID));
  if (!raw) {
    return undefined;
  }
  try {
    return JSON.parse(raw) as WorkflowCanvasDraft;
  } catch (error) {
    console.error('failed to parse stored workflow draft', error);
    return undefined;
  }
}

export function storeDraft(draft: WorkflowCanvasDraft, taskID?: string): void {
  if (typeof window === 'undefined') {
    return;
  }
  try {
    window.localStorage.setItem(storageKey(taskID), JSON.stringify(draft));
  } catch (error) {
    console.error('failed to store workflow draft', error);
  }
}

export function migrateStoredDraft(fromTaskID: string | undefined, toTaskID: string): void {
  if (typeof window === 'undefined') {
    return;
  }
  const fromKey = storageKey(fromTaskID);
  const toKey = storageKey(toTaskID);
  const cached = window.localStorage.getItem(fromKey);
  if (!cached) {
    return;
  }
  try {
    window.localStorage.setItem(toKey, cached);
    window.localStorage.removeItem(fromKey);
  } catch (error) {
    console.error('failed to migrate workflow draft cache', error);
  }
}

export function mergeServerDraftWithCachedCanvas(
  serverDraft: WorkflowCanvasDraft,
  cachedDraft?: WorkflowCanvasDraft,
): WorkflowCanvasDraft {
  if (!cachedDraft) {
    return serverDraft;
  }

  const nodePositionByID = new Map(
    cachedDraft.nodes.map((node) => [node.id, { position: node.position, ui: node.ui }] as const),
  );
  const nodes = serverDraft.nodes.map((node) => {
    const cached = nodePositionByID.get(node.id);
    if (!cached) {
      return node;
    }
    return {
      ...node,
      position: cached.position,
      ui: cached.ui,
    };
  });

  return {
    ...serverDraft,
    nodes,
  };
}

function buildDraftFingerprint(draft: WorkflowCanvasDraft): string {
  return JSON.stringify({
    schedule: draft.schedule,
    nodes: draft.nodes.map((node) => ({
      id: node.id,
      type: node.type,
      position: node.position,
      ui: node.ui,
      start: node.start,
      tool: node.tool,
      llm: node.llm,
      agent: node.agent,
      if: node.if,
      loop: node.loop,
    })),
    edges: draft.edges,
  });
}

function storageKey(taskID?: string): string {
  return `${WORKFLOW_DRAFT_STORAGE_PREFIX}:${taskID ?? 'new'}`;
}
