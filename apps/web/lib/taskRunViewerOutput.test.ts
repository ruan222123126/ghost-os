import { buildTaskRunCardOutput } from './taskRunViewerOutput';
import type { AgentStreamEvent, SessionDetail } from './types';
import type { LiveTaskRunCard } from './taskRunViewerCards';

const SESSION_PAGE = {
  limit: 100,
  before: null,
  start_index: 0,
  end_index: 0,
  has_more_before: false,
  next_before: null,
} as const;

describe('taskRunViewerOutput', () => {
  it('falls back to summary text for completed relay cards', () => {
    const output = buildTaskRunCardOutput({
      card_id: 'card-relay',
      kind: 'relay_round',
      status: 'success',
      final_text: 'did: inspect\\nnext_step: fix config',
      source_events: [],
    }, null);

    expect(output.committedMessages).toMatchObject([
      { kind: 'assistant', content: 'did: inspect\\nnext_step: fix config' },
    ]);
    expect(output.streamingRows).toHaveLength(0);
  });

  it('projects live source events into output rows', () => {
    const card: LiveTaskRunCard = {
      card_id: 'card-live',
      kind: 'workflow_agent',
      live_source_session_id: 'session-live',
      source_events: [
        {
          id: 'event-1',
          step_id: 'turn-0001-assistant',
          trace_id: 'trace-live',
          session_id: 'session-live',
          turn: 1,
          type: 'message',
          payload: { text: 'final answer', session_id: 'session-live' },
          at: '2026-05-30T00:00:00Z',
        },
      ],
    };

    const output = buildTaskRunCardOutput(card, null);

    expect(output.committedMessages).toMatchObject([
      { kind: 'assistant', content: 'final answer' },
    ]);
  });

  it('does not duplicate committed assistant text already present in the source session snapshot', () => {
    const card: LiveTaskRunCard = {
      card_id: 'card-live',
      kind: 'workflow_agent',
      live_source_session_id: 'session-live',
      source_events: [
        buildEvent('message', {
          text: 'final answer',
          session_id: 'session-live',
        }, {
          at: '2026-05-30T00:00:01Z',
        }),
      ],
    };
    const session: SessionDetail = {
      id: 'session-live',
      title: 'Live',
      created_at: '2026-05-30T00:00:00Z',
      updated_at: '2026-05-30T00:00:02Z',
      message_count: 2,
      page: SESSION_PAGE,
      token_count: 0,
      messages: [
        { index: 0, role: 'user', text: 'hi' },
        { index: 1, role: 'assistant', text: 'final answer' },
      ],
      turn_draft: null,
    };

    const output = buildTaskRunCardOutput(card, session);

    expect(output.committedMessages).toMatchObject([
      { kind: 'user', content: 'hi' },
      { kind: 'assistant', content: 'final answer' },
    ]);
    expect(output.committedMessages).toHaveLength(2);
  });

  it('projects live completion deltas into streaming assistant rows', () => {
    const card: LiveTaskRunCard = {
      card_id: 'card-live',
      kind: 'workflow_agent',
      live_source_session_id: 'session-live',
      source_events: [
        buildEvent('completion_delta', {
          kind: 'text',
          text: 'partial answer',
        }),
      ],
    };

    const output = buildTaskRunCardOutput(card, null);

    expect(output.committedMessages).toHaveLength(0);
    expect(output.streamingRows).toMatchObject([
      { message: { kind: 'assistant', content: 'partial answer' } },
    ]);
  });

  it('replays only source events newer than the loaded source session draft', () => {
    const card: LiveTaskRunCard = {
      card_id: 'card-live',
      kind: 'workflow_agent',
      live_source_session_id: 'session-live',
      source_events: [
        buildEvent('completion_delta', {
          kind: 'text',
          text: 'partial ',
        }, {
          at: '2026-05-30T00:00:01Z',
        }),
        buildEvent('completion_delta', {
          kind: 'text',
          text: 'answer',
        }, {
          at: '2026-05-30T00:00:03Z',
        }),
      ],
    };
    const session: SessionDetail = {
      id: 'session-live',
      title: 'Live',
      created_at: '2026-05-30T00:00:00Z',
      updated_at: '2026-05-30T00:00:02Z',
      message_count: 1,
      page: SESSION_PAGE,
      token_count: 0,
      messages: [],
      turn_draft: {
        trace_id: 'trace-live',
        turn: 1,
        status: 'streaming',
        pending_questions: [],
        assistant_segments: [{ id: 'stream-assistant:trace-live', content: 'partial ' }],
        thinking_segments: [],
        tools: [],
        item_order: ['assistant:stream-assistant:trace-live'],
      },
    };

    const output = buildTaskRunCardOutput(card, session);

    expect(output.committedMessages).toHaveLength(0);
    expect(output.streamingRows).toMatchObject([
      { message: { kind: 'assistant', content: 'partial answer' } },
    ]);
  });

  it('hides active tools and keeps completed tools in the live viewer output', () => {
    const card: LiveTaskRunCard = {
      card_id: 'card-live',
      kind: 'workflow_agent',
      live_source_session_id: 'session-live',
      source_events: [
        buildEvent('tool_call_started', {
          tool: 'script_exec',
          tool_call_id: 'call-running',
          arguments_json: '{"script":"pwd"}',
        }),
        buildEvent('tool_call_started', {
          tool: 'read_file',
          tool_call_id: 'call-success',
          arguments_json: '{"path":"README.md"}',
        }),
        buildEvent('tool_call_finished', {
          tool: 'read_file',
          tool_call_id: 'call-success',
          status: 'success',
          output: 'README',
        }),
      ],
    };
    const session: SessionDetail = {
      id: 'session-live',
      title: 'Live',
      created_at: '2026-05-30T00:00:00Z',
      updated_at: '2026-05-30T00:00:01Z',
      message_count: 0,
      page: SESSION_PAGE,
      token_count: 0,
      messages: [],
      turn_draft: {
        trace_id: 'trace-live',
        turn: 1,
        status: 'streaming',
        pending_questions: [],
        assistant_segments: [],
        thinking_segments: [],
        tools: [
          {
            id: 'tool-running',
            content: '{"script":"ls"}',
            tool_name: 'script_exec',
            tool_status: 'running',
          },
          {
            id: 'tool-success',
            content: 'README.md',
            tool_name: 'read_file',
            tool_status: 'success',
          },
        ],
        item_order: [
          'tool:tool-running',
          'tool:tool-success',
        ],
      },
    };

    const output = buildTaskRunCardOutput(card, session);
    const tools = output.streamingRows
      .map((row) => row.message)
      .filter((message): message is Extract<typeof message, { kind: 'tool' }> => message.kind === 'tool');

    expect(tools).toMatchObject([
      { id: 'tool-success', toolStatus: 'success', content: 'README.md' },
      { toolCallId: 'call-success', toolStatus: 'success', content: 'README' },
    ]);
    expect(tools).toHaveLength(2);
  });
});

function buildEvent(
  type: AgentStreamEvent['type'],
  payload: Record<string, unknown>,
  options?: {
    at?: string;
    id?: string;
    sessionId?: string;
    traceId?: string;
  },
): AgentStreamEvent {
  return {
    id: options?.id ?? `event-${type}`,
    step_id: `step-${type}`,
    trace_id: options?.traceId ?? 'trace-live',
    session_id: options?.sessionId ?? 'session-live',
    turn: 1,
    type,
    payload,
    at: options?.at,
  };
}
