import { deleteSession, getSession, listSessions } from './api';
import { fetchMock, installFetchMock, mockFetchJSON } from '@/lib/api.test.helpers';
import type { SessionDetail, SessionMetadata } from '@/lib/types';

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
      token_count: 128,
      messages: [
        { role: 'user', text: 'hello' },
        { role: 'assistant', text: 'hi' },
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
      token_count: 128,
      messages: [
        {
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

  it('getSession rejects legacy-cased session messages', async () => {
    mockFetchJSON({
      status: 'success',
      payload: {
        id: 'session-1',
        created_at: '2026-02-28T10:00:00Z',
        updated_at: '2026-02-28T10:05:00Z',
        token_count: 128,
        messages: [{ Role: 'user', Text: 'hello' }],
      },
      error: '',
    });

    await expect(getSession('session-1')).rejects.toThrow(
      'Invalid session detail.messages[0]: unexpected field "Role"',
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
});
