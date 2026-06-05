import type {
  PendingQuestionMessage,
  StreamingAssistantSegment,
  StreamingThinkingSegment,
  StreamingToolState,
} from '@/lib/types';
import { getOrderedStreamingRows } from './streamingRows';

function buildTool(id: string): StreamingToolState {
  return {
    id,
    content: `${id} running`,
    toolName: 'script_exec',
    toolStatus: 'running',
    traceId: 'trace-1',
  };
}

function buildQuestion(questionId: string, id: string): PendingQuestionMessage {
  return {
    id,
    kind: 'pending_question',
    content: 'continue?',
    questionId,
    sessionId: 'session-1',
    selectionMode: 'single',
  };
}

function buildAssistantSegment(id: string, content = 'assistant text'): StreamingAssistantSegment {
  return {
    id,
    content,
  };
}

function buildThinkingSegment(id: string, content = 'thinking text'): StreamingThinkingSegment {
  return {
    id,
    content,
  };
}

describe('lib/chat-view/streamingRows', () => {
  it('renders streaming rows in explicit event order', () => {
    const rows = getOrderedStreamingRows({
      pendingQuestions: [buildQuestion('q-1', 'question-1')],
      streamingAssistantSegments: [buildAssistantSegment('assistant-segment-1')],
      streamingThinkingSegments: [buildThinkingSegment('thinking-segment-1')],
      streamingItemOrder: [
        'tool:tool-2',
        'thinking:thinking-segment-1',
        'assistant:assistant-segment-1',
        'question:q-1',
        'tool:tool-1',
      ],
      streamingTools: [buildTool('tool-1'), buildTool('tool-2')],
    });

    expect(rows.map((row) => row.key)).toEqual([
      'tool-2',
      'thinking-segment-1',
      'assistant-segment-1',
      'question-1',
      'tool-1',
    ]);
  });

  it('appends missing rows when order trail is incomplete', () => {
    const rows = getOrderedStreamingRows({
      pendingQuestions: [buildQuestion('q-1', 'question-1')],
      streamingAssistantSegments: [buildAssistantSegment('assistant-segment-1')],
      streamingThinkingSegments: [buildThinkingSegment('thinking-segment-1')],
      streamingItemOrder: [],
      streamingTools: [buildTool('tool-1')],
    });

    expect(rows.map((row) => row.key)).toEqual([
      'assistant-segment-1',
      'thinking-segment-1',
      'tool-1',
      'question-1',
    ]);
  });

  it('skips stale order entries that no longer exist in state', () => {
    const rows = getOrderedStreamingRows({
      pendingQuestions: [],
      streamingAssistantSegments: [],
      streamingThinkingSegments: [],
      streamingItemOrder: ['assistant:missing', 'thinking:missing', 'tool:missing', 'question:missing'],
      streamingTools: [],
    });

    expect(rows).toEqual([]);
  });

  it('preserves toolInput on streaming tool rows', () => {
    const rows = getOrderedStreamingRows({
      pendingQuestions: [],
      streamingAssistantSegments: [],
      streamingThinkingSegments: [],
      streamingItemOrder: ['tool:tool-bash'],
      streamingTools: [{
        id: 'tool-bash',
        content: '{"command":"which agent-browser"}',
        toolInput: '{"command":"which agent-browser"}',
        toolName: 'bash_exec',
        toolStatus: 'running',
        traceId: 'trace-bash',
      }],
    });

    expect(rows).toHaveLength(1);
    expect(rows[0].message).toMatchObject({
      kind: 'tool',
      toolInput: '{"command":"which agent-browser"}',
      toolName: 'bash_exec',
    });
  });

  it('marks streaming assistant rows as in progress', () => {
    const rows = getOrderedStreamingRows({
      pendingQuestions: [],
      streamingAssistantSegments: [buildAssistantSegment('assistant-segment-1', 'partial answer')],
      streamingThinkingSegments: [],
      streamingItemOrder: ['assistant:assistant-segment-1'],
      streamingTools: [],
    });

    expect(rows).toMatchObject([
      {
        message: {
          kind: 'assistant',
          content: 'partial answer',
          inProgress: true,
        },
      },
    ]);
  });
});
