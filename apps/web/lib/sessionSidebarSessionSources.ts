import type { ChatCopy } from '@/lib/i18n/messages/chat';
import {
  UNCLASSIFIED_PARTITION_ID,
  type SessionPartitionView,
} from '@/lib/sessionSidebarPartitions';
import { collectHiddenSessionIDsFromRun } from '@/lib/sessionSidebarSourceHiddenSessions';
import { compareSessionsByRecentActivity } from '@/lib/sessionSidebarSessionSort';
import type { SessionMetadata, TaskRunLog } from '@/lib/types';

export type SessionSourceKind = 'workflow' | 'orchestration' | 'loop' | 'task';
export interface SessionSourceAssignment {
  kind: SessionSourceKind;
  ownerID: string;
  ownerName: string;
}

export type SessionSourceAssignments = Record<string, SessionSourceAssignment>;

export interface SessionSourceResolution {
  assignments: SessionSourceAssignments;
  hiddenSessionIDs: string[];
}

export interface CollectSessionSourceAssignmentsOptions {
  sourceNamesByID?: Record<string, string>;
  loopTaskIDs?: string[];
}

export const SOURCE_PARTITION_IDS: Record<SessionSourceKind, string> = {
  workflow: '__source_workflow__',
  orchestration: '__source_orchestration__',
  loop: '__source_loop__',
  task: '__source_task__',
};

const SOURCE_CHILD_PARTITION_SEPARATOR = '::';
const SOURCE_ORDER: readonly SessionSourceKind[] = ['workflow', 'orchestration', 'loop', 'task'];
const SOURCE_PRIORITY: Record<SessionSourceKind, number> = {
  workflow: 4,
  orchestration: 3,
  loop: 2,
  task: 1,
};

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
  const loopTaskIDs = new Set(options.loopTaskIDs ?? []);
  for (const run of runs) {
    const source = sourceKindFromRun(run, loopTaskIDs);
    if (!source) {
      continue;
    }
    const owner = sourceOwnerFromRun(run, options.sourceNamesByID ?? {});
    for (const sessionID of sessionIDsFromRun(run)) {
      assignSessionSource(assignments, sessionID, source, owner);
    }
    for (const sessionID of collectHiddenSessionIDsFromRun(run, source)) {
      addSessionID(hiddenSessionIDs, sessionID);
    }
  }
  return {
    assignments,
    hiddenSessionIDs: [...hiddenSessionIDs].filter((sessionID) => !assignments[sessionID]),
  };
}

export function mergeSessionSourcePartitionViews(input: {
  manualViews: SessionPartitionView[];
  sessions: SessionMetadata[];
  sourceAssignments: SessionSourceAssignments;
  hiddenSessionIDs?: string[];
  searchQuery: string;
  copy: ChatCopy;
}): SessionPartitionView[] {
  const sourceSessionIDs = new Set(Object.keys(input.sourceAssignments));
  const hiddenSessionIDs = new Set(input.hiddenSessionIDs ?? []);
  const hideEmptyManual = input.searchQuery.trim().length > 0;
  const manualViews = input.manualViews
    .map((view) => ({
      ...view,
      sessions: view.sessions.filter((session) => {
        return !sourceSessionIDs.has(session.id) && !hiddenSessionIDs.has(session.id);
      }),
    }))
    .filter((view) => !hideEmptyManual || view.sessions.length > 0);
  const unclassifiedView = manualViews.find((view) => view.id === UNCLASSIFIED_PARTITION_ID);
  const customViews = manualViews.filter((view) => view.id !== UNCLASSIFIED_PARTITION_ID);

  return [
    ...(unclassifiedView ? [unclassifiedView] : []),
    ...buildSystemViews(input),
    ...customViews,
  ];
}

function buildSystemViews(input: {
  sessions: SessionMetadata[];
  sourceAssignments: SessionSourceAssignments;
  searchQuery: string;
  copy: ChatCopy;
}): SessionPartitionView[] {
  const query = input.searchQuery.trim().toLowerCase();
  return SOURCE_ORDER
    .map((kind) => buildSystemView(input, kind, query))
    .filter((view) => view.sessions.length > 0);
}

function buildSystemView(
  input: {
    sessions: SessionMetadata[];
    sourceAssignments: SessionSourceAssignments;
    copy: ChatCopy;
  },
  kind: SessionSourceKind,
  query: string,
): SessionPartitionView {
  const sessions = systemSessionsForKind(input.sessions, input.sourceAssignments, kind, query);
  return {
    id: SOURCE_PARTITION_IDS[kind],
    name: systemPartitionName(kind, input.copy),
    readOnly: true,
    sessions,
    childPartitions: buildSystemChildPartitions(kind, sessions, input.sourceAssignments),
  };
}

function buildSystemChildPartitions(
  kind: SessionSourceKind,
  sessions: SessionMetadata[],
  assignments: SessionSourceAssignments,
): SessionPartitionView[] {
  const groups = new Map<string, { name: string; sessions: SessionMetadata[] }>();
  for (const session of sessions) {
    const assignment = assignments[session.id];
    if (!assignment || assignment.kind !== kind) {
      continue;
    }
    const ownerID = assignment.ownerID.trim() || kind;
    const group = groups.get(ownerID) ?? { name: assignment.ownerName.trim() || ownerID, sessions: [] };
    group.sessions.push(session);
    groups.set(ownerID, group);
  }
  return [...groups.entries()].map(([ownerID, group]) => ({
    id: sourceChildPartitionID(kind, ownerID),
    name: group.name,
    readOnly: true,
    sessions: group.sessions,
  }));
}

function systemSessionsForKind(
  sessions: SessionMetadata[],
  assignments: SessionSourceAssignments,
  kind: SessionSourceKind,
  query: string,
): SessionMetadata[] {
  return sessions
    .filter((session) => assignments[session.id]?.kind === kind)
    .filter((session) => session.id.toLowerCase().includes(query))
    .sort(compareSessionsByRecentActivity);
}

function systemPartitionName(kind: SessionSourceKind, copy: ChatCopy): string {
  if (kind === 'workflow') {
    return copy.sidebarPartitionWorkflow;
  }
  if (kind === 'orchestration') {
    return copy.sidebarPartitionOrchestration;
  }
  if (kind === 'loop') {
    return copy.sidebarPartitionLoop;
  }
  return copy.sidebarPartitionTask;
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
): { id: string; name: string } {
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
  sessionID: string,
  source: SessionSourceKind,
  owner: { id: string; name: string },
): void {
  const id = sessionID.trim();
  if (!id) {
    return;
  }
  const existing = assignments[id];
  if (!existing || SOURCE_PRIORITY[source] > SOURCE_PRIORITY[existing.kind]) {
    assignments[id] = {
      kind: source,
      ownerID: owner.id,
      ownerName: owner.name,
    };
  }
}

function sourceChildPartitionID(kind: SessionSourceKind, ownerID: string): string {
  return `${SOURCE_PARTITION_IDS[kind]}${SOURCE_CHILD_PARTITION_SEPARATOR}${encodeURIComponent(ownerID)}`;
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
