import {
  buildFlatSessionList,
  buildFlatSessionRows,
  buildPartitionSessionRows,
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
