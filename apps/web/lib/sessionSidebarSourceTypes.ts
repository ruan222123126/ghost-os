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

export interface SessionSourceOwner {
  id: string;
  name: string;
}

export const SOURCE_PARTITION_IDS: Record<SessionSourceKind, string> = {
  workflow: '__source_workflow__',
  orchestration: '__source_orchestration__',
  loop: '__source_loop__',
  task: '__source_task__',
};

export const SOURCE_CHILD_PARTITION_SEPARATOR = '::';

export const SOURCE_ORDER: readonly SessionSourceKind[] = [
  'workflow',
  'orchestration',
  'loop',
  'task',
];

export const SOURCE_PRIORITY: Record<SessionSourceKind, number> = {
  workflow: 4,
  orchestration: 3,
  loop: 2,
  task: 1,
};
