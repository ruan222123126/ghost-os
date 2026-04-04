import { useCallback } from 'react';
import { streamHumanResponse, streamMessage } from '@/lib/api/agent/stream';
import { isPureToolTagDocument } from '@/lib/toolTagText';
import type { AgentStreamEvent } from '@/lib/types';
import type { StreamAgentRunInput } from './types';
import { applyEventState } from './chatStreamControllerEvents';
import { createStreamRuntimeState, syncActiveSession } from './chatStreamControllerSession';
import type {
  StreamHumanRunOptions,
  StreamRuntimeState,
  UseChatStreamControllerOptions,
} from './chatStreamControllerTypes';

export function useChatStreamController(options: UseChatStreamControllerOptions) {
  const {
    activeRunRef,
    appendCommittedMessages,
    appendStreamingAssistantText,
    clearStreamingAssistantText,
    clearStreamingState,
    currentSessionId,
    onSessionResolved,
    setActiveRun,
    syncRecentHistory,
    upsertPendingQuestion,
    upsertStreamingTool,
  } = options;

  const syncSession = useCallback(
    async (state: StreamRuntimeState, sessionId?: string) => {
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
      await syncRecentHistory(trimmedSessionId);
      clearStreamingState();
      state.assistantBuffer = '';
    },
    [
      activeRunRef,
      clearStreamingState,
      currentSessionId,
      onSessionResolved,
      setActiveRun,
      syncRecentHistory,
    ],
  );

  const applyEvent = useCallback(
    (state: StreamRuntimeState, event: AgentStreamEvent) => {
      syncActiveSession({
        activeRunRef,
        currentSessionId,
        event,
        onSessionResolved,
        setActiveRun,
        state,
      });
      applyEventState({
        appendCommittedMessages,
        appendStreamingAssistantText,
        clearStreamingAssistantText,
        event,
        state,
        upsertPendingQuestion,
        upsertStreamingTool,
      });
    },
    [
      activeRunRef,
      appendCommittedMessages,
      appendStreamingAssistantText,
      clearStreamingAssistantText,
      currentSessionId,
      onSessionResolved,
      setActiveRun,
      upsertPendingQuestion,
      upsertStreamingTool,
    ],
  );

  const runAgentStream = useCallback(
    async (run: StreamAgentRunInput) => {
      const state = createStreamRuntimeState(run.traceId, run.sessionId);
      const result = await streamMessage({
        images: run.images,
        message: run.message,
        onEvent: async (event) => {
          applyEvent(state, event);
        },
        sessionId: run.sessionId,
        signal: run.signal,
        traceId: run.traceId,
      });
      await syncSession(state, result.sessionId || state.sessionId);
    },
    [applyEvent, syncSession],
  );

  const runHumanStream = useCallback(
    async (run: StreamHumanRunOptions) => {
      const state = createStreamRuntimeState(run.traceId, run.sessionId);
      const result = await streamHumanResponse({
        answer: run.answer,
        cancelled: run.cancelled,
        onEvent: async (event) => {
          applyEvent(state, event);
        },
        questionId: run.questionId,
        sessionId: run.sessionId,
        signal: run.signal,
        traceId: run.traceId,
      });
      await syncSession(state, result.sessionId || state.sessionId);
    },
    [applyEvent, syncSession],
  );

  return {
    runAgentStream,
    runHumanStream,
  };
}

export { isPureToolTagDocument };
