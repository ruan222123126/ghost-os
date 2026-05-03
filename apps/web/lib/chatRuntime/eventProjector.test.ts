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
        type: 'mark_streaming_thinking_boundary',
      },
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
        type: 'mark_streaming_thinking_boundary',
      },
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

  it('shows a pending tool card as soon as structured tool name is streamed', () => {
    const runtime = createChatRuntimeState('trace-6', 'session-6');

    const started = projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'tool_call_start',
          tool_call_index: 0,
          tool_call_id: 'call-6',
          tool_name: 'web_search',
        },
        { traceId: 'trace-6' },
      ),
    });
    const delta = projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'tool_call_delta',
          tool_call_index: 0,
          arguments_fragment: '{"query":"gol',
        },
        { traceId: 'trace-6' },
      ),
    });
    const executionStart = projectAgentEvent({
      runtime,
      event: buildEvent(
        'tool_call_started',
        {
          tool: 'web_search',
          tool_call_id: 'call-6',
        },
        { traceId: 'trace-6' },
      ),
    });

    expect(started).toEqual([
      {
        type: 'mark_streaming_thinking_boundary',
      },
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tool:trace-6:call-6',
          content: '',
          toolCallId: 'call-6',
          toolName: 'web_search',
          toolStatus: 'pending',
          traceId: 'trace-6',
        },
      },
    ]);
    expect(delta).toEqual([
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tool:trace-6:call-6',
          content: '{"query":"gol',
          toolCallId: 'call-6',
          toolName: 'web_search',
          toolStatus: 'pending',
          traceId: 'trace-6',
        },
      },
    ]);
    expect(executionStart).toEqual([
      {
        type: 'mark_streaming_thinking_boundary',
      },
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tool:trace-6:call-6',
          content: '{"query":"gol',
          toolCallId: 'call-6',
          toolName: 'web_search',
          toolStatus: 'running',
          traceId: 'trace-6',
        },
      },
    ]);
  });

  it('keeps bash_exec command visible during structured preview and running states', () => {
    const runtime = createChatRuntimeState('trace-bash', 'session-bash');

    const started = projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'tool_call_start',
          tool_call_index: 0,
          tool_call_id: 'call-bash',
          tool_name: 'bash_exec',
        },
        { traceId: 'trace-bash' },
      ),
    });
    const delta = projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'tool_call_delta',
          tool_call_index: 0,
          tool_call_id: 'call-bash',
          arguments_fragment: '{"command":"which agent-browser"}',
        },
        { traceId: 'trace-bash' },
      ),
    });
    const executionStart = projectAgentEvent({
      runtime,
      event: buildEvent(
        'tool_call_started',
        {
          tool: 'bash_exec',
          tool_call_id: 'call-bash',
        },
        { traceId: 'trace-bash' },
      ),
    });

    expect(started).toEqual([
      {
        type: 'mark_streaming_thinking_boundary',
      },
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tool:trace-bash:call-bash',
          content: '',
          toolCallId: 'call-bash',
          toolName: 'bash_exec',
          toolStatus: 'pending',
          traceId: 'trace-bash',
        },
      },
    ]);
    expect(delta).toEqual([
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tool:trace-bash:call-bash',
          content: '{"command":"which agent-browser"}',
          toolInput: '{"command":"which agent-browser"}',
          toolCallId: 'call-bash',
          toolName: 'bash_exec',
          toolStatus: 'pending',
          traceId: 'trace-bash',
        },
      },
    ]);
    expect(executionStart).toEqual([
      {
        type: 'mark_streaming_thinking_boundary',
      },
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tool:trace-bash:call-bash',
          content: '{"command":"which agent-browser"}',
          toolInput: '{"command":"which agent-browser"}',
          toolCallId: 'call-bash',
          toolName: 'bash_exec',
          toolStatus: 'running',
          traceId: 'trace-bash',
        },
      },
    ]);
  });

  it('keeps promoted file action visible after tool output replaces streamed arguments', () => {
    const runtime = createChatRuntimeState('trace-read', 'session-read');

    projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'tool_call_start',
          tool_call_index: 0,
          tool_call_id: 'call-read',
          tool_name: 'read_file',
        },
        { traceId: 'trace-read' },
      ),
    });
    projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'tool_call_delta',
          tool_call_index: 0,
          tool_call_id: 'call-read',
          arguments_fragment: '{"path":"/tmp/main.go"}',
        },
        { traceId: 'trace-read' },
      ),
    });

    const finished = projectAgentEvent({
      runtime,
      event: buildEvent(
        'tool_call_finished',
        {
          tool: 'read_file',
          tool_call_id: 'call-read',
          status: 'success',
          output: 'package main',
        },
        { traceId: 'trace-read' },
      ),
    });

    expect(finished).toEqual([
      {
        type: 'mark_streaming_thinking_boundary',
      },
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tool:trace-read:call-read',
          content: 'package main',
          toolInput: '{"path":"/tmp/main.go"}',
          toolCallId: 'call-read',
          toolName: 'read_file',
          toolStatus: 'success',
          traceId: 'trace-read',
        },
      },
    ]);
  });

  it('keeps bash_exec command visible when legacy tool-tag preview enters running state', () => {
    const runtime = createChatRuntimeState('trace-tag-bash', 'session-tag-bash');
    projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'text',
          text: '<t:1>{"command":"which agent-browser"}</t>',
        },
        { traceId: 'trace-tag-bash' },
      ),
    });
    const actions = projectAgentEvent({
      runtime,
      event: buildEvent(
        'tool_call_started',
        {
          tool: 'bash_exec',
          tool_call_id: 'call-tag-bash',
        },
        { traceId: 'trace-tag-bash' },
      ),
    });

    expect(actions).toEqual([
      {
        type: 'mark_streaming_thinking_boundary',
      },
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tag-tool:trace-tag-bash:1',
          content: '{"command":"which agent-browser"}',
          toolInput: '{"command":"which agent-browser"}',
          toolCallId: 'call-tag-bash',
          toolName: 'bash_exec',
          toolStatus: 'running',
          traceId: 'trace-tag-bash',
        },
      },
    ]);
  });

  it('starts a new tool card when a later structured batch reuses tool_call_index 0', () => {
    const runtime = createChatRuntimeState('trace-8', 'session-8');

    projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'tool_call_start',
          tool_call_index: 0,
          tool_call_id: 'call-8a',
          tool_name: 'web_search',
        },
        { traceId: 'trace-8' },
      ),
    });
    projectAgentEvent({
      runtime,
      event: buildEvent(
        'tool_call_started',
        {
          tool: 'web_search',
          tool_call_id: 'call-8a',
        },
        { traceId: 'trace-8' },
      ),
    });

    const secondBatchStarted = projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'tool_call_start',
          tool_call_index: 0,
          tool_call_id: 'call-8b',
          tool_name: 'read_file',
        },
        { traceId: 'trace-8' },
      ),
    });

    expect(secondBatchStarted).toEqual([
      {
        type: 'mark_streaming_thinking_boundary',
      },
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tool:trace-8:call-8b',
          content: '',
          toolCallId: 'call-8b',
          toolName: 'read_file',
          toolStatus: 'pending',
          traceId: 'trace-8',
        },
      },
    ]);
  });

  it('reuses structured preview card when tool_call_id is only known after tool text is complete', () => {
    const runtime = createChatRuntimeState('trace-7', 'session-7');

    projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'tool_call_start',
          tool_call_index: 0,
          tool_name: 'script_exec',
        },
        { traceId: 'trace-7' },
      ),
    });
    projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'tool_call_delta',
          tool_call_index: 0,
          arguments_fragment: '{"script":"echo ok"}',
        },
        { traceId: 'trace-7' },
      ),
    });
    projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'tool_call_end',
          tool_call_index: 0,
        },
        { traceId: 'trace-7' },
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
        { traceId: 'trace-7' },
      ),
    });

    expect(started).toEqual([
      {
        type: 'mark_streaming_thinking_boundary',
      },
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tool:trace-7:preview:1:index:0',
          content: '{"script":"echo ok"}',
          toolCallId: 'call-7',
          toolName: 'script_exec',
          toolStatus: 'running',
          traceId: 'trace-7',
        },
      },
    ]);
  });

  it('creates a fresh preview card for the next delayed-id structured batch', () => {
    const runtime = createChatRuntimeState('trace-9', 'session-9');

    projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'tool_call_start',
          tool_call_index: 0,
          tool_name: 'script_exec',
        },
        { traceId: 'trace-9' },
      ),
    });
    projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'tool_call_end',
          tool_call_index: 0,
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
          tool_call_id: 'call-9a',
        },
        { traceId: 'trace-9' },
      ),
    });

    const secondPreview = projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'tool_call_start',
          tool_call_index: 0,
          tool_name: 'read_file',
        },
        { traceId: 'trace-9' },
      ),
    });
    projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'tool_call_end',
          tool_call_index: 0,
        },
        { traceId: 'trace-9' },
      ),
    });
    const secondStarted = projectAgentEvent({
      runtime,
      event: buildEvent(
        'tool_call_started',
        {
          tool: 'read_file',
          tool_call_id: 'call-9b',
        },
        { traceId: 'trace-9' },
      ),
    });

    expect(secondPreview).toEqual([
      {
        type: 'mark_streaming_thinking_boundary',
      },
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tool:trace-9:preview:2:index:0',
          content: '',
          toolName: 'read_file',
          toolStatus: 'pending',
          traceId: 'trace-9',
        },
      },
    ]);
    expect(secondStarted).toEqual([
      {
        type: 'mark_streaming_thinking_boundary',
      },
      {
        type: 'upsert_streaming_tool',
        tool: {
          id: 'stream-tool:trace-9:preview:2:index:0',
          content: 'read_file running',
          toolCallId: 'call-9b',
          toolName: 'read_file',
          toolStatus: 'running',
          traceId: 'trace-9',
        },
      },
    ]);
  });

  it('projects thinking deltas into streaming thinking actions', () => {
    const runtime = createChatRuntimeState('trace-1', 'session-1');
    const actions = projectAgentEvent({
      runtime,
      event: buildEvent(
        'completion_delta',
        {
          kind: 'thinking',
          thinking: 'Analyzing...',
        },
        { traceId: 'trace-1' },
      ),
    });

    expect(actions).toEqual([
      {
        type: 'append_streaming_thinking_text',
        text: 'Analyzing...',
      },
    ]);
  });

  it('clears streaming thinking on terminal error events', () => {
    const runtime = createChatRuntimeState('trace-3', 'session-3');
    const actions = projectAgentEvent({
      runtime,
      event: buildEvent(
        'error',
        {
          message: 'boom',
        },
        { traceId: 'trace-3' },
      ),
    });

    expect(actions).toEqual([
      {
        type: 'clear_streaming_thinking_text',
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
