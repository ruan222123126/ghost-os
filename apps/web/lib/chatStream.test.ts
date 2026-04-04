import {
  appendStreamingAssistantState,
  clearStreamingAssistantState,
  markStreamingAssistantBoundary,
  type PendingQuestionState,
  removePendingQuestionState,
  type StreamingAssistantState,
  type StreamingToolTableState,
  upsertPendingQuestionState,
  upsertStreamingToolState,
} from './chatStream';

describe('lib/chatStream', () => {
  it('appends assistant deltas into timeline segments and splits on boundary', () => {
    let state: StreamingAssistantState = clearStreamingAssistantState();

    state = appendStreamingAssistantState(state, 'hello').state;
    state = appendStreamingAssistantState(state, ' world').state;
    state = markStreamingAssistantBoundary(state);
    state = appendStreamingAssistantState(state, 'after tool').state;

    expect(state.order).toEqual([
      'stream-segment:assistant:1',
      'stream-segment:assistant:2',
    ]);
    expect(state.segmentsById['stream-segment:assistant:1']?.content).toBe('hello world');
    expect(state.segmentsById['stream-segment:assistant:2']?.content).toBe('after tool');
  });

  it('upserts streaming tools without rewriting the order', () => {
    let state: StreamingToolTableState = {
      order: [],
      toolsById: {},
    };

    state = upsertStreamingToolState(state, {
      id: 'tool-1',
      content: 'script_exec running',
      toolCallId: 'call-1',
      toolName: 'script_exec',
      toolStatus: 'running',
      traceId: 'trace-1',
    });
    state = upsertStreamingToolState(state, {
      id: 'tool-1',
      content: 'script_exec finished',
      toolStatus: 'success',
    });

    expect(state).toEqual({
      order: ['tool-1'],
      toolsById: {
        'tool-1': {
          id: 'tool-1',
          content: 'script_exec finished',
          toolCallId: 'call-1',
          toolName: 'script_exec',
          toolStatus: 'success',
          traceId: 'trace-1',
        },
      },
    });
  });

  it('upserts and removes pending questions by question id', () => {
    let state: PendingQuestionState = {
      order: [],
      questionsById: {},
    };

    state = upsertPendingQuestionState(state, {
      id: 'question-1',
      kind: 'pending_question',
      content: 'Choose a database',
      questionId: 'q-1',
      sessionId: 'session-1',
      selectionMode: 'single',
    });
    state = upsertPendingQuestionState(state, {
      id: 'question-1',
      kind: 'pending_question',
      content: 'Choose a primary database',
      questionId: 'q-1',
      sessionId: 'session-1',
      selectionMode: 'single',
      options: [{ label: 'PostgreSQL' }],
    });
    state = removePendingQuestionState(state, 'q-1');

    expect(state).toEqual({
      order: [],
      questionsById: {},
    });
  });
});
