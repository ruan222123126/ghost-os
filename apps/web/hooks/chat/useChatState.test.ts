import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { useChatState } from './useChatState';

describe('hooks/chat/useChatState', () => {
  it('keeps streaming and error state isolated by session', async () => {
    let latest: ChatStateProbeState | null = null;
    const renderer = TestRenderer.create(
      React.createElement(ChatStateProbe, {
        currentSessionId: 'session-a',
        onRender: (state) => {
          latest = state;
        },
      }),
    );

    await act(async () => {
      latest!.applyRuntimeActions('session-a', [
        { type: 'append_streaming_assistant_text', text: 'answer a' },
      ]);
      latest!.setChatError('session-a', 'error a');
    });
    expect(latest!.streamingAssistantSegments).toEqual([
      { id: 'stream-segment:assistant:1', content: 'answer a' },
    ]);
    expect(latest!.chatError).toBe('error a');

    await act(async () => {
      renderer.update(
        React.createElement(ChatStateProbe, {
          currentSessionId: 'session-b',
          onRender: (state) => {
            latest = state;
          },
        }),
      );
    });
    expect(latest!.streamingAssistantSegments).toEqual([]);
    expect(latest!.chatError).toBe('');

    await act(async () => {
      latest!.setChatError('session-b', 'error b');
    });
    expect(latest!.chatError).toBe('error b');

    await act(async () => {
      renderer.update(
        React.createElement(ChatStateProbe, {
          currentSessionId: 'session-a',
          onRender: (state) => {
            latest = state;
          },
        }),
      );
    });
    expect(latest!.streamingAssistantSegments).toEqual([
      { id: 'stream-segment:assistant:1', content: 'answer a' },
    ]);
    expect(latest!.chatError).toBe('error a');
  });

  it('tracks and clears background completion by session', async () => {
    let latest: ChatStateProbeState | null = null;
    TestRenderer.create(
      React.createElement(ChatStateProbe, {
        currentSessionId: 'session-b',
        onRender: (state) => {
          latest = state;
        },
      }),
    );

    await act(async () => {
      latest!.markBackgroundCompleted('session-a');
    });
    expect(latest!.backgroundCompletedSessionIds.has('session-a')).toBe(true);
    expect(latest!.backgroundCompletedSessionIds.has('session-b')).toBe(false);

    await act(async () => {
      latest!.clearBackgroundCompletion('session-a');
    });
    expect(latest!.backgroundCompletedSessionIds.has('session-a')).toBe(false);
  });

  it('migrates active run, stop pending, and history sync refs between sessions', async () => {
    let latest: ChatStateProbeState | null = null;
    const renderer = TestRenderer.create(
      React.createElement(ChatStateProbe, {
        currentSessionId: 'draft-session',
        onRender: (state) => {
          latest = state;
        },
      }),
    );

    await act(async () => {
      latest!.setActiveRun('draft-session', { sessionId: 'draft-session', traceId: 'trace-1' });
      latest!.setStopPending('draft-session', true);
      latest!.beginHistorySync('draft-session');
    });

    await act(async () => {
      latest!.migrateSessionState('draft-session', 'resolved-session');
    });

    await act(async () => {
      renderer.update(
        React.createElement(ChatStateProbe, {
          currentSessionId: 'resolved-session',
          onRender: (state) => {
            latest = state;
          },
        }),
      );
    });

    expect(latest!.activeRun).toEqual({ sessionId: 'resolved-session', traceId: 'trace-1' });
    expect(latest!.getActiveRun('draft-session')).toBeNull();
    expect(latest!.getActiveRun('resolved-session')).toEqual({ sessionId: 'resolved-session', traceId: 'trace-1' });
    expect(latest!.getStopPending('draft-session')).toBe(false);
    expect(latest!.getStopPending('resolved-session')).toBe(true);
    expect(latest!.stopPending).toBe(true);
    expect(latest!.historySyncing).toBe(true);

    await act(async () => {
      latest!.endHistorySync('resolved-session');
    });

    expect(latest!.historySyncing).toBe(false);
  });
});

function ChatStateProbe(props: {
  currentSessionId: string;
  onRender: (state: ChatStateProbeState) => void;
}) {
  const state = useChatState(props.currentSessionId);
  props.onRender(state);
  return null;
}

type ChatStateProbeState = ReturnType<typeof useChatState>;
