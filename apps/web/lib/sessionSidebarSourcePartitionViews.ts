import type { ChatCopy } from '@/lib/i18n/messages/chat';
import {
  UNCLASSIFIED_PARTITION_ID,
  partitionNameMatches,
  type SessionPartitionView,
  type SessionSearchMatcher,
} from '@/lib/sessionSidebarPartitions';
import {
  SOURCE_CHILD_PARTITION_SEPARATOR,
  SOURCE_ORDER,
  SOURCE_PARTITION_IDS,
  type SessionSourceAssignments,
  type SessionSourceKind,
} from '@/lib/sessionSidebarSourceTypes';
import { compareSessionsByRecentActivity } from '@/lib/sessionSidebarSessionSort';
import type { SessionMetadata } from '@/lib/types';

interface MergeSessionSourcePartitionViewsInput {
  manualViews: SessionPartitionView[];
  sessions: SessionMetadata[];
  sourceAssignments: SessionSourceAssignments;
  hiddenSessionIDs?: string[];
  searchQuery: string;
  copy: ChatCopy;
  matchesSearch?: SessionSearchMatcher;
}

interface SystemViewContext {
  sessions: SessionMetadata[];
  sourceAssignments: SessionSourceAssignments;
  query: string;
  copy: ChatCopy;
  matchesSearch: SessionSearchMatcher;
}

interface SourceSessionGroup {
  ownerID: string;
  name: string;
  sessions: SessionMetadata[];
}

interface BuildSystemChildPartitionInput {
  context: SystemViewContext;
  kind: SessionSourceKind;
  group: SourceSessionGroup;
  parentMatches: boolean;
}

const SYSTEM_PARTITION_NAME_RESOLVERS: Record<
  SessionSourceKind,
  (copy: ChatCopy) => string
> = {
  workflow: (copy) => copy.sidebarPartitionWorkflow,
  orchestration: (copy) => copy.sidebarPartitionOrchestration,
  loop: (copy) => copy.sidebarPartitionLoop,
  task: (copy) => copy.sidebarPartitionTask,
};

export function mergeSessionSourcePartitionViews(
  input: MergeSessionSourcePartitionViewsInput,
): SessionPartitionView[] {
  const { unclassifiedView, customViews } = splitManualViews(filterManualViews(input));

  return [
    ...(unclassifiedView ? [unclassifiedView] : []),
    ...buildSystemViews(input),
    ...customViews,
  ];
}

function filterManualViews(
  input: MergeSessionSourcePartitionViewsInput,
): SessionPartitionView[] {
  const sourceSessionIDs = new Set(Object.keys(input.sourceAssignments));
  const hiddenSessionIDs = new Set(input.hiddenSessionIDs ?? []);
  const hideEmptyManual = input.searchQuery.trim().length > 0;

  return input.manualViews
    .map((view) => filterManualViewSessions(view, sourceSessionIDs, hiddenSessionIDs))
    .filter((view) => !hideEmptyManual || view.sessions.length > 0);
}

function filterManualViewSessions(
  view: SessionPartitionView,
  sourceSessionIDs: ReadonlySet<string>,
  hiddenSessionIDs: ReadonlySet<string>,
): SessionPartitionView {
  return {
    ...view,
    sessions: view.sessions.filter((session) => {
      return !sourceSessionIDs.has(session.id) && !hiddenSessionIDs.has(session.id);
    }),
  };
}

function splitManualViews(manualViews: SessionPartitionView[]): {
  unclassifiedView: SessionPartitionView | undefined;
  customViews: SessionPartitionView[];
} {
  return {
    unclassifiedView: manualViews.find((view) => view.id === UNCLASSIFIED_PARTITION_ID),
    customViews: manualViews.filter((view) => view.id !== UNCLASSIFIED_PARTITION_ID),
  };
}

