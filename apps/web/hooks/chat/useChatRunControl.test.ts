import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { stopAgent, stopExternalAgent } from '@/lib/api/agent/api';
import { WebLocaleProvider } from '@/lib/i18n/provider';
import type { ChatMessage } from '@/lib/types';
import type { StreamAgentRunInput } from './types';
import { useChatRunControl } from './useChatRunControl';

jest.mock('@/lib/api/agent/api', () => ({
  stopAgent: jest.fn(),
  stopExternalAgent: jest.fn(),
}));

const mockedStopAgent = stopAgent as jest.MockedFunction<typeof stopAgent>;
const mockedStopExternalAgent = stopExternalAgent as jest.MockedFunction<typeof stopExternalAgent>;

describe('hooks/chat/useChatRunControl', () => {
  beforeEach(() => {
    jest.resetAllMocks();
  });

  it('keeps selected skill visible locally and sends it as already loaded to the agent', async () => {
    const activeRunRef: ActiveRunRef = { current: null };
    const stopPendingRef: StopPendingRef = { current: false };
    const committed: ChatMessage[] = [];
    const streamedRuns: StreamAgentRunInput[] = [];
    let latestState: HookRenderState | null = null;

    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          { initialLocale: 'en-US' },
          React.createElement(HookProbe, {
            activeRunRef,
            appendCommittedMessages: (_sessionId, messages) => {
              committed.push(...messages);
            },
            beginHistorySync: () => undefined,
            clearChatError: () => undefined,
            clearStreamingState: () => undefined,
            currentSessionId: '',
            endHistorySync: () => undefined,
            onRender: (state) => {
              latestState = state;
            },
            onSessionResolved: () => undefined,
            runAgentStream: async (run) => {
              streamedRuns.push(run);
              return { sessionId: '', terminalType: 'done' };
            },
            setActiveRun: (_sessionId, value) => {
              activeRunRef.current = value;
            },
            setChatError: () => undefined,
            setLoading: () => undefined,
            setStopPending: (_sessionId, value) => {
              stopPendingRef.current = value;
            },
            stopPendingRef,
            syncRecentHistory: async () => undefined,
          }),
        ),
      );
      await Promise.resolve();
    });

    await act(async () => {
      await latestState!.sendChatMessage({
        images: [],
        message: 'ship release',
        selectedSkill: {
          id: 'skill_release',
          name: 'release_flow',
        },
      });
    });

    expect(committed).toEqual([
      expect.objectContaining({
        content: 'ship release',
        kind: 'user',
        selectedSkill: {
          id: 'skill_release',
          name: 'release_flow',
        },
      }),
    ]);
    expect(streamedRuns).toHaveLength(1);
    expect(streamedRuns[0].message).toContain('[Ghost-OS selected skill]');
    expect(streamedRuns[0].message).toContain('already been loaded through sfind');
    expect(streamedRuns[0].message).toContain('do not call sfind just to load or verify it');
    expect(streamedRuns[0].message).toContain('release_flow');
    expect(streamedRuns[0].message).toContain('ship release');
  });

  it('routes codex messages through the external agent stream', async () => {
    const streamedRuns: StreamAgentRunInput[] = [];
    let latestState: HookRenderState | null = null;

    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          { initialLocale: 'en-US' },
          React.createElement(HookProbe, {
            activeRunRef: { current: null },
            beginHistorySync: () => undefined,
            clearChatError: () => undefined,
            clearStreamingState: () => undefined,
            currentSessionId: '',
            endHistorySync: () => undefined,
            externalCodexPermissionMode: 'safe-yolo',
            externalProjectRoot: '/workspace/project',
            onRender: (state) => {
              latestState = state;
            },
            onSessionResolved: () => undefined,
            runAgentStream: async (run) => {
              streamedRuns.push(run);
              return { sessionId: 'session-codex', terminalType: 'done' };
            },
            setActiveRun: () => undefined,
            setChatError: () => undefined,
            setLoading: () => undefined,
            setStopPending: () => undefined,
            stopPendingRef: { current: false },
            syncRecentHistory: async () => undefined,
          }),
        ),
      );
      await Promise.resolve();
    });

    await act(async () => {
      await latestState!.sendChatMessage({
        agentRuntime: 'codex',
        images: [],
        message: 'ship release',
      });
    });

    expect(streamedRuns).toEqual([
      expect.objectContaining({
        agentRuntime: 'codex',
        message: 'ship release',
        permissionMode: 'safe-yolo',
        projectRoot: '/workspace/project',
      }),
    ]);
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

  it('stops codex runs through the external agent stop action', async () => {
    const abortController = new AbortController();
    const activeRunRef: ActiveRunRef = {
      current: {
        abortController,
        runtime: 'codex',
        sessionId: 'session-codex',
        traceId: 'trace-codex',
      },
    };
    mockedStopExternalAgent.mockResolvedValue({
      status: 'idle',
      provider: 'codex',
      session_id: 'session-codex',
    });

    let latestState: HookRenderState | null = null;
    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          { initialLocale: 'en-US' },
          React.createElement(HookProbe, {
            activeRunRef,
            beginHistorySync: () => undefined,
            clearChatError: () => undefined,
            clearStreamingState: () => undefined,
            currentSessionId: 'session-codex',
            endHistorySync: () => undefined,
            onRender: (state) => {
              latestState = state;
            },
            onSessionResolved: () => undefined,
            setActiveRun: (sessionId, value) => {
              activeRunRef.current = value;
            },
            setChatError: () => undefined,
            setLoading: () => undefined,
            setStopPending: () => undefined,
            stopPendingRef: { current: false },
            syncRecentHistory: async () => undefined,
          }),
        ),
      );
      await Promise.resolve();
    });

    await act(async () => {
      await latestState!.stopCurrentRun();
    });

    expect(mockedStopExternalAgent).toHaveBeenCalledWith('session-codex');
    expect(mockedStopAgent).not.toHaveBeenCalled();
    expect(abortController.signal.aborted).toBe(true);
  });
});

