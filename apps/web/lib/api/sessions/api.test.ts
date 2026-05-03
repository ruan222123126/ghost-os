import {
  deleteSession,
  getSession,
  getSessionSidebarPartitions,
  listSessions,
  putSessionSidebarPartitions,
} from './api';
import { fetchMock, installFetchMock, mockFetchJSON } from '@/lib/api.test.helpers';
import type {
  SessionDetail,
  SessionMetadata,
  SessionSidebarPartitionState,
} from '@/lib/types';

const SESSION_PAGE = {
  limit: 100,
  before: null,
  start_index: 0,
  end_index: 1,
  has_more_before: false,
  next_before: null,
} as const;

describe('lib/api/sessions/api', () => {
  beforeEach(() => {
    installFetchMock();
  });

  it('listSessions reads /api/sessions with GET', async () => {
    const expected: SessionMetadata[] = [
      {
        id: 'session-1',
        created_at: '2026-02-28T10:00:00Z',
        updated_at: '2026-02-28T10:05:00Z',
        message_count: 2,
        token_count: 128,
      },
    ];

    mockFetchJSON({
      status: 'success',
      payload: expected,
      error: '',
    });

    const sessions = await listSessions();

    expect(sessions).toEqual(expected);
    expect(fetchMock).toHaveBeenCalledWith('/api/sessions', expect.any(Object));
  });

  it('getSession reads session detail by id', async () => {
    const expected: SessionDetail = {
      id: 'session-1',
      created_at: '2026-02-28T10:00:00Z',
      updated_at: '2026-02-28T10:05:00Z',
      message_count: 2,
      page: SESSION_PAGE,
      token_count: 128,
      messages: [
        { index: 0, role: 'user', text: 'hello' },
        { index: 1, role: 'assistant', text: 'hi' },
      ],
    };

    mockFetchJSON({
      status: 'success',
      payload: expected,
      error: '',
    });

    const session = await getSession('session-1');

    expect(session).toEqual(expected);
    expect(fetchMock).toHaveBeenCalledWith('/api/sessions/session-1', expect.any(Object));
  });

  it('getSession accepts structured tool projections from the bridge contract', async () => {
    const expected: SessionDetail = {
      id: 'session-1',
      created_at: '2026-02-28T10:00:00Z',
      updated_at: '2026-02-28T10:05:00Z',
      message_count: 1,
      page: {
        ...SESSION_PAGE,
        end_index: 0,
      },
      token_count: 128,
      messages: [
        {
          index: 0,
          role: 'tool',
          text: 'Which database should I use?\nPostgreSQL',
          tool_result: {
            status: 'success',
            tool: 'ask_human',
            trace_id: 'trace-2',
          },
          human_interaction: {
            question_id: 'q-1',
            prompt: 'Which database should I use?',
            selection_mode: 'single',
            options: [
              { label: 'PostgreSQL' },
              { label: 'Other', allow_custom: true },
            ],
            answer: 'PostgreSQL',
          },
          tool_call_id: 'call-1',
        },
      ],
    };

    mockFetchJSON({
      status: 'success',
      payload: expected,
      error: '',
    });

    const session = await getSession('session-1');

    expect(session).toEqual(expected);
  });

  it('getSession ignores unknown fields added by newer bridge payloads', async () => {
    mockFetchJSON({
      status: 'success',
      payload: {
        id: 'session-1',
        created_at: '2026-02-28T10:00:00Z',
        updated_at: '2026-02-28T10:05:00Z',
        message_count: 1,
        page: {
          ...SESSION_PAGE,
          end_index: 0,
        },
        token_count: 128,
        schema_version: 'vNext',
        messages: [{ index: 0, role: 'user', text: 'hello', extra_field: 'ignored' }],
      },
      error: '',
    });

    await expect(getSession('session-1')).resolves.toEqual({
      id: 'session-1',
      created_at: '2026-02-28T10:00:00Z',
      updated_at: '2026-02-28T10:05:00Z',
      message_count: 1,
      page: {
        ...SESSION_PAGE,
        end_index: 0,
      },
      token_count: 128,
      messages: [{ index: 0, role: 'user', text: 'hello' }],
    });
  });

  it('getSession still rejects messages missing required canonical fields', async () => {
    mockFetchJSON({
      status: 'success',
      payload: {
        id: 'session-1',
        created_at: '2026-02-28T10:00:00Z',
        updated_at: '2026-02-28T10:05:00Z',
        message_count: 1,
        page: {
          ...SESSION_PAGE,
          end_index: 0,
        },
        token_count: 128,
        messages: [{ index: 0, Role: 'user', Text: 'hello' }],
      },
      error: '',
    });

    await expect(getSession('session-1')).rejects.toThrow(
      'Invalid session detail.messages[0].role: expected string',
    );
  });

  it('deleteSession sends delete request by id', async () => {
    mockFetchJSON({
      status: 'success',
      payload: { id: 'session-1', deleted: true },
      error: '',
    });

    await deleteSession('session-1');

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/sessions/session-1',
      expect.objectContaining({
        method: 'DELETE',
      }),
    );
  });

  it('getSessionSidebarPartitions reads /api/sessions/partitions with GET', async () => {
    const expected: SessionSidebarPartitionState = {
      version: 1,
      partitions: [{ id: 'work', name: 'Work' }],
      assignments: { 'session-1': 'work' },
    };

    mockFetchJSON({
      status: 'success',
      payload: expected,
      error: '',
    });

    await expect(getSessionSidebarPartitions()).resolves.toEqual(expected);
    expect(fetchMock).toHaveBeenCalledWith('/api/sessions/partitions', expect.any(Object));
  });

  it('putSessionSidebarPartitions sends the full state payload', async () => {
    const expected: SessionSidebarPartitionState = {
      version: 1,
      partitions: [{ id: 'work', name: 'Work' }],
      assignments: { 'session-1': 'work' },
    };

    mockFetchJSON({
      status: 'success',
      payload: expected,
      error: '',
    });

    await expect(putSessionSidebarPartitions(expected, 'trace-session-partitions')).resolves.toEqual(expected);
    expect(fetchMock).toHaveBeenCalledWith('/api/sessions/partitions', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({
        ...expected,
        trace_id: 'trace-session-partitions',
      }),
    }));
  });
});