function buildSystemViews(input: {
  sessions: SessionMetadata[];
  sourceAssignments: SessionSourceAssignments;
  searchQuery: string;
  copy: ChatCopy;
  matchesSearch?: SessionSearchMatcher;
}): SessionPartitionView[] {
  const context = createSystemViewContext(input);
  return SOURCE_ORDER
    .map((kind) => buildSystemView(context, kind))
    .filter((view) => view.sessions.length > 0);
}

function createSystemViewContext(input: {
  sessions: SessionMetadata[];
  sourceAssignments: SessionSourceAssignments;
  searchQuery: string;
  copy: ChatCopy;
  matchesSearch?: SessionSearchMatcher;
}): SystemViewContext {
  return {
    sessions: input.sessions,
    sourceAssignments: input.sourceAssignments,
    query: input.searchQuery.trim().toLowerCase(),
    copy: input.copy,
    matchesSearch: input.matchesSearch ?? defaultSessionSearchMatcher,
  };
}

function buildSystemView(
  context: SystemViewContext,
  kind: SessionSourceKind,
): SessionPartitionView {
  const name = systemPartitionName(kind, context.copy);
  const parentMatches = partitionNameMatches(name, context.query);
  const childPartitions = buildSystemChildPartitions(context, kind, parentMatches);
  const sessions = sortSystemSessions(childPartitions.flatMap((partition) => partition.sessions));
  return {
    id: SOURCE_PARTITION_IDS[kind],
    name,
    readOnly: true,
    sessions,
    childPartitions,
  };
}

function buildSystemChildPartitions(
  context: SystemViewContext,
  kind: SessionSourceKind,
  parentMatches: boolean,
): SessionPartitionView[] {
  return [...groupSourceSessions(context, kind).values()]
    .map((group) => buildSystemChildPartition({ context, kind, group, parentMatches }))
    .filter((partition) => partition.sessions.length > 0);
}

function groupSourceSessions(
  context: SystemViewContext,
  kind: SessionSourceKind,
): Map<string, SourceSessionGroup> {
  const groups = new Map<string, SourceSessionGroup>();
  for (const session of context.sessions) {
    const assignment = context.sourceAssignments[session.id];
    if (!assignment || assignment.kind !== kind) {
      continue;
    }
    sourceSessionGroup(groups, kind, assignment).sessions.push(session);
  }
  return groups;
}

function sourceSessionGroup(
  groups: Map<string, SourceSessionGroup>,
  kind: SessionSourceKind,
  assignment: SessionSourceAssignments[string],
): SourceSessionGroup {
  const ownerID = assignment.ownerID.trim() || kind;
  const existing = groups.get(ownerID);
  if (existing) {
    return existing;
  }

  const group = {
    ownerID,
    name: assignment.ownerName.trim() || ownerID,
    sessions: [],
  };
  groups.set(ownerID, group);
  return group;
}

function buildSystemChildPartition(input: BuildSystemChildPartitionInput): SessionPartitionView {
  const { context, group, kind, parentMatches } = input;
  const childMatches = partitionNameMatches(group.name, context.query);
  const sessions = sortSystemSessions(group.sessions.filter((session) => {
    return parentMatches || childMatches || context.matchesSearch(session, context.query);
  }));
  return {
    id: sourceChildPartitionID(kind, group.ownerID),
    name: group.name,
    readOnly: true,
    sessions,
  };
}

function sortSystemSessions(sessions: SessionMetadata[]): SessionMetadata[] {
  return sessions
    .sort(compareSessionsByRecentActivity);
}

function systemPartitionName(kind: SessionSourceKind, copy: ChatCopy): string {
  return SYSTEM_PARTITION_NAME_RESOLVERS[kind](copy);
}

function sourceChildPartitionID(kind: SessionSourceKind, ownerID: string): string {
  return `${SOURCE_PARTITION_IDS[kind]}${SOURCE_CHILD_PARTITION_SEPARATOR}${encodeURIComponent(ownerID)}`;
}

function defaultSessionSearchMatcher(
  session: SessionMetadata,
  normalizedQuery: string,
): boolean {
  return !normalizedQuery || session.id.toLowerCase().includes(normalizedQuery);
}
