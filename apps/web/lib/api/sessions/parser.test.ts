import { parseSessionDetail, parseSessionMetadataList } from './parser';
import type { SessionDetail, SessionMetadata } from '@/lib/types';

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
      token_count: 128,
      messages: [
        {
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

  it('rejects unexpected fields in session detail messages', () => {
    expect(() => {
      parseSessionDetail({
        id: 'session-1',
        created_at: '2026-02-28T10:00:00Z',
        updated_at: '2026-02-28T10:05:00Z',
        token_count: 128,
        messages: [{ Role: 'user', text: 'hello' }],
      });
    }).toThrow('Invalid session detail.messages[0]: unexpected field "Role"');
  });
});
