import { useCallback } from 'react';
import { streamHumanResponse, streamMessage } from '@/lib/api/agent/stream';
import { projectAgentEvent } from '@/lib/chatRuntime/eventProjector';
import { createChatRuntimeState } from '@/lib/chatRuntime/runtimeState';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import { isPureToolTagDocument } from '@/lib/toolTagText';
import type { AgentStreamEvent } from '@/lib/types';
import type { StreamAgentRunInput } from './types';
import { syncActiveSession } from './chatStreamControllerSession';
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
      const trimmedSessionId = sessionId?.trim();
      if (!trimmedSessionId) {
        return;
      }

      const currentRun = activeRunRef.current;
      if (currentRun && currentRun.sessionId !== trimmedSessionId) {
        setActiveRun({ ...currentRun, sessionId: trimmedSessionId });
      }
      if (trimmedSessionId !== currentSessionId) {
        onSessionResolved?.(trimmedSessionId);
      }
      clearStreamingState();
      runtime.assistantBuffer = '';
      syncRecentHistoryInBackground(trimmedSessionId);
    },
    [
      activeRunRef,
      clearStreamingState,
      currentSessionId,
      onSessionResolved,
      setActiveRun,
      syncRecentHistoryInBackground,
    ],
  );

  const applyEvent = useCallback(
    (runtime: ChatRuntimeState, event: AgentStreamEvent) => {
      syncActiveSession({
        activeRunRef,
        currentSessionId,
        event,
        onSessionResolved,
        runtime,
        setActiveRun,
      });
      applyRuntimeActions(projectAgentEvent({ event, runtime }));
    },
    [
      activeRunRef,
      applyRuntimeActions,
      currentSessionId,
      onSessionResolved,
      setActiveRun,
    ],
  );

  const runAgentStream = useCallback(
    async (run: StreamAgentRunInput) => {
      const runtime = createChatRuntimeState(run.traceId, run.sessionId);
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
    },
    [applyEvent, syncSession],
  );

  const runHumanStream = useCallback(
    async (run: StreamHumanRunOptions) => {
      const runtime = createChatRuntimeState(run.traceId, run.sessionId);
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
    },
    [applyEvent, syncSession],
  );

  return {
    runAgentStream,
    runHumanStream,
  };
}

export { isPureToolTagDocument };
