import {
  parseSessionDetail,
  parseSessionMetadataList,
  parseSessionSidebarPartitionState,
} from './parser';
import type {
  SessionDetail,
  SessionMetadata,
  SessionSidebarPartitionState,
} from '@/lib/types';

const SESSION_PAGE = {
  limit: 100,
  before: null,
  start_index: 0,
  end_index: 0,
  has_more_before: false,
  next_before: null,
} as const;

describe('lib/api/sessions/parser', () => {
  it('parses session metadata lists', () => {
    const payload: SessionMetadata[] = [
      {
        id: 'session-1',
        created_at: '2026-02-28T10:00:00Z',
        updated_at: '2026-02-28T10:05:00Z',
        message_count: 2,
        token_count: 128,
      },
    ];

    expect(parseSessionMetadataList(payload)).toEqual(payload);
  });

  it('parses session detail with tool and human interaction projections', () => {
    const payload: SessionDetail = {
      id: 'session-1',
      created_at: '2026-02-28T10:00:00Z',
      updated_at: '2026-02-28T10:05:00Z',
      message_count: 1,
      page: SESSION_PAGE,
      token_count: 128,
      messages: [
        {
          index: 0,
          role: 'tool',
          text: 'Need confirmation',
          content: [{ type: 'text', text: 'Need confirmation' }],
          tool_calls: [{ id: 'call-1', name: 'ask_human', arguments: { topic: 'db' } }],
          tool_result: {
            status: 'success',
            tool: 'ask_human',
            trace_id: 'trace-1',
          },
          human_interaction: {
            question_id: 'q-1',
            prompt: 'Which database should I use?',
            selection_mode: 'single',
            options: [{ label: 'PostgreSQL' }],
            answer: 'PostgreSQL',
          },
          tool_call_id: 'call-1',
        },
      ],
    };

    expect(parseSessionDetail(payload)).toEqual(payload);
  });

  it('parses internal session messages', () => {
    const payload: SessionDetail = {
      id: 'session-2',
      created_at: '2026-02-28T10:00:00Z',
      updated_at: '2026-02-28T10:05:00Z',
      message_count: 1,
      page: SESSION_PAGE,
      token_count: 64,
      messages: [
        {
          index: 0,
          role: 'internal',
          text: '[GRAPHQL_EXECUTION_RESULT]\n{"data":{"viewer":{"id":"1"}}}',
        },
      ],
    };

    expect(parseSessionDetail(payload)).toEqual(payload);
  });

  it('parses assistant draft messages with in_progress flag', () => {
    const payload: SessionDetail = {
      id: 'session-draft',
      created_at: '2026-04-04T10:00:00Z',
      updated_at: '2026-04-04T10:05:00Z',
      message_count: 2,
      page: SESSION_PAGE,
      token_count: 256,
      messages: [
        {
          index: 2,
          role: 'assistant',
          text: 'partial answer',
          in_progress: true,
        },
      ],
    };

    expect(parseSessionDetail(payload)).toEqual(payload);
  });

  it('ignores unknown fields in session detail payloads', () => {
    const payload = {
      id: 'session-1',
      created_at: '2026-02-28T10:00:00Z',
      updated_at: '2026-02-28T10:05:00Z',
      message_count: 1,
      page: SESSION_PAGE,
      token_count: 128,
      future_field: true,
      messages: [
        {
          index: 0,
          role: 'user',
          text: 'hello',
          extra_message_field: 'ignored',
          content: [{ type: 'text', text: 'hello', extra_part_field: 'ignored' }],
        },
      ],
    };

    expect(parseSessionDetail(payload)).toEqual({
      id: 'session-1',
      created_at: '2026-02-28T10:00:00Z',
      updated_at: '2026-02-28T10:05:00Z',
      message_count: 1,
      page: SESSION_PAGE,
      token_count: 128,
      messages: [
        {
          index: 0,
          role: 'user',
          text: 'hello',
          content: [{ type: 'text', text: 'hello' }],
        },
      ],
    });
  });

  it('parses session sidebar partition state', () => {
    const payload: SessionSidebarPartitionState = {
      version: 1,
      partitions: [{ id: 'work', name: 'Work' }],
      assignments: { 'session-1': 'work' },
    };

    expect(parseSessionSidebarPartitionState(payload)).toEqual({
      version: 1,
      partitions: [{ id: 'work', name: 'Work' }],
      assignments: { 'session-1': 'work' },
    });
  });

  it('rejects invalid session sidebar partition field types', () => {
    expect(() => parseSessionSidebarPartitionState({
      version: 1,
      partitions: [{ id: 'work', name: 'Work' }],
      assignments: { 'session-1': 1 },
    })).toThrow('Invalid session sidebar partition state.assignments.session-1: expected string');
  });
});
