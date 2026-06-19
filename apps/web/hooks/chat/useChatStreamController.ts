import { useCallback } from 'react';
import { streamHumanResponse, streamMessage } from '@/lib/api/agent/stream';
import { resolveEventSessionId } from '@/lib/chat-stream/sessionEvent';
import { projectAgentEvent } from '@/lib/chatRuntime/eventProjector';
import { createChatRuntimeState } from '@/lib/chatRuntime/runtimeState';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { AgentStreamEvent } from '@/lib/types';
import type { ChatStreamRunResult, StreamAgentRunInput } from './types';
import type {
  StreamHumanRunOptions,
  ChatRuntimeState,
  UseChatStreamControllerOptions,
} from './chatStreamControllerTypes';

export function useChatStreamController(options: UseChatStreamControllerOptions) {
  const { copy } = useWebLocale();
  const {
    applyRuntimeActions,
    endHistorySync,
    getCurrentSessionId,
    migrateSessionState,
    onSessionResolved,
    setChatError,
    beginHistorySync,
    syncRecentHistory,
  } = options;

  const applySessionResolution = useCallback(
    (runtime: ChatRuntimeState, sessionId?: string) => {
      const nextSessionId = sessionId?.trim();
      if (!nextSessionId || nextSessionId === runtime.sessionId) {
        return;
      }

      const previousSessionId = runtime.sessionId;
      migrateSessionState(previousSessionId, nextSessionId);
      runtime.sessionId = nextSessionId;
      if (!previousSessionId && !getCurrentSessionId().trim()) {
        onSessionResolved?.(nextSessionId);
      }
    },
    [getCurrentSessionId, migrateSessionState, onSessionResolved],
  );

  const syncRecentHistoryInBackground = useCallback((sessionId: string) => {
    const trimmedSessionId = sessionId.trim();
    if (!trimmedSessionId) {
      return;
    }

    beginHistorySync(trimmedSessionId);
    void syncRecentHistory(trimmedSessionId)
      .catch((error) => {
        setChatError(trimmedSessionId, toErrorMessage(error, copy.system.genericRequestFailed));
      })
      .finally(() => {
        endHistorySync(trimmedSessionId);
      });
  }, [beginHistorySync, copy.system.genericRequestFailed, endHistorySync, setChatError, syncRecentHistory]);

  const syncSession = useCallback(
    (runtime: ChatRuntimeState, sessionId?: string) => {
      applySessionResolution(runtime, sessionId);
      runtime.assistantBuffer = '';
      if (!runtime.sessionId.trim()) {
        return '';
      }
      syncRecentHistoryInBackground(runtime.sessionId);
      return runtime.sessionId;
    },
    [applySessionResolution, syncRecentHistoryInBackground],
  );

  const applyEvent = useCallback(
    (runtime: ChatRuntimeState, event: AgentStreamEvent) => {
      applySessionResolution(runtime, resolveEventSessionId(event));
      applyRuntimeActions(runtime.sessionId, projectAgentEvent({ event, runtime }));
    },
    [applyRuntimeActions, applySessionResolution],
  );

  const runAgentStream = useCallback(
    async (run: StreamAgentRunInput): Promise<ChatStreamRunResult> => {
      const runtime = createChatRuntimeState(run.traceId, run.sessionId);
      let terminalType: ChatStreamRunResult['terminalType'] = '';
      try {
        const result = await streamMessage({
          images: run.images,
          message: run.message,
          onEvent: async (event) => {
            terminalType = resolveTerminalType(terminalType, event);
            applyEvent(runtime, event);
          },
          sessionId: run.sessionId,
          signal: run.signal,
          traceId: run.traceId,
        });
        return {
          sessionId: syncSession(runtime, result.sessionId || runtime.sessionId),
          terminalType,
        };
      } catch (error) {
        if (!run.signal?.aborted) {
          syncSession(runtime, runtime.sessionId);
        }
        throw error;
      }
    },
    [applyEvent, syncSession],
  );

  const runHumanStream = useCallback(
    async (run: StreamHumanRunOptions): Promise<ChatStreamRunResult> => {
      const runtime = createChatRuntimeState(run.traceId, run.sessionId);
      let terminalType: ChatStreamRunResult['terminalType'] = '';
      try {
        const result = await streamHumanResponse({
          answer: run.answer,
          cancelled: run.cancelled,
          onEvent: async (event) => {
            terminalType = resolveTerminalType(terminalType, event);
            applyEvent(runtime, event);
          },
          questionId: run.questionId,
          sessionId: run.sessionId,
          signal: run.signal,
          traceId: run.traceId,
        });
        return {
          sessionId: syncSession(runtime, result.sessionId || runtime.sessionId),
          terminalType,
        };
      } catch (error) {
        if (!run.signal?.aborted) {
          syncSession(runtime, runtime.sessionId);
        }
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

function resolveTerminalType(
  current: ChatStreamRunResult['terminalType'],
  event: AgentStreamEvent,
): ChatStreamRunResult['terminalType'] {
  if (current) {
    return current;
  }
  if (event.type === 'awaiting_human' || event.type === 'done') {
    return event.type;
  }
  return '';
}
