import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { getFullSession } from '@/lib/api/sessions/api';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { ChatMessage, SessionDetail, SessionTurnDraft } from '@/lib/types';
import { mergeLatestCommittedMessages } from './chatHistoryMerge';
import { useChatHistory } from './useChatHistory';
import { useChatState } from './useChatState';

jest.mock('@/lib/api/sessions/api', () => ({
  getFullSession: jest.fn(),
  getSession: jest.fn(),
}));

const mockedGetFullSession = getFullSession as jest.MockedFunction<typeof getFullSession>;

function buildUserMessage(id: string, content: string): ChatMessage {
  return {
    id,
    kind: 'user',
    content,
  };
}

function buildAssistantMessage(id: string, content: string): ChatMessage {
  return {
    id,
    kind: 'assistant',
    content,
  };
}

function buildToolMessage(id: string, content: string, toolCallId?: string): ChatMessage {
  return {
    id,
    kind: 'tool',
    content,
    toolCallId,
  };
}

function buildThinkingMessage(id: string, content: string): ChatMessage {
  return {
    id,
    kind: 'thinking',
    content,
  };
}

describe('hooks/chat/useChatHistory mergeLatestCommittedMessages', () => {
  beforeEach(() => {
    jest.resetAllMocks();
  });

  it('drops local and streaming placeholders once server history arrives', () => {
    const previous = [
      buildUserMessage('session:user:1', 'older question'),
      buildUserMessage('local:user:trace-1', 'current question'),
      buildThinkingMessage('stream-thinking:trace-1', 'analyzing...'),
      buildAssistantMessage('stream-assistant:trace-1', 'final answer'),
    ];

    const latest = [
      buildUserMessage('session:user:1', 'older question'),
      buildUserMessage('session:user:2', 'current question'),
      buildAssistantMessage('session:assistant:2', 'final answer'),
    ];

    const merged = mergeLatestCommittedMessages(previous, latest);

    expect(merged.map((message) => message.id)).toEqual([
      'session:user:1',
      'session:user:2',
      'session:assistant:2',
    ]);
  });

  it('preserves older paged committed messages outside the latest window', () => {
    const previous = [
      buildUserMessage('session:user:1', 'question 1'),
      buildThinkingMessage('stream-thinking:trace-1', 'reasoning 1'),
      buildAssistantMessage('session:assistant:1', 'answer 1'),
      buildUserMessage('local:user:trace-2', 'question 2'),
      buildThinkingMessage('stream-thinking:trace-2', 'reasoning 2'),
      buildAssistantMessage('stream-assistant:trace-2', 'answer 2'),
    ];

    const latest = [
      buildUserMessage('session:user:1', 'question 1'),
      buildAssistantMessage('session:assistant:1', 'answer 1'),
      buildUserMessage('session:user:2', 'question 2'),
      buildAssistantMessage('session:assistant:2', 'answer 2'),
    ];

    const merged = mergeLatestCommittedMessages(previous, latest);

    expect(merged.map((message) => message.id)).toEqual([
      'session:user:1',
      'session:assistant:1',
      'session:user:2',
      'session:assistant:2',
    ]);
  });

  it('keeps older committed tool loops but removes current streaming artifacts', () => {
    const previous = [
      buildUserMessage('session:user:old', 'old question'),
      buildToolMessage('session:tool:old', 'cat README.md', 'call-old'),
      buildAssistantMessage('session:assistant:old', 'old answer'),
      buildUserMessage('local:user:trace-3', 'question'),
      buildThinkingMessage('stream-thinking:trace-3:1', 'before tool'),
      buildToolMessage('stream-tool:trace-3:call-1', 'ls -la', 'call-1'),
      buildAssistantMessage('stream-assistant:trace-3', 'answer'),
    ];

    const latest = [
      buildUserMessage('session:user:3', 'question'),
      buildToolMessage('session:tool:3', 'ls -la', 'call-1'),
      buildAssistantMessage('session:assistant:3', 'answer'),
    ];

    const merged = mergeLatestCommittedMessages(previous, latest);

    expect(merged.map((message) => message.id)).toEqual([
      'session:user:old',
      'session:tool:old',
      'session:assistant:old',
      'session:user:3',
      'session:tool:3',
      'session:assistant:3',
    ]);
  });

  it('keeps the current assistant reply visible when synced history has only caught up to the user message', () => {
    const previous = [
      buildUserMessage('session:user:old', 'old question'),
      buildAssistantMessage('session:assistant:old', 'old answer'),
      buildUserMessage('local:user:trace-4', 'new question'),
      buildThinkingMessage('stream-thinking:trace-4', 'thinking'),
      buildAssistantMessage('stream-assistant:trace-4', 'new answer'),
    ];

    const latest = [
      buildUserMessage('session:user:old', 'old question'),
      buildAssistantMessage('session:assistant:old', 'old answer'),
      buildUserMessage('session:user:4', 'new question'),
    ];

    const merged = mergeLatestCommittedMessages(previous, latest);

    expect(merged.map((message) => message.id)).toEqual([
      'session:user:old',
      'session:assistant:old',
      'session:user:4',
      'stream-assistant:trace-4',
    ]);
  });

  it('keeps the unfinished current turn visible when synced history has not caught up yet', () => {
    const previous = [
      buildUserMessage('session:user:old', 'old question'),
      buildAssistantMessage('session:assistant:old', 'old answer'),
      buildUserMessage('local:user:trace-5', 'new question'),
      buildThinkingMessage('stream-thinking:trace-5', 'thinking'),
      buildToolMessage('stream-tool:trace-5:call-5', 'ls -la', 'call-5'),
      buildAssistantMessage('stream-assistant:trace-5', 'new answer'),
    ];

    const latest = [
      buildUserMessage('session:user:old', 'old question'),
      buildAssistantMessage('session:assistant:old', 'old answer'),
    ];

    const merged = mergeLatestCommittedMessages(previous, latest);

    expect(merged.map((message) => message.id)).toEqual([
      'session:user:old',
      'session:assistant:old',
      'local:user:trace-5',
      'stream-tool:trace-5:call-5',
      'stream-assistant:trace-5',
    ]);
  });
});

