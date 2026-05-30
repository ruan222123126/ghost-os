import {
  chatStateReducer,
  createInitialChatState,
  type ChatStateStore,
} from './reducer';
import { buildChatStateView } from './runtimeReducer';

describe('lib/chat-store/reducer', () => {
  it('keeps streaming order and boundaries when applying runtime actions', () => {
    let state = createInitialState();
    state = chatStateReducer(state, {
      type: 'apply_runtime_actions',
      actions: [
        { type: 'append_streaming_thinking_text', text: 'thinking...' },
        { type: 'append_streaming_assistant_text', text: 'alpha' },
        {
          type: 'upsert_streaming_tool',
          tool: {
            id: 'tool-1',
            content: 'running',
            toolStatus: 'running',
          },
        },
        { type: 'append_streaming_assistant_text', text: 'omega' },
        {
          type: 'upsert_pending_question',
          question: {
            id: 'question-1',
            kind: 'pending_question',
            content: 'continue?',
            questionId: 'q-1',
            sessionId: 'session-1',
          },
        },
      ],
    });

    const view = buildChatStateView(state);
    expect(view.streamingAssistantSegments).toEqual([
      { id: 'stream-segment:assistant:1', content: 'alpha' },
      { id: 'stream-segment:assistant:2', content: 'omega' },
    ]);
    expect(view.streamingThinkingSegments).toEqual([
      { id: 'stream-segment:thinking:1', content: 'thinking...' },
    ]);
    expect(state.streamingItemOrder).toEqual([
      'thinking:stream-segment:thinking:1',
      'assistant:stream-segment:assistant:1',
      'tool:tool-1',
      'assistant:stream-segment:assistant:2',
      'question:q-1',
    ]);
    expect(view.streamingTools).toEqual([
      {
        id: 'tool-1',
        content: 'running',
        toolStatus: 'running',
      },
    ]);
    expect(view.pendingQuestions).toEqual([
      {
        id: 'question-1',
        kind: 'pending_question',
        content: 'continue?',
        questionId: 'q-1',
        sessionId: 'session-1',
      },
    ]);
  });

  it('clears and resets chat timeline when clear_messages is dispatched', () => {
    let state = createInitialState();
    state = chatStateReducer(state, {
      type: 'set_scalar',
      key: 'historySyncing',
      value: true,
    });
    state = chatStateReducer(state, {
      type: 'set_scalar',
      key: 'hasOlderHistory',
      value: true,
    });
    state = chatStateReducer(state, {
      type: 'append_error_message',
      messageText: 'boom',
    });
    state = chatStateReducer(state, {
      type: 'apply_runtime_actions',
      actions: [
        { type: 'append_streaming_assistant_text', text: 'partial' },
        { type: 'append_streaming_thinking_text', text: 'thinking...' },
      ],
    });
    state = chatStateReducer(state, { type: 'clear_messages' });

    const view = buildChatStateView(state);
    expect(state.chatError).toBe('');
    expect(state.committedMessages).toEqual([]);
    expect(state.historySyncing).toBe(false);
    expect(state.hasOlderHistory).toBe(false);
    expect(state.nextHistoryBefore).toBeNull();
    expect(view.streamingAssistantSegments).toEqual([]);
    expect(view.streamingThinkingSegments).toEqual([]);
    expect(view.streamingTools).toEqual([]);
    expect(view.pendingQuestions).toEqual([]);
  });

  it('finalizes a streaming turn in top-to-bottom thinking and tool order', () => {
    let state = createInitialState();
    state = chatStateReducer(state, {
      type: 'apply_runtime_actions',
      actions: [
        { type: 'append_streaming_thinking_text', text: 'before tool' },
        {
          type: 'upsert_streaming_tool',
          tool: {
            id: 'tool-1',
            content: 'ls -la',
            toolCallId: 'call-1',
            toolStatus: 'success',
          },
        },
        { type: 'append_streaming_thinking_text', text: 'after tool' },
        {
          type: 'finalize_streaming_turn',
          assistantMessageId: 'stream-assistant:trace-1',
          assistantText: 'final answer',
        },
      ],
    });

    expect(state.committedMessages).toEqual([
      {
        id: 'stream-segment:thinking:1',
        kind: 'thinking',
        content: 'before tool',
      },
      {
        id: 'tool-1',
        kind: 'tool',
        content: 'ls -la',
        toolCallId: 'call-1',
        toolStatus: 'success',
        toolName: undefined,
        traceId: undefined,
      },
      {
        id: 'stream-segment:thinking:2',
        kind: 'thinking',
        content: 'after tool',
      },
      {
        id: 'stream-assistant:trace-1',
        kind: 'assistant',
        content: 'final answer',
      },
    ]);
    expect(state.streamingItemOrder).toEqual([]);
    const view = buildChatStateView(state);
    expect(view.streamingAssistantSegments).toEqual([]);
    expect(view.streamingThinkingSegments).toEqual([]);
    expect(view.streamingTools).toEqual([]);
  });

  it('preserves bash_exec tool input when a later update replaces the visible output', () => {
    let state = createInitialState();
    state = chatStateReducer(state, {
      type: 'apply_runtime_actions',
      actions: [
        {
          type: 'upsert_streaming_tool',
          tool: {
            id: 'tool-bash',
            content: '{"command":"echo ok"}',
            toolInput: '{"command":"echo ok"}',
            toolName: 'bash_exec',
            toolStatus: 'running',
          },
        },
        {
          type: 'upsert_streaming_tool',
          tool: {
            id: 'tool-bash',
            content: 'ok',
            toolName: 'bash_exec',
            toolStatus: 'success',
          },
        },
      ],
    });

    expect(buildChatStateView(state).streamingTools[0]).toMatchObject({
      id: 'tool-bash',
      content: 'ok',
      toolInput: '{"command":"echo ok"}',
    });
  });

  it('hydrates streaming draft state from server turn_draft', () => {
    const state = chatStateReducer(createInitialState(), {
      type: 'hydrate_turn_draft',
      sessionId: 'session-9',
      draft: {
        trace_id: 'trace-draft',
        turn: 3,
        status: 'awaiting_human',
        pending_questions: [{
          question_id: 'q-1',
          prompt: 'Ship it?',
          selection_mode: 'single',
        }],
        assistant_segments: [{ id: 'stream-segment:assistant:1', content: 'partial answer' }],
        thinking_segments: [{ id: 'stream-segment:thinking:1', content: 'analysis' }],
        tools: [{
          id: 'stream-tool:trace-draft:call-1',
          content: '{"path":"README.md"}',
          tool_name: 'read_file',
          tool_status: 'running',
          tool_call_id: 'call-1',
          trace_id: 'trace-draft',
        }],
        item_order: [
          'thinking:stream-segment:thinking:1',
          'tool:stream-tool:trace-draft:call-1',
          'assistant:stream-segment:assistant:1',
          'question:q-1',
        ],
      },
    });

    const view = buildChatStateView(state);
    expect(view.streamingThinkingSegments).toEqual([
      { id: 'stream-segment:thinking:1', content: 'analysis' },
    ]);
    expect(view.streamingAssistantSegments).toEqual([
      { id: 'stream-segment:assistant:1', content: 'partial answer' },
    ]);
    expect(view.streamingTools).toEqual([
      {
        id: 'stream-tool:trace-draft:call-1',
        content: '{"path":"README.md"}',
        toolName: 'read_file',
        toolStatus: 'running',
        toolCallId: 'call-1',
        traceId: 'trace-draft',
      },
    ]);
    expect(state.streamingItemOrder).toEqual([
      'thinking:stream-segment:thinking:1',
      'tool:stream-tool:trace-draft:call-1',
      'assistant:stream-segment:assistant:1',
      'question:q-1',
    ]);
    expect(view.pendingQuestions).toEqual([
      {
        id: 'stream-question:trace-draft:q-1',
        kind: 'pending_question',
        content: 'Ship it?',
        questionId: 'q-1',
        selectionMode: 'single',
        sessionId: 'session-9',
      },
    ]);
  });
});

function createInitialState(): ChatStateStore {
  return createInitialChatState();
}
