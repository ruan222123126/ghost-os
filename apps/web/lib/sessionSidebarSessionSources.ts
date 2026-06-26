import { collectHiddenSessionIDsFromRun } from '@/lib/sessionSidebarSourceHiddenSessions';
import {
  SOURCE_CHILD_PARTITION_SEPARATOR,
  SOURCE_ORDER,
  SOURCE_PARTITION_IDS,
  SOURCE_PRIORITY,
  type CollectSessionSourceAssignmentsOptions,
  type SessionSourceAssignments,
  type SessionSourceKind,
  type SessionSourceOwner,
  type SessionSourceResolution,
} from '@/lib/sessionSidebarSourceTypes';
import type {
  SessionSourceAssignment as SharedSessionSourceAssignment,
  TaskRunLog,
} from '@/lib/types';

export { mergeSessionSourcePartitionViews } from '@/lib/sessionSidebarSourcePartitionViews';
export { SOURCE_PARTITION_IDS } from '@/lib/sessionSidebarSourceTypes';
export type {
  CollectSessionSourceAssignmentsOptions,
  SessionSourceAssignment,
  SessionSourceAssignments,
  SessionSourceKind,
  SessionSourceResolution,
} from '@/lib/sessionSidebarSourceTypes';

interface SessionSourceCandidate {
  sessionID: string;
  source: SessionSourceKind;
  owner: SessionSourceOwner;
}

interface SessionSourceCollectionContext {
  assignments: SessionSourceAssignments;
  hiddenSessionIDs: Set<string>;
  loopTaskIDs: ReadonlySet<string>;
  sourceNamesByID: Record<string, string>;
}

export function sessionSourceAssignmentsFromPayload(
  assignments: Record<string, SharedSessionSourceAssignment>,
): SessionSourceAssignments {
  const normalized: SessionSourceAssignments = {};
  for (const [sessionID, assignment] of Object.entries(assignments)) {
    normalized[sessionID] = {
      kind: assignment.kind,
      ownerID: assignment.owner_id,
      ownerName: assignment.owner_name,
    };
  }
  return normalized;
}

export function isSystemSessionPartitionID(partitionID: string): boolean {
  const id = partitionID.trim();
  return SOURCE_ORDER.some((kind) => {
    const sourceID = SOURCE_PARTITION_IDS[kind];
    return id === sourceID || id.startsWith(`${sourceID}${SOURCE_CHILD_PARTITION_SEPARATOR}`);
  });
}

export function collectSessionSourceAssignments(
  runs: TaskRunLog[],
  options: CollectSessionSourceAssignmentsOptions = {},
): SessionSourceAssignments {
  return collectSessionSourceResolution(runs, options).assignments;
}

export function collectSessionSourceResolution(
  runs: TaskRunLog[],
  options: CollectSessionSourceAssignmentsOptions = {},
): SessionSourceResolution {
  const assignments: SessionSourceAssignments = {};
  const hiddenSessionIDs = new Set<string>();
  const context = {
    assignments,
    hiddenSessionIDs,
    loopTaskIDs: new Set(options.loopTaskIDs ?? []),
    sourceNamesByID: options.sourceNamesByID ?? {},
  };
  for (const run of runs) {
    collectSessionSourceRun(run, context);
  }
  return {
    assignments,
    hiddenSessionIDs: [...hiddenSessionIDs].filter((sessionID) => !assignments[sessionID]),
  };
}

function collectSessionSourceRun(
  run: TaskRunLog,
  context: SessionSourceCollectionContext,
): void {
  const source = sourceKindFromRun(run, context.loopTaskIDs);
  if (!source) {
    return;
  }
  const owner = sourceOwnerFromRun(run, context.sourceNamesByID);
  for (const sessionID of sessionIDsFromRun(run)) {
    assignSessionSource(context.assignments, { sessionID, source, owner });
  }
  for (const sessionID of collectHiddenSessionIDsFromRun(run, source)) {
    addSessionID(context.hiddenSessionIDs, sessionID);
  }
}

function sourceKindFromRun(
  run: TaskRunLog,
  loopTaskIDs: ReadonlySet<string>,
): SessionSourceKind | undefined {
  if (run.task_kind === 'workflow' || run.task_kind === 'orchestration') {
    return run.task_kind;
  }
  if (run.task_kind === 'agent_message' && !run.session_id_input?.trim()) {
    return loopTaskIDs.has(run.task_id.trim()) ? 'loop' : 'task';
  }
  return undefined;
}

function sourceOwnerFromRun(
  run: TaskRunLog,
  sourceNamesByID: Record<string, string>,
): SessionSourceOwner {
  const id = run.task_id.trim();
  const name = sourceNamesByID[id]?.trim() ?? '';
  return {
    id,
    name: name || id,
  };
}

function sessionIDsFromRun(run: TaskRunLog): string[] {
  const ids = new Set<string>();
  addSessionID(ids, run.session_id_output);
  return [...ids];
}

function assignSessionSource(
  assignments: SessionSourceAssignments,
  candidate: SessionSourceCandidate,
): void {
  const id = candidate.sessionID.trim();
  if (!id) {
    return;
  }
  const existing = assignments[id];
  if (!existing || SOURCE_PRIORITY[candidate.source] > SOURCE_PRIORITY[existing.kind]) {
    assignments[id] = {
      kind: candidate.source,
      ownerID: candidate.owner.id,
      ownerName: candidate.owner.name,
    };
  }
}

function addSessionID(ids: Set<string>, value: unknown): void {
  if (typeof value !== 'string') {
    return;
  }
  const id = value.trim();
  if (id) {
    ids.add(id);
  }
}
