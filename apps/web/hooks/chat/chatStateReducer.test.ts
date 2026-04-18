import {
  chatStateReducer,
  createInitialChatState,
  type ChatStateStore,
} from './chatStateReducer';
import { buildChatStateView } from './chatStateRuntime';

describe('hooks/chat/chatStateReducer', () => {
  it('keeps streaming order and boundaries when applying runtime actions', () => {
    let state = createInitialState();
    state = chatStateReducer(state, {
      type: 'apply_runtime_actions',
      actions: [
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
    expect(state.streamingItemOrder).toEqual([
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
      actions: [{ type: 'append_streaming_assistant_text', text: 'partial' }],
    });
    state = chatStateReducer(state, { type: 'clear_messages' });

    const view = buildChatStateView(state);
    expect(state.chatError).toBe('');
    expect(state.committedMessages).toEqual([]);
    expect(state.historySyncing).toBe(false);
    expect(state.hasOlderHistory).toBe(false);
    expect(state.nextHistoryBefore).toBeNull();
    expect(view.streamingAssistantSegments).toEqual([]);
    expect(view.streamingTools).toEqual([]);
    expect(view.pendingQuestions).toEqual([]);
  });
});

function createInitialState(): ChatStateStore {
  return createInitialChatState();
}
