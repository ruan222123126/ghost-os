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
            beginHistorySync: (sessionId) => events.push(`beginHistorySync:${sessionId}`),
            clearChatError: (sessionId) => events.push(`clearChatError:${sessionId}`),
            clearStreamingState: () => events.push('clearStreamingState'),
            currentSessionId: '',
            endHistorySync: (sessionId) => events.push(`endHistorySync:${sessionId}`),
            onRender: (state) => {
              latestState = state;
            },
            onSessionResolved: (sessionId) => events.push(`onSessionResolved:${sessionId}`),
            setActiveRun: (sessionId, value) => {
              activeRunRef.current = value;
              events.push(value ? `setActiveRun:${sessionId}:${value.sessionId}` : `setActiveRun:${sessionId}:null`);
            },
            setChatError: () => undefined,
            setLoading: (sessionId, value) => events.push(`setLoading:${sessionId}:${value}`),
            setStopPending: (sessionId, value) => {
              stopPendingRef.current = value;
              events.push(`setStopPending:${sessionId}:${value}`);
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
      'clearChatError:',
      'setStopPending::true',
      'abort',
      'onSessionResolved:session-stop',
      'beginHistorySync:session-stop',
      'syncRecentHistory:session-stop',
      'endHistorySync:session-stop',
      'setLoading:session-stop:false',
      'setActiveRun:session-stop:null',
      'setStopPending:session-stop:false',
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
            beginHistorySync: (sessionId) => events.push(`beginHistorySync:${sessionId}`),
            clearChatError: (sessionId) => events.push(`clearChatError:${sessionId}`),
            clearStreamingState: () => events.push('clearStreamingState'),
            currentSessionId: 'session-stop',
            endHistorySync: (sessionId) => events.push(`endHistorySync:${sessionId}`),
            onRender: (state) => {
              latestState = state;
            },
            onSessionResolved: (sessionId) => events.push(`onSessionResolved:${sessionId}`),
            setActiveRun: (sessionId, value) => {
              activeRunRef.current = value;
              events.push(value ? `setActiveRun:${sessionId}:${value.sessionId}` : `setActiveRun:${sessionId}:null`);
            },
            setChatError: (sessionId, value) => {
              errors.push(value);
              events.push(`setChatError:${sessionId}:${value}`);
            },
            setLoading: (sessionId, value) => events.push(`setLoading:${sessionId}:${value}`),
            setStopPending: (sessionId, value) => {
              stopPendingRef.current = value;
              events.push(`setStopPending:${sessionId}:${value}`);
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
      'clearChatError:session-stop',
      'setStopPending:session-stop:true',
      'beginHistorySync:session-stop',
      'setChatError:session-stop:history unavailable',
      'endHistorySync:session-stop',
      'setLoading:session-stop:false',
      'setActiveRun:session-stop:null',
      'setStopPending:session-stop:false',
    ]);
  });
});

function HookProbe(props: {
  activeRunRef: ActiveRunRef;
  beginHistorySync: (sessionId: string) => void;
  clearChatError: (sessionId: string) => void;
  clearStreamingState: () => void;
  currentSessionId: string;
  endHistorySync: (sessionId: string) => void;
  onRender: (state: HookRenderState) => void;
  onSessionResolved: (sessionId: string) => void;
  setActiveRun: (sessionId: string, value: ActiveRunRef['current']) => void;
  setChatError: (sessionId: string, value: string) => void;
  setLoading: (sessionId: string, value: boolean) => void;
  setStopPending: (sessionId: string, value: boolean) => void;
  stopPendingRef: StopPendingRef;
  syncRecentHistory: (sessionId: string) => Promise<void>;
}) {
  const state = useChatRunControl({
    appendErrorMessage: () => undefined,
    appendCommittedMessages: () => undefined,
    beginHistorySync: props.beginHistorySync,
    clearChatError: props.clearChatError,
    clearStreamingState: () => props.clearStreamingState(),
    currentSessionId: props.currentSessionId,
    endHistorySync: props.endHistorySync,
    getCurrentSessionId: () => props.currentSessionId,
    getActiveRun: () => props.activeRunRef.current,
    getStopPending: () => props.stopPendingRef.current,
    hasPendingQuestionInSession: () => false,
    markBackgroundCompleted: () => undefined,
    migrateSessionState: (_fromSessionId, toSessionId) => {
      if (props.activeRunRef.current) {
        props.activeRunRef.current = { ...props.activeRunRef.current, sessionId: toSessionId };
      }
    },
    onSessionResolved: props.onSessionResolved,
    runAgentStream: async () => ({ sessionId: props.currentSessionId, terminalType: 'done' }),
    resolveActiveRunSessionId: () => props.activeRunRef.current?.sessionId ?? props.currentSessionId,
    setActiveRun: props.setActiveRun,
    setLoading: props.setLoading,
    setStopPending: props.setStopPending,
    setChatError: props.setChatError,
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
