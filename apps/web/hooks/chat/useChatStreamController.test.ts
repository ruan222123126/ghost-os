import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { streamHumanResponse, streamMessage } from '@/lib/api/agent/stream';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { AgentStreamEvent } from '@/lib/types';
import { useChatStreamController } from './useChatStreamController';

jest.mock('@/lib/api/agent/stream', () => ({
  streamHumanResponse: jest.fn(),
  streamMessage: jest.fn(),
}));

const mockedStreamMessage = streamMessage as jest.MockedFunction<typeof streamMessage>;
const mockedStreamHumanResponse = streamHumanResponse as jest.MockedFunction<typeof streamHumanResponse>;

describe('hooks/chat/useChatStreamController', () => {
  beforeEach(() => {
    jest.resetAllMocks();
  });

  it('syncs persisted history after an agent stream terminal error', async () => {
    const events: string[] = [];
    mockedStreamMessage.mockImplementation(async (options) => {
      await options.onEvent(buildErrorEvent('session-error'));
      throw new Error('stream failed');
    });

    let latestState: HookRenderState | null = null;
    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          {
            initialLocale: 'en-US',
            children: React.createElement(HookProbe, {
              activeRunRef: { current: null },
              beginHistorySync: () => events.push('beginHistorySync'),
              clearStreamingState: () => events.push('clearStreamingState'),
              currentSessionId: 'session-error',
              endHistorySync: () => events.push('endHistorySync'),
              onRender: (state) => {
                latestState = state;
              },
              setActiveRun: () => undefined,
              setChatError: () => undefined,
              syncRecentHistory: async (sessionId) => {
                events.push(`syncRecentHistory:${sessionId}`);
              },
            }),
          },
        ),
      );
      await Promise.resolve();
    });

    await act(async () => {
      await expect(latestState!.runAgentStream({
        message: 'hello',
        traceId: 'trace-error',
      })).rejects.toThrow('stream failed');
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(events).toEqual([
      'clearStreamingState',
      'beginHistorySync',
      'syncRecentHistory:session-error',
      'endHistorySync',
    ]);
  });
});

function HookProbe(props: {
  activeRunRef: HookProps['activeRunRef'];
  beginHistorySync: HookProps['beginHistorySync'];
  clearStreamingState: HookProps['clearStreamingState'];
  currentSessionId: string;
  endHistorySync: HookProps['endHistorySync'];
  onRender: (state: HookRenderState) => void;
  setActiveRun: HookProps['setActiveRun'];
  setChatError: HookProps['setChatError'];
  syncRecentHistory: HookProps['syncRecentHistory'];
}) {
  const state = useChatStreamController({
    activeRunRef: props.activeRunRef,
    applyRuntimeActions: () => undefined,
    beginHistorySync: props.beginHistorySync,
    clearStreamingState: props.clearStreamingState,
    currentSessionId: props.currentSessionId,
    endHistorySync: props.endHistorySync,
    onSessionResolved: () => undefined,
    setActiveRun: props.setActiveRun,
    setChatError: props.setChatError,
    syncRecentHistory: props.syncRecentHistory,
  });
  props.onRender(state);
  return null;
}

function buildErrorEvent(sessionId: string): AgentStreamEvent {
  return {
    id: 'trace-error:000002',
    step_id: 'turn-0000-assistant',
    trace_id: 'trace-error',
    session_id: sessionId,
    turn: 0,
    type: 'error',
    payload: {
      message: 'stream failed',
      session_id: sessionId,
    },
  };
}

type HookRenderState = ReturnType<typeof useChatStreamController>;
type HookProps = Parameters<typeof useChatStreamController>[0];

void mockedStreamHumanResponse;