function HookProbe(props: {
  activeRunRef: ActiveRunRef;
  appendCommittedMessages?: (sessionId: string, messages: ChatMessage[]) => void;
  beginHistorySync: (sessionId: string) => void;
  clearChatError: (sessionId: string) => void;
  clearStreamingState: () => void;
  currentSessionId: string;
  endHistorySync: (sessionId: string) => void;
  externalCodexPermissionMode?: 'read-only' | 'default' | 'safe-yolo' | 'yolo';
  externalProjectRoot?: string;
  onRender: (state: HookRenderState) => void;
  onSessionResolved: (sessionId: string) => void;
  runAgentStream?: (run: StreamAgentRunInput) => Promise<{ sessionId: string; terminalType: 'awaiting_human' | 'done' | '' }>;
  setActiveRun: (sessionId: string, value: ActiveRunRef['current']) => void;
  setChatError: (sessionId: string, value: string) => void;
  setLoading: (sessionId: string, value: boolean) => void;
  setStopPending: (sessionId: string, value: boolean) => void;
  stopPendingRef: StopPendingRef;
  syncRecentHistory: (sessionId: string) => Promise<void>;
}) {
  const state = useChatRunControl({
    appendErrorMessage: () => undefined,
    appendCommittedMessages: props.appendCommittedMessages ?? (() => undefined),
    beginHistorySync: props.beginHistorySync,
    clearChatError: props.clearChatError,
    clearStreamingState: () => props.clearStreamingState(),
    currentSessionId: props.currentSessionId,
    endHistorySync: props.endHistorySync,
    externalCodexPermissionMode: props.externalCodexPermissionMode,
    externalProjectRoot: props.externalProjectRoot,
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
    runAgentStream: props.runAgentStream ?? (async () => ({ sessionId: props.currentSessionId, terminalType: 'done' })),
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
    runtime?: 'ghost' | 'codex';
    sessionId: string;
    traceId: string;
  } | null;
}

interface StopPendingRef {
  current: boolean;
}
