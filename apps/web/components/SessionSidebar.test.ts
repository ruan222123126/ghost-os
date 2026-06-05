import {
  buildFlatSessionList,
  buildFlatSessionRows,
  buildPartitionSessionRows,
  countSessionsInPartitionViews,
  limitPartitionViewsBySessionCount,
} from './SessionSidebarFlatList';
import type { SessionMetadata } from '@/lib/types';
import type { SessionPartitionView } from '@/lib/sessionSidebarPartitions';

describe('components/SessionSidebar', () => {
  it('builds flat session rows by recent activity', () => {
    const sessions: SessionMetadata[] = [
      createSession('older-3333', '2026-04-10T00:00:00Z'),
      createSession('latest-1111', '2026-04-12T12:00:00Z'),
      createSession('middle-2222', '2026-04-11T08:00:00Z'),
    ];

    const rows = buildFlatSessionRows(buildFlatSessionList(sessions, ''));

    expect(rows.map((row) => row.kind === 'session' ? row.session.id : '')).toEqual([
      'latest-1111',
      'middle-2222',
      'older-3333',
    ]);
  });

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
});

function createSession(id: string, updatedAt = '2026-04-12T00:00:00Z'): SessionMetadata {
  return {
    id,
    title: '',
    created_at: '2026-04-12T00:00:00Z',
    updated_at: updatedAt,
    message_count: 0,
    token_count: 0,
  };
}