describe('hooks/chat/useChatHistory syncRecentHistory', () => {
  beforeEach(() => {
    jest.resetAllMocks();
  });

  it('merges stopped user history and hydrates the persisted assistant turn draft', async () => {
    const draft: SessionTurnDraft = {
      trace_id: 'trace-stop',
      turn: 1,
      status: 'error',
      error: 'context canceled',
      pending_questions: [],
      assistant_segments: [{ id: 'stream-segment:assistant:1', content: 'partial answer' }],
      thinking_segments: [],
      tools: [],
      item_order: ['assistant:stream-segment:assistant:1'],
    };
    mockedGetFullSession.mockResolvedValue(buildSessionDetail({
      messages: [{ index: 0, role: 'user', text: 'hello' }],
      turn_draft: draft,
    }));

    let latest: HistoryProbeState | null = null;
    await renderHistoryProbe((state) => {
      latest = state;
    });

    await act(async () => {
      latest!.state.appendCommittedMessages([buildUserMessage('local:user:trace-stop', 'hello')]);
      latest!.state.applyRuntimeActions([
        { type: 'append_streaming_assistant_text', text: 'local partial' },
      ]);
    });

    await act(async () => {
      await latest!.history.syncRecentHistory('session-stop');
    });

    expect(latest!.state.committedMessages).toEqual([
      { id: 'session:session-stop:message:0:user', kind: 'user', content: 'hello', images: undefined },
    ]);
    expect(latest!.state.streamingAssistantSegments).toEqual([
      { id: 'stream-segment:assistant:1', content: 'partial answer' },
    ]);
    expect(latest!.state.streamingItemOrder).toEqual(['assistant:stream-segment:assistant:1']);
  });

  it('keeps the stopped user message and clears streaming state when the assistant has no draft output', async () => {
    mockedGetFullSession.mockResolvedValue(buildSessionDetail({
      messages: [{ index: 0, role: 'user', text: 'hello' }],
      turn_draft: null,
    }));

    let latest: HistoryProbeState | null = null;
    await renderHistoryProbe((state) => {
      latest = state;
    });

    await act(async () => {
      latest!.state.appendCommittedMessages([buildUserMessage('local:user:trace-stop', 'hello')]);
      latest!.state.applyRuntimeActions([
        { type: 'append_streaming_assistant_text', text: 'local partial' },
      ]);
    });

    await act(async () => {
      await latest!.history.syncRecentHistory('session-stop');
    });

    expect(latest!.state.committedMessages).toEqual([
      { id: 'session:session-stop:message:0:user', kind: 'user', content: 'hello', images: undefined },
    ]);
    expect(latest!.state.streamingAssistantSegments).toEqual([]);
    expect(latest!.state.streamingThinkingSegments).toEqual([]);
    expect(latest!.state.streamingTools).toEqual([]);
  });
});

function HistoryProbe(props: {
  onRender: (state: HistoryProbeState) => void;
}) {
  const state = useChatState();
  const history = useChatHistory({
    clearChatError: state.clearChatError,
    clearPendingQuestions: state.clearPendingQuestions,
    clearStreamingState: state.clearStreamingState,
    applyRuntimeActions: state.applyRuntimeActions,
    hydrateTurnDraft: state.hydrateTurnDraft,
    replaceWithErrorMessage: state.replaceWithErrorMessage,
    setCommittedMessages: state.setCommittedMessages,
    setHasOlderHistory: state.setHasOlderHistory,
    setHistoryLoading: state.setHistoryLoading,
    setLoadingOlderHistory: state.setLoadingOlderHistory,
    setNextHistoryBefore: state.setNextHistoryBefore,
    setChatError: state.setChatError,
    setActiveRun: state.setActiveRun,
    setLoading: state.setLoading,
    setStopPending: state.setStopPending,
    beginHistorySync: state.beginHistorySync,
    endHistorySync: state.endHistorySync,
    nextHistoryBefore: state.nextHistoryBefore,
  });
  props.onRender({ history, state });
  return null;
}

async function renderHistoryProbe(onRender: (state: HistoryProbeState) => void): Promise<void> {
  await act(async () => {
    TestRenderer.create(
      React.createElement(
        WebLocaleProvider,
        { initialLocale: 'en-US' },
        React.createElement(HistoryProbe, { onRender }),
      ),
    );
    await Promise.resolve();
  });
}

function buildSessionDetail(options: {
  messages: SessionDetail['messages'];
  turn_draft?: SessionTurnDraft | null;
}): SessionDetail {
  return {
    id: 'session-stop',
    title: 'Stopped',
    created_at: '2026-06-06T00:00:00Z',
    updated_at: '2026-06-06T00:00:01Z',
    message_count: options.messages.length,
    page: {
      limit: 100,
      before: null,
      start_index: options.messages[0]?.index ?? 0,
      end_index: options.messages.at(-1)?.index ?? 0,
      has_more_before: false,
      next_before: null,
    },
    token_count: 0,
    messages: options.messages,
    turn_draft: options.turn_draft,
  };
}

interface HistoryProbeState {
  history: ReturnType<typeof useChatHistory>;
  state: ReturnType<typeof useChatState>;
}
