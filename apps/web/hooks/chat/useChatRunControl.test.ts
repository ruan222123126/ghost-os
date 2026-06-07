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
    abortController.signal.addEventListener('abort', () => {
      events.push('abort');
    });

    let latestState: HookRenderState | null = null;
    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          { initialLocale: 'en-US' },
          React.createElement(HookProbe, {
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
            setActiveRun: (value) => {
              activeRunRef.current = value;
              events.push(value ? `setActiveRun:${value.sessionId}` : 'setActiveRun:null');
            },
            setChatError: () => undefined,
            setLoading: (value) => events.push(`setLoading:${value}`),
            setStopPending: (value) => {
              stopPendingRef.current = value;
              events.push(`setStopPending:${value}`);
            },
            stopPendingRef,
            syncRecentHistory: async (sessionId) => {
              events.push(`syncRecentHistory:${sessionId}`);
            },
          }),
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
      'setStopPending:true',
      'abort',
      'onSessionResolved:session-stop',
      'beginHistorySync',
      'syncRecentHistory:session-stop',
      'endHistorySync',
      'setLoading:false',
      'setActiveRun:null',
      'setStopPending:false',
    ]);
  });

  it('keeps the optimistic turn visible and exits running state when stopped history sync fails', async () => {
    const events: string[] = [];
    const errors: string[] = [];
    const abortController = new AbortController();
    const activeRunRef: ActiveRunRef = {
      current: {
        abortController,
        sessionId: 'session-stop',
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
          { initialLocale: 'en-US' },
          React.createElement(HookProbe, {
            activeRunRef,
            beginHistorySync: () => events.push('beginHistorySync'),
            clearChatError: () => events.push('clearChatError'),
            clearStreamingState: () => events.push('clearStreamingState'),
            currentSessionId: 'session-stop',
            endHistorySync: () => events.push('endHistorySync'),
            onRender: (state) => {
              latestState = state;
            },
            onSessionResolved: (sessionId) => events.push(`onSessionResolved:${sessionId}`),
            setActiveRun: (value) => {
              activeRunRef.current = value;
              events.push(value ? `setActiveRun:${value.sessionId}` : 'setActiveRun:null');
            },
            setChatError: (value) => {
              errors.push(value);
              events.push(`setChatError:${value}`);
            },
            setLoading: (value) => events.push(`setLoading:${value}`),
            setStopPending: (value) => {
              stopPendingRef.current = value;
              events.push(`setStopPending:${value}`);
            },
            stopPendingRef,
            syncRecentHistory: async () => {
              throw new Error('history unavailable');
            },
          }),
        ),
      );
      await Promise.resolve();
    });

    await act(async () => {
      await latestState!.stopCurrentRun();
    });

    expect(errors).toEqual(['history unavailable']);
    expect(events).not.toContain('clearStreamingState');
    expect(events).toEqual([
      'clearChatError',
      'setStopPending:true',
      'beginHistorySync',
      'setChatError:history unavailable',
      'endHistorySync',
      'setLoading:false',
      'setActiveRun:null',
      'setStopPending:false',
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
