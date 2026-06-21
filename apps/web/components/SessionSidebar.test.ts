import {
  buildPartitionSessionRows,
  countSessionsInPartitionViews,
  limitPartitionViewsBySessionCount,
} from './SessionSidebarFlatList';
import { buildSearchDialogOriginStyle, buildSessionSearchResults, mergeSessionSearchResults } from './SessionSearchDialog';
import type { SessionMetadata } from '@/lib/types';
import type { SessionPartitionView } from '@/lib/sessionSidebarPartitions';

describe('components/SessionSidebar', () => {
  it('builds grouped session rows without rendering the full sidebar DOM', () => {
    const partitions: SessionPartitionView[] = [
      {
        id: 'work',
        name: 'Work',
        sessions: [
          createSession('latest-1111', '2026-04-12T12:00:00Z'),
          createSession('middle-2222', '2026-04-11T08:00:00Z'),
        ],
      },
    ];

    const rows = buildPartitionSessionRows({
      partitionViews: partitions,
      draggingSessionID: '',
    });

    expect(rows.map((row) => row.key)).toEqual([
      'partition:work',
      'partition:work:session:latest-1111',
      'partition:work:session:middle-2222',
    ]);
  });

  it('builds collapsed grouped rows with partition headers only', () => {
    const partitions: SessionPartitionView[] = [
      {
        id: 'recent',
        name: 'Recent',
        sessions: [
          createSession('latest-1111', '2026-04-12T12:00:00Z'),
          createSession('middle-2222', '2026-04-11T08:00:00Z'),
        ],
      },
      {
        id: 'workflow',
        name: 'Workflow',
        readOnly: true,
        sessions: [
          createSession('workflow-1', '2026-04-10T08:00:00Z'),
        ],
        childPartitions: [
          {
            id: 'workflow::daily',
            name: 'Daily',
            readOnly: true,
            sessions: [
              createSession('workflow-1', '2026-04-10T08:00:00Z'),
            ],
          },
        ],
      },
    ];
    const collapsedPartitionIDs = new Set(['recent', 'workflow']);

    const rows = buildPartitionSessionRows({
      partitionViews: partitions,
      draggingSessionID: '',
      collapsedPartitionIDs,
    });

    expect(rows.map((row) => row.key)).toEqual([
      'partition:recent',
      'partition:workflow',
    ]);
    expect(rows.every((row) => row.kind === 'partition-header' && row.isCollapsed)).toBe(true);
    expect(countSessionsInPartitionViews(partitions, collapsedPartitionIDs)).toBe(0);
    expect(limitPartitionViewsBySessionCount(partitions, 0, collapsedPartitionIDs).map((view) => view.id)).toEqual([
      'recent',
      'workflow',
    ]);
  });

  it('limits grouped history to the requested number of sessions', () => {
    const partitions: SessionPartitionView[] = [
      {
        id: 'workflow',
        name: 'Workflow',
        sessions: [
          createSession('workflow-1', '2026-04-12T12:00:00Z'),
          createSession('workflow-2', '2026-04-11T08:00:00Z'),
        ],
      },
      {
        id: 'task',
        name: 'Task',
        sessions: [
          createSession('task-1', '2026-04-10T08:00:00Z'),
        ],
      },
    ];

    expect(countSessionsInPartitionViews(partitions)).toBe(3);
    expect(limitPartitionViewsBySessionCount(partitions, 2)).toEqual([
      {
        id: 'workflow',
        name: 'Workflow',
        sessions: [
          createSession('workflow-1', '2026-04-12T12:00:00Z'),
          createSession('workflow-2', '2026-04-11T08:00:00Z'),
        ],
      },
    ]);
  });

  it('builds search dialog results from session title and id', () => {
    const sessions: SessionMetadata[] = [
      createSession('older-3333', '2026-04-10T00:00:00Z', 'Daily Notes'),
      createSession('latest-1111', '2026-04-12T12:00:00Z', 'GPT 概览与核心能力介绍'),
      createSession('middle-2222', '2026-04-11T08:00:00Z', 'Hello'),
    ];

    const byTitle = buildSessionSearchResults({
      sessions,
      query: '核心能力',
      resolveSessionTitle: (session) => session.title,
    });
    expect(byTitle.map((session) => session.id)).toEqual(['latest-1111']);

    const byID = buildSessionSearchResults({
      sessions,
      query: '2222',
      resolveSessionTitle: (session) => session.title,
    });
    expect(byID.map((session) => session.id)).toEqual(['middle-2222']);
  });

  it('builds search dialog results from visible partition names', () => {
    const sessions: SessionMetadata[] = [
      createSession('older-3333', '2026-04-10T00:00:00Z', 'Daily Notes'),
      createSession('latest-1111', '2026-04-12T12:00:00Z', 'Review'),
      createSession('middle-2222', '2026-04-11T08:00:00Z', 'Hello'),
    ];
    const partitionViews: SessionPartitionView[] = [
      {
        id: 'client-work',
        name: 'Client Work',
        sessions: [sessions[1], sessions[2]],
      },
    ];

    const results = buildSessionSearchResults({
      sessions,
      partitionViews,
      query: 'client',
      resolveSessionTitle: (session) => session.title,
    });

    expect(results.map((session) => session.id)).toEqual(['latest-1111', 'middle-2222']);
  });

  it('merges backend and local alias search results without duplicates', () => {
    const remote = [
      createSession('remote-1', '2026-04-12T12:00:00Z', 'Remote'),
      createSession('shared-1', '2026-04-11T12:00:00Z', 'Shared'),
    ];
    const local = [
      createSession('shared-1', '2026-04-11T12:00:00Z', 'Shared Alias'),
      createSession('local-1', '2026-04-10T12:00:00Z', 'Local Alias'),
    ];

    expect(mergeSessionSearchResults(remote, local).map((session) => session.id)).toEqual([
      'remote-1',
      'shared-1',
      'local-1',
    ]);
  });

  it('places the search dialog animation origin at the trigger center', () => {
    expect(buildSearchDialogOriginStyle({
      dialogRect: { left: 180, top: 96, width: 640, height: 360 },
      triggerRect: { left: 272, top: 20, width: 40, height: 40 },
    })).toEqual({ transformOrigin: '112px -56px' });
  });
});

function createSession(id: string, updatedAt = '2026-04-12T00:00:00Z', title = ''): SessionMetadata {
  return {
    id,
    title,
    created_at: '2026-04-12T00:00:00Z',
    updated_at: updatedAt,
    message_count: 0,
    token_count: 0,
  };
}
