import { projectAgentEvent } from './eventProjector';
import { createChatRuntimeState } from './runtimeState';
import type { AgentStreamEvent } from '@/lib/types';

describe('lib/chatRuntime/eventProjector finalization', () => {
  it('uses structured arguments_json and output from tool lifecycle payloads', () => {
    const runtime = createChatRuntimeState('trace-10', 'session-10');
    const started = projectAgentEvent({
      runtime,
      event: buildEvent(
        'tool_call_started',
        {
          tool: 'script_exec',
          tool_call_id: 'call-10',
          arguments_json: '{"script":"print(1)"}',
        },
        { traceId: 'trace-10' },
      ),
    });
    const finished = projectAgentEvent({
      runtime,
      event: buildEvent(
        'tool_call_finished',
        {
          tool: 'script_exec',
          tool_call_id: 'call-10',
          status: 'success',
          output: '{"summary":{"step_count":1},"steps":[]}',
        },
        { traceId: 'trace-10' },
      ),
    });

    expect(started).toEqual([
      {
        type: 'mark_streaming_thinking_boundary',
      },
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tool:trace-10:call-10',
          content: '{"script":"print(1)"}',
          toolCallId: 'call-10',
          toolName: 'script_exec',
          toolStatus: 'running',
          traceId: 'trace-10',
        },
      },
    ]);
    expect(finished).toEqual([
      {
        type: 'mark_streaming_thinking_boundary',
      },
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tool:trace-10:call-10',
          content: '{"summary":{"step_count":1},"steps":[]}',
          toolCallId: 'call-10',
          toolName: 'script_exec',
          toolStatus: 'success',
          traceId: 'trace-10',
        },
      },
    ]);
  });

  it('splits thinking messages when tool lifecycle events arrive', () => {
    const runtime = createChatRuntimeState('trace-9', 'session-9');
    projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'thinking',
          thinking: 'before tool',
        },
        { traceId: 'trace-9' },
      ),
    });
    projectAgentEvent({
      runtime,
      event: buildEvent(
        'tool_call_started',
        {
          tool: 'script_exec',
          tool_call_id: 'call-9',
        },
        { traceId: 'trace-9' },
      ),
    });
    projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'thinking',
          thinking: 'after tool',
        },
        { traceId: 'trace-9' },
      ),
    });

    const actions = projectAgentEvent({
      runtime,
      event: buildEvent(
        'message',
        {
          text: 'final',
        },
        { traceId: 'trace-9' },
      ),
    });

    expect(actions).toEqual([
      {
        type: 'finalize_streaming_turn',
        assistantMessageId: 'stream-assistant:trace-9',
        assistantText: 'final',
      },
    ]);
  });

  it('commits thinking message before assistant output when message is finalized', () => {
    const runtime = createChatRuntimeState('trace-5', 'session-5');
    projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'thinking',
          thinking: 'step 1\nstep 2',
        },
        { traceId: 'trace-5' },
      ),
    });

    const actions = projectAgentEvent({
      runtime,
      event: buildEvent(
        'message',
        {
          text: 'final answer',
        },
        { traceId: 'trace-5' },
      ),
    });

    expect(actions).toEqual([
      {
        type: 'finalize_streaming_turn',
        assistantMessageId: 'stream-assistant:trace-5',
        assistantText: 'final answer',
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
        type: 'finalize_streaming_turn',
        assistantMessageId: 'stream-assistant:trace-3',
        assistantText: 'hi there',
      },
    ]);
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
