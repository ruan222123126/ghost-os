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
          { initialLocale: 'en-US' },
          React.createElement(HookProbe, {
            beginHistorySync: (sessionId) => events.push(`beginHistorySync:${sessionId}`),
            endHistorySync: (sessionId) => events.push(`endHistorySync:${sessionId}`),
            getCurrentSessionId: () => 'session-error',
            onRender: (state) => {
              latestState = state;
            },
            setChatError: () => undefined,
            syncRecentHistory: async (sessionId) => {
              events.push(`syncRecentHistory:${sessionId}`);
            },
          }),
        ),
      );
      await Promise.resolve();
    });

    await act(async () => {
      await expect(latestState!.runAgentStream({
        agentRuntime: 'ghost',
        message: 'hello',
        traceId: 'trace-error',
      })).rejects.toThrow('stream failed');
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(events).toEqual([
      'beginHistorySync:session-error',
      'syncRecentHistory:session-error',
      'endHistorySync:session-error',
    ]);
  });

  it('does not start a second history sync for an aborted stopped stream', async () => {
    const events: string[] = [];
    const abortController = new AbortController();
    abortController.abort();
    mockedStreamMessage.mockRejectedValue(new DOMException('aborted', 'AbortError'));

    let latestState: HookRenderState | null = null;
    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          { initialLocale: 'en-US' },
          React.createElement(HookProbe, {
            beginHistorySync: (sessionId) => events.push(`beginHistorySync:${sessionId}`),
            endHistorySync: (sessionId) => events.push(`endHistorySync:${sessionId}`),
            getCurrentSessionId: () => 'session-stop',
            onRender: (state) => {
              latestState = state;
            },
            setChatError: () => undefined,
            syncRecentHistory: async (sessionId) => {
              events.push(`syncRecentHistory:${sessionId}`);
            },
          }),
        ),
      );
      await Promise.resolve();
    });

    await act(async () => {
      await expect(latestState!.runAgentStream({
        agentRuntime: 'ghost',
        message: 'hello',
        signal: abortController.signal,
        traceId: 'trace-stop',
      })).rejects.toThrow('aborted');
    });

    expect(events).toEqual([]);
  });

  it('migrates draft runs to the resolved session and routes deltas there', async () => {
    const events: string[] = [];
    mockedStreamMessage.mockImplementation(async (options) => {
      await options.onEvent(buildRunStartedEvent('session-resolved'));
      await options.onEvent(buildTextDeltaEvent('session-resolved', 'hello'));
      return {
        sessionEnded: false,
        sessionId: 'session-resolved',
      };
    });

    let latestState: HookRenderState | null = null;
    await act(async () => {
      TestRenderer.create(
        React.createElement(
          WebLocaleProvider,
          { initialLocale: 'en-US' },
          React.createElement(HookProbe, {
            applyRuntimeActions: (sessionId, actions) => {
              if (actions.length > 0) {
                events.push(`runtime:${sessionId}:${actions[0].type}`);
              }
            },
            beginHistorySync: (sessionId) => events.push(`beginHistorySync:${sessionId}`),
            endHistorySync: (sessionId) => events.push(`endHistorySync:${sessionId}`),
            getCurrentSessionId: () => '',
            migrateSessionState: (fromSessionId, toSessionId) => {
              events.push(`migrate:${fromSessionId}->${toSessionId}`);
            },
            onRender: (state) => {
              latestState = state;
            },
            onSessionResolved: (sessionId) => events.push(`onSessionResolved:${sessionId}`),
            setChatError: () => undefined,
            syncRecentHistory: async (sessionId) => {
              events.push(`syncRecentHistory:${sessionId}`);
            },
          }),
        ),
      );
      await Promise.resolve();
    });

    await act(async () => {
      await expect(latestState!.runAgentStream({
        agentRuntime: 'ghost',
        message: 'hello',
        traceId: 'trace-resolved',
      })).resolves.toEqual({
        sessionId: 'session-resolved',
        terminalType: '',
      });
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(events).toEqual([
      'migrate:->session-resolved',
      'onSessionResolved:session-resolved',
      'runtime:session-resolved:append_streaming_assistant_text',
      'beginHistorySync:session-resolved',
      'syncRecentHistory:session-resolved',
      'endHistorySync:session-resolved',
    ]);
  });
});

interface HookProbeProps {
  applyRuntimeActions?: HookProps['applyRuntimeActions'];
  beginHistorySync: HookProps['beginHistorySync'];
  endHistorySync: HookProps['endHistorySync'];
  getCurrentSessionId: HookProps['getCurrentSessionId'];
  migrateSessionState?: HookProps['migrateSessionState'];
  onRender: (state: HookRenderState) => void;
  onSessionResolved?: HookProps['onSessionResolved'];
  setChatError: HookProps['setChatError'];
  syncRecentHistory: HookProps['syncRecentHistory'];
}

function noopApplyRuntimeActions() {}

function noopMigrateSessionState() {}

function noopSessionResolved() {}

function HookProbe({
  applyRuntimeActions = noopApplyRuntimeActions,
  beginHistorySync,
  endHistorySync,
  getCurrentSessionId,
  migrateSessionState = noopMigrateSessionState,
  onRender,
  onSessionResolved = noopSessionResolved,
  setChatError,
  syncRecentHistory,
}: HookProbeProps) {
  const state = useChatStreamController({
    applyRuntimeActions,
    beginHistorySync,
    endHistorySync,
    getCurrentSessionId,
    migrateSessionState,
    onSessionResolved,
    setChatError,
    syncRecentHistory,
  });
  onRender(state);
  return null;
}

function buildRunStartedEvent(sessionId: string): AgentStreamEvent {
  return {
    id: 'trace-resolved:000001',
    step_id: 'turn-0000-start',
    trace_id: 'trace-resolved',
    session_id: sessionId,
    turn: 0,
    type: 'run_started',
    payload: {
      session_id: sessionId,
    },
  };
}

function buildTextDeltaEvent(sessionId: string, text: string): AgentStreamEvent {
  return {
    id: 'trace-resolved:000002',
    step_id: 'turn-0000-assistant',
    trace_id: 'trace-resolved',
    session_id: sessionId,
    turn: 0,
    type: 'completion_delta',
    payload: {
      kind: 'text',
      text,
    },
  };
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
