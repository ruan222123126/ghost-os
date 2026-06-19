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
