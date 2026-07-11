import type { ChatMessage } from '@/lib/types';
import { buildMessageListProjection } from './messageRows';
import type { MessageListMessageRow } from './types';

describe('lib/chat-view/messageRows', () => {
  it('projects visible committed, streaming, and indicator rows in display order', () => {
    const committedMessages: ChatMessage[] = [
      { id: 'system-1', kind: 'system', content: 'prompt', sourceRole: 'system' },
      { id: 'assistant-1', kind: 'assistant', content: 'done' },
    ];

    const projection = buildMessageListProjection({
      committedMessages,
      loading: true,
      loadingOlderHistory: true,
      pendingQuestions: [],
      showSystemPromptMessages: false,
      streamingAssistantSegments: [],
      streamingItemOrder: [],
      streamingThinkingSegments: [],
      streamingTools: [],
    });

    expect(projection.visibleCommittedMessages.map((message) => message.id)).toEqual(['assistant-1']);
    expect(projection.showThinkingIndicator).toBe(true);
    expect(projection.estimatedRowSize).toBe(96);
    expect(projection.rows.map((row) => row.key)).toEqual([
      'history-loading',
      'assistant-1',
      'thinking-indicator',
    ]);
  });

  it('uses streaming item order for projected message rows', () => {
    const projection = buildMessageListProjection({
      committedMessages: [],
      loading: false,
      loadingOlderHistory: false,
      pendingQuestions: [{
        id: 'pending-1',
        kind: 'pending_question',
        content: 'continue?',
        questionId: 'question-1',
        sessionId: 'session-1',
      }],
      showSystemPromptMessages: true,
      streamingAssistantSegments: [{ id: 'assistant-stream-1', content: 'partial' }],
      streamingItemOrder: ['question:question-1', 'assistant:assistant-stream-1'],
      streamingThinkingSegments: [],
      streamingTools: [],
    });

    expect(projection.streamingRows.map((row) => row.key)).toEqual([
      'pending-1',
      'assistant-stream-1',
    ]);
    expect(projection.rows.map((row) => row.key)).toEqual([
      'pending-1',
      'assistant-stream-1',
    ]);
  });

  it('projects committed and streaming tool cards through the same view model', () => {
    const committedTool: ChatMessage = {
      id: 'tool-history',
      kind: 'tool',
      content: '{"command":"echo ok"}',
      toolInput: '{"command":"echo ok"}',
      toolName: 'bash_exec',
      toolStatus: 'success',
    };

    const projection = buildMessageListProjection({
      committedMessages: [committedTool],
      loading: false,
      loadingOlderHistory: false,
      pendingQuestions: [],
      showSystemPromptMessages: true,
      streamingAssistantSegments: [],
      streamingItemOrder: ['tool:tool-stream'],
      streamingThinkingSegments: [],
      streamingTools: [{
        id: 'tool-stream',
        content: '{"command":"echo ok"}',
        toolInput: '{"command":"echo ok"}',
        toolName: 'bash_exec',
        toolStatus: 'success',
      }],
      toolCard: {
        fallbackTitle: 'Tool',
        preparingDetails: 'Preparing',
      },
    });

    const toolCards = projection.rows
      .filter((row): row is MessageListMessageRow => row.kind === 'message' && row.message.kind === 'tool')
      .map((row) => row.toolCard);

    expect(toolCards).toEqual([
      {
        title: 'echo ok',
        tone: 'success',
        statusLabel: 'SUCCESS',
        details: '{"command":"echo ok"}',
        titleMode: 'status',
        showTerminalIcon: true,
      },
      {
        title: 'echo ok',
        tone: 'success',
        statusLabel: 'SUCCESS',
        details: '{"command":"echo ok"}',
        titleMode: 'status',
        showTerminalIcon: true,
      },
    ]);
  });

  it('projects thinking auto-collapse readiness from visible stream state', () => {
    const projection = buildMessageListProjection({
      committedMessages: [],
      loading: true,
      loadingOlderHistory: false,
      pendingQuestions: [],
      showSystemPromptMessages: true,
      streamingAssistantSegments: [{ id: 'assistant-stream-1', content: 'answer' }],
      streamingItemOrder: [],
      streamingThinkingSegments: [{ id: 'thinking-stream-1', content: 'plan' }],
      streamingTools: [],
    });

    expect(projection.shouldAutoCollapseLatestThinkingPanel).toBe(true);
  });

  it('marks the active streaming thinking row as in progress', () => {
    const projection = buildMessageListProjection({
      activeStreamingThinkingId: 'thinking-stream-1',
      committedMessages: [],
      loading: true,
      loadingOlderHistory: false,
      pendingQuestions: [],
      showSystemPromptMessages: true,
      streamingAssistantSegments: [],
      streamingItemOrder: ['thinking:thinking-stream-1'],
      streamingThinkingSegments: [{ id: 'thinking-stream-1', content: 'plan' }],
      streamingTools: [],
    });

    expect(projection.rows.find((row) => row.kind === 'message' && row.message.kind === 'thinking')).toMatchObject({
      kind: 'message',
      message: {
        kind: 'thinking',
        inProgress: true,
      },
    });
  });
});
