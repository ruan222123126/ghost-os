import { projectAgentEvent } from './eventProjector';
import { createChatRuntimeState } from './runtimeState';
import type { AgentStreamEvent } from '@/lib/types';

describe('lib/chatRuntime/eventProjector', () => {
  it('projects completion delta tool tags into ordered runtime actions', () => {
    const runtime = createChatRuntimeState('trace-1', 'session-1');
    const actions = projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'text',
          text: 'alpha<t:1>{"q":"x"}</t>omega',
        },
        { traceId: 'trace-1' },
      ),
    });

    expect(actions).toEqual([
      {
        type: 'append_streaming_assistant_text',
        text: 'alpha',
      },
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tag-tool:trace-1:1',
          content: '',
          toolName: 'tool#1',
          toolStatus: 'pending',
          traceId: 'trace-1',
        },
      },
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tag-tool:trace-1:1',
          content: '{"q":"x"}',
          toolName: 'tool#1',
          toolStatus: 'pending',
          traceId: 'trace-1',
        },
      },
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tag-tool:trace-1:1',
          content: '{"q":"x"}',
          toolName: 'tool#1',
          toolStatus: 'pending',
          traceId: 'trace-1',
        },
      },
      {
        type: 'append_streaming_assistant_text',
        text: 'omega',
      },
    ]);
  });

  it('maps preview tool message id to tool_call_id lifecycle', () => {
    const runtime = createChatRuntimeState('trace-2', 'session-2');
    projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'text',
          text: '<t:7>{"script":"echo ok"}</t>',
        },
        { traceId: 'trace-2' },
      ),
    });

    const started = projectAgentEvent({
      runtime,
      event: buildEvent(
        'tool_call_started',
        {
          tool: 'script_exec',
          tool_call_id: 'call-7',
        },
        { traceId: 'trace-2' },
      ),
    });
    const finished = projectAgentEvent({
      runtime,
      event: buildEvent(
        'tool_call_finished',
        {
          tool: 'script_exec',
          tool_call_id: 'call-7',
          status: 'success',
        },
        { traceId: 'trace-2' },
      ),
    });

    expect(started).toEqual([
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tag-tool:trace-2:1',
          content: '{"script":"echo ok"}',
          toolCallId: 'call-7',
          toolName: 'script_exec',
          toolStatus: 'running',
          traceId: 'trace-2',
        },
      },
    ]);
    expect(finished).toEqual([
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tag-tool:trace-2:1',
          content: '{"script":"echo ok"}',
          toolCallId: 'call-7',
          toolName: 'script_exec',
          toolStatus: 'success',
          traceId: 'trace-2',
        },
      },
    ]);
  });

  it('finalizes message event by clearing stream text and committing assistant message', () => {
    const runtime = createChatRuntimeState('trace-3', 'session-3');
    projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'text',
          text: 'hi there',
        },
        { traceId: 'trace-3' },
      ),
    });

    const actions = projectAgentEvent({
      runtime,
      event: buildEvent(
        'message',
        {
          text: 'hi there',
        },
        { traceId: 'trace-3' },
      ),
    });

    expect(actions).toEqual([
      {
        type: 'clear_streaming_assistant_text',
      },
      {
        type: 'append_committed_messages',
        messages: [
          {
            id: 'stream-assistant:trace-3',
            kind: 'assistant',
            content: 'hi there',
          },
        ],
      },
    ]);
  });

  it('throws when awaiting_human arrives before session id is known', () => {
    const runtime = createChatRuntimeState('trace-4');
    expect(() => {
      projectAgentEvent({
        runtime,
        event: buildEvent(
          'awaiting_human',
          {
            question_id: 'q-1',
            prompt: 'Choose one',
          },
          { traceId: 'trace-4' },
        ),
      });
    }).toThrow('awaiting_human stream event is missing session_id');
  });
});

function buildEvent(
  type: AgentStreamEvent['type'],
  payload: Record<string, unknown>,
  options?: {
    traceId?: string;
    sessionId?: string;
  },
): AgentStreamEvent {
  return {
    id: `event-${type}`,
    step_id: `step-${type}`,
    trace_id: options?.traceId || 'trace-1',
    session_id: options?.sessionId,
    turn: 1,
    type,
    payload,
    at: undefined,
  };
}
