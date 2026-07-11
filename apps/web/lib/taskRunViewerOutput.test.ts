import { buildTaskRunCardOutput } from './taskRunViewerOutput';
import type { AgentStreamEvent, SessionDetail } from './types';
import { hydrateLiveTaskRunCards, type LiveTaskRunCard } from './taskRunViewerCards';

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

  it('projects completed relay cards with source events into output rows', () => {
    const output = buildTaskRunCardOutput({
      card_id: 'card-relay-complete',
      kind: 'relay_round',
      status: 'success',
      final_text: 'did: finished\nnext_step: none',
      source_events: [
        buildEvent('completion_delta', {
          kind: 'thinking',
          thinking: 'checking final state',
        }),
        buildEvent('tool_call_finished', {
          tool: 'relay_complete',
          tool_call_id: 'call-relay-complete',
          status: 'success',
          output: '{"status":"relay_completed"}',
        }),
      ],
    }, null);

    expect(output.committedMessages).toHaveLength(0);
    expect(output.streamingRows).toMatchObject([
      {
        message: {
          kind: 'tool',
          toolName: 'relay_complete',
          toolStatus: 'success',
          content: '{"status":"relay_completed"}',
        },
      },
    ]);
    expect(output.streamingRows).toHaveLength(1);
  });

  it('does not replace completed live card source events with relay preview text', () => {
    const output = buildTaskRunCardOutput({
      card_id: 'card-relay-preview',
      kind: 'relay_round',
      status: 'success',
      preview: 'did: finished from preview',
      source_events: [
        buildEvent('completion_delta', {
          kind: 'thinking',
          thinking: 'checking final state',
        }),
      ],
    }, null);

    expect(output.committedMessages).toHaveLength(0);
    expect(output.streamingRows).toHaveLength(0);
  });

  it('does not keep thinking rows for terminal cards without done events', () => {
    const output = buildTaskRunCardOutput({
      card_id: 'card-cancelled',
      kind: 'workflow_agent',
      status: 'cancelled',
      source_events: [
        buildEvent('completion_delta', {
          kind: 'thinking',
          thinking: 'still thinking',
        }),
      ],
    }, null);

    expect(output.committedMessages).toHaveLength(0);
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

  it('uses card source events instead of replaying the full source session snapshot', () => {
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
      { kind: 'assistant', content: 'final answer' },
    ]);
    expect(output.committedMessages).toHaveLength(1);
  });

  it('limits selected log cards to that card output when the source session has multiple rounds', () => {
    const card: LiveTaskRunCard = {
      card_id: 'member-round-2',
      kind: 'orchestration_member',
      round: 2,
      live_source_session_id: 'member-session',
      source_events: [
        buildEvent('message', {
          text: 'round 2 answer',
          session_id: 'member-session',
        }, {
          at: '2026-05-30T00:00:03Z',
          id: 'round-2-message',
          sessionId: 'member-session',
          traceId: 'trace-round-2',
        }),
      ],
    };
    const session: SessionDetail = {
      id: 'member-session',
      title: 'Member',
      created_at: '2026-05-30T00:00:00Z',
      updated_at: '2026-05-30T00:00:04Z',
      message_count: 6,
      page: SESSION_PAGE,
      token_count: 0,
      messages: [
        { index: 0, role: 'user', text: 'round 1 prompt' },
        { index: 1, role: 'assistant', text: 'round 1 answer' },
        { index: 2, role: 'user', text: 'round 2 prompt' },
        { index: 3, role: 'assistant', text: 'round 2 answer' },
        { index: 4, role: 'user', text: 'round 3 prompt' },
        { index: 5, role: 'assistant', text: 'round 3 answer' },
      ],
      turn_draft: null,
    };

    const output = buildTaskRunCardOutput(card, session);

    expect(output.committedMessages).toMatchObject([
      { kind: 'assistant', content: 'round 2 answer' },
    ]);
    expect(output.committedMessages.map((message) => message.content)).toEqual(['round 2 answer']);
  });

  it('uses card summary instead of replaying a full source session when card events are missing', () => {
    const card: LiveTaskRunCard = {
      card_id: 'member-round-2',
      kind: 'orchestration_member',
      round: 2,
      status: 'success',
      preview: 'round 2 answer',
      source_events: [],
    };
    const session: SessionDetail = {
      id: 'member-session',
      title: 'Member',
      created_at: '2026-05-30T00:00:00Z',
      updated_at: '2026-05-30T00:00:04Z',
      message_count: 4,
      page: SESSION_PAGE,
      token_count: 0,
      messages: [
        { index: 0, role: 'user', text: 'round 1 prompt' },
        { index: 1, role: 'assistant', text: 'round 1 answer' },
        { index: 2, role: 'user', text: 'round 2 prompt' },
        { index: 3, role: 'assistant', text: 'round 2 answer' },
      ],
      turn_draft: null,
    };

    const output = buildTaskRunCardOutput(card, session);

    expect(output.committedMessages).toMatchObject([
      { kind: 'assistant', content: 'round 2 answer' },
    ]);
    expect(output.committedMessages).toHaveLength(1);
  });

  it('scopes source session fallback to the selected card round when card events are missing', () => {
    const card: LiveTaskRunCard = {
      card_id: 'member-round-2',
      kind: 'orchestration_member',
      round: 2,
      status: 'running',
      source_events: [],
    };
    const session: SessionDetail = {
      id: 'member-session',
      title: 'Member',
      created_at: '2026-05-30T00:00:00Z',
      updated_at: '2026-05-30T00:00:04Z',
      message_count: 4,
      page: SESSION_PAGE,
      token_count: 0,
      messages: [
        { index: 0, role: 'user', text: 'round 1 prompt' },
        { index: 1, role: 'assistant', text: 'round 1 answer' },
        { index: 2, role: 'user', text: 'round 2 prompt' },
        { index: 3, role: 'assistant', text: 'round 2 answer' },
      ],
      turn_draft: {
        trace_id: 'trace-other-round',
        turn: 0,
        status: 'streaming',
        pending_questions: [],
        assistant_segments: [{ id: 'stream-assistant:trace-other-round', content: 'round 3 draft' }],
        thinking_segments: [],
        tools: [],
        item_order: ['assistant:stream-assistant:trace-other-round'],
      },
    };

    const output = buildTaskRunCardOutput(card, session);

    expect(output.committedMessages.map((message) => message.content)).toEqual([
      'round 2 prompt',
      'round 2 answer',
    ]);
    expect(output.streamingRows).toHaveLength(0);
  });

  it('does not replay a multi-round source session when a card cannot be scoped', () => {
    const card: LiveTaskRunCard = {
      card_id: 'workflow-card',
      kind: 'workflow_agent',
      status: 'running',
      source_events: [],
    };
    const session: SessionDetail = {
      id: 'workflow-session',
      title: 'Workflow',
      created_at: '2026-05-30T00:00:00Z',
      updated_at: '2026-05-30T00:00:04Z',
      message_count: 4,
      page: SESSION_PAGE,
      token_count: 0,
      messages: [
        { index: 0, role: 'user', text: 'round 1 prompt' },
        { index: 1, role: 'assistant', text: 'round 1 answer' },
        { index: 2, role: 'user', text: 'round 2 prompt' },
        { index: 3, role: 'assistant', text: 'round 2 answer' },
      ],
      turn_draft: null,
    };

    const output = buildTaskRunCardOutput(card, session);

    expect(output.committedMessages).toHaveLength(0);
    expect(output.streamingRows).toHaveLength(0);
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

  it('uses card source events even when the loaded source session has a draft', () => {
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
      { toolCallId: 'call-success', toolStatus: 'success', content: 'README' },
    ]);
    expect(tools).toHaveLength(1);
  });

  it('replays empty-id historical structured tool deltas into completed tool input', () => {
    const [card] = hydrateLiveTaskRunCards([
      {
        card_id: 'relay-card',
        kind: 'relay_round',
        source_events: [
          {
            id: '',
            step_id: 'turn-0001-assistant',
            trace_id: 'trace-relay',
            session_id: 'session-relay',
            turn: 1,
            type: 'completion_delta',
            payload: { kind: 'tool_call_start', tool_call_index: 0, tool_name: 'bash_exec' },
            at: '2026-05-30T00:00:01Z',
          },
          {
            id: '',
            step_id: 'turn-0001-assistant',
            trace_id: 'trace-relay',
            session_id: 'session-relay',
            turn: 1,
            type: 'completion_delta',
            payload: { kind: 'tool_call_delta', tool_call_index: 0, arguments_fragment: '{"command":"pwd"}' },
            at: '2026-05-30T00:00:01Z',
          },
          {
            id: '',
            step_id: 'turn-0001-assistant',
            trace_id: 'trace-relay',
            session_id: 'session-relay',
            turn: 1,
            type: 'completion_delta',
            payload: { kind: 'tool_call_end', tool_call_index: 0 },
            at: '2026-05-30T00:00:01Z',
          },
          {
            id: '',
            step_id: 'turn-0001-tool-0000',
            trace_id: 'trace-relay',
            session_id: 'session-relay',
            turn: 1,
            type: 'tool_call_started',
            payload: { tool: 'bash_exec', tool_call_id: 'call-1' },
            at: '2026-05-30T00:00:02Z',
          },
          {
            id: '',
            step_id: 'turn-0001-tool-0000',
            trace_id: 'trace-relay',
            session_id: 'session-relay',
            turn: 1,
            type: 'tool_call_finished',
            payload: { tool: 'bash_exec', tool_call_id: 'call-1', status: 'success', output: 'done' },
            at: '2026-05-30T00:00:03Z',
          },
        ],
      },
    ]);

    const output = buildTaskRunCardOutput(card, null);
    const tools = output.streamingRows
      .map((row) => row.message)
      .filter((message): message is Extract<typeof message, { kind: 'tool' }> => message.kind === 'tool');

    expect(tools).toMatchObject([
      {
        toolCallId: 'call-1',
        toolName: 'bash_exec',
        toolStatus: 'success',
        toolInput: '{"command":"pwd"}',
        content: 'done',
      },
    ]);
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
