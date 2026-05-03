import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { stopAgent } from '@/lib/api/agent/api';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import { useChatRunControl } from './useChatRunControl';

jest.mock('@/lib/api/agent/api', () => ({
  stopAgent: jest.fn(),
}));

const mockedStopAgent = stopAgent as jest.MockedFunction<typeof stopAgent>;

describe('hooks/chat/useChatRunControl', () => {
  beforeEach(() => {
    jest.resetAllMocks();
  });

  it('syncs persisted history for the stopped session before clearing streaming state', async () => {
    const events: string[] = [];
    const abortController = new AbortController();
    const activeRunRef: ActiveRunRef = {
      current: {
        abortController,
        sessionId: '',
        traceId: 'trace-stop',
      },
    };
    const stopPendingRef: StopPendingRef = { current: false };
    mockedStopAgent.mockResolvedValue({
      status: 'stopped',
      message: 'agent run cancelled successfully',
      session_id: 'session-stop',
    });

    let latestState: HookRenderState | null = null;
    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          {
            initialLocale: 'en-US',
            children: React.createElement(HookProbe, {
              activeRunRef,
              beginHistorySync: () => events.push('beginHistorySync'),
              clearChatError: () => events.push('clearChatError'),
              clearStreamingState: () => events.push('clearStreamingState'),
              currentSessionId: '',
              endHistorySync: () => events.push('endHistorySync'),
              onRender: (state) => {
                latestState = state;
              },
              onSessionResolved: (sessionId) => events.push(`onSessionResolved:${sessionId}`),
              setActiveRun: () => undefined,
              setChatError: () => undefined,
              setLoading: () => undefined,
              setStopPending: () => undefined,
              stopPendingRef,
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
      await latestState!.stopCurrentRun();
    });

    expect(mockedStopAgent).toHaveBeenCalledWith(undefined, 'trace-stop');
    expect(abortController.signal.aborted).toBe(true);
    expect(events).toEqual([
      'clearChatError',
      'onSessionResolved:session-stop',
      'beginHistorySync',
      'syncRecentHistory:session-stop',
      'clearStreamingState',
      'endHistorySync',
    ]);
  });
});

function HookProbe(props: {
  activeRunRef: ActiveRunRef;
  beginHistorySync: () => void;
  clearChatError: () => void;
  clearStreamingState: () => void;
  currentSessionId: string;
  endHistorySync: () => void;
  onRender: (state: HookRenderState) => void;
  onSessionResolved: (sessionId: string) => void;
  setActiveRun: (value: ActiveRunRef['current']) => void;
  setChatError: (value: string) => void;
  setLoading: (value: boolean) => void;
  setStopPending: (value: boolean) => void;
  stopPendingRef: StopPendingRef;
  syncRecentHistory: (sessionId: string) => Promise<void>;
}) {
  const state = useChatRunControl({
    appendErrorMessage: () => undefined,
    appendCommittedMessages: () => undefined,
    beginHistorySync: props.beginHistorySync,
    clearChatError: props.clearChatError,
    clearStreamingState: props.clearStreamingState,
    currentSessionId: props.currentSessionId,
    endHistorySync: props.endHistorySync,
    onSessionResolved: props.onSessionResolved,
    runAgentStream: async () => undefined,
    activeRunRef: props.activeRunRef,
    setActiveRun: props.setActiveRun,
    setLoading: props.setLoading,
    setStopPending: props.setStopPending,
    setChatError: props.setChatError,
    stopPendingRef: props.stopPendingRef,
    syncRecentHistory: props.syncRecentHistory,
  });
  props.onRender(state);
  return null;
}

type HookRenderState = ReturnType<typeof useChatRunControl>;

interface ActiveRunRef {
  current: {
    abortController?: AbortController;
    sessionId: string;
    traceId: string;
  } | null;
}

interface StopPendingRef {
  current: boolean;
}
