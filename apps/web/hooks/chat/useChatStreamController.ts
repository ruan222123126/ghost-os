import { useCallback } from 'react';
import { streamHumanResponse, streamMessage } from '@/lib/api/agent/stream';
import { projectAgentEvent } from '@/lib/chatRuntime/eventProjector';
import { createChatRuntimeState } from '@/lib/chatRuntime/runtimeState';
import {
  resolveEventSession,
  resolveStreamSession,
  type RuntimeSessionResolution,
} from '@/lib/chat-stream/runSession';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { AgentStreamEvent } from '@/lib/types';
import type { StreamAgentRunInput } from './types';
import type {
  StreamHumanRunOptions,
  ChatRuntimeState,
  UseChatStreamControllerOptions,
} from './chatStreamControllerTypes';

export function useChatStreamController(options: UseChatStreamControllerOptions) {
  const { copy } = useWebLocale();
  const {
    activeRunRef,
    applyRuntimeActions,
    clearStreamingState,
    currentSessionId,
    endHistorySync,
    onSessionResolved,
    setChatError,
    setActiveRun,
    beginHistorySync,
    syncRecentHistory,
  } = options;

  const applySessionResolution = useCallback(
    (resolution: RuntimeSessionResolution | null) => {
      if (!resolution) {
        return;
      }
      if (resolution.nextActiveRun) {
        setActiveRun(resolution.nextActiveRun);
      }
      if (resolution.notifySessionResolved) {
        onSessionResolved?.(resolution.sessionId);
      }
    },
    [onSessionResolved, setActiveRun],
  );

  const syncRecentHistoryInBackground = useCallback((sessionId: string) => {
    const trimmedSessionId = sessionId.trim();
    if (!trimmedSessionId) {
      return;
    }

    beginHistorySync();
    void syncRecentHistory(trimmedSessionId)
      .catch((error) => {
        setChatError(toErrorMessage(error, copy.system.genericRequestFailed));
      })
      .finally(() => {
        endHistorySync();
      });
  }, [beginHistorySync, copy.system.genericRequestFailed, endHistorySync, setChatError, syncRecentHistory]);

  const syncSession = useCallback(
    (runtime: ChatRuntimeState, sessionId?: string) => {
      const resolution = resolveStreamSession({
        activeRun: activeRunRef.current,
        currentSessionId,
        sessionId,
      });
      if (!resolution) {
        return;
      }

      applySessionResolution(resolution);
      clearStreamingState();
      runtime.assistantBuffer = '';
      syncRecentHistoryInBackground(resolution.sessionId);
    },
    [
      activeRunRef,
      applySessionResolution,
      clearStreamingState,
      currentSessionId,
      syncRecentHistoryInBackground,
    ],
  );

  const applyEvent = useCallback(
    (runtime: ChatRuntimeState, event: AgentStreamEvent) => {
      const resolution = resolveEventSession({
        activeRun: activeRunRef.current,
        currentSessionId,
        event,
        runtimeSessionId: runtime.sessionId,
      });
      applySessionResolution(resolution);
      if (resolution) {
        runtime.sessionId = resolution.sessionId;
      }
      applyRuntimeActions(projectAgentEvent({ event, runtime }));
    },
    [
      activeRunRef,
      applySessionResolution,
      applyRuntimeActions,
      currentSessionId,
    ],
  );

  const runAgentStream = useCallback(
    async (run: StreamAgentRunInput) => {
      const runtime = createChatRuntimeState(run.traceId, run.sessionId);
      try {
        const result = await streamMessage({
          images: run.images,
          message: run.message,
          onEvent: async (event) => {
            applyEvent(runtime, event);
          },
          sessionId: run.sessionId,
          signal: run.signal,
          traceId: run.traceId,
        });
        syncSession(runtime, result.sessionId || runtime.sessionId);
      } catch (error) {
        syncSession(runtime, runtime.sessionId);
        throw error;
      }
    },
    [applyEvent, syncSession],
  );

  const runHumanStream = useCallback(
    async (run: StreamHumanRunOptions) => {
      const runtime = createChatRuntimeState(run.traceId, run.sessionId);
      try {
        const result = await streamHumanResponse({
          answer: run.answer,
          cancelled: run.cancelled,
          onEvent: async (event) => {
            applyEvent(runtime, event);
          },
          questionId: run.questionId,
          sessionId: run.sessionId,
          signal: run.signal,
          traceId: run.traceId,
        });
        syncSession(runtime, result.sessionId || runtime.sessionId);
      } catch (error) {
        syncSession(runtime, runtime.sessionId);
        throw error;
      }
    },
    [applyEvent, syncSession],
  );

  return {
    runAgentStream,
    runHumanStream,
  };
}
