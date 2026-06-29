import { useCallback } from 'react';
import { streamExternalMessage, streamHumanResponse, streamMessage } from '@/lib/api/agent/stream';
import { resolveEventSessionId } from '@/lib/chat-stream/sessionEvent';
import { createChatRuntimeActionBuffer, type ChatRuntimeActionBuffer } from '@/lib/chatRuntime/actionBuffer';
import { projectAgentEvent } from '@/lib/chatRuntime/eventProjector';
import { createChatRuntimeState } from '@/lib/chatRuntime/runtimeState';
import { toErrorMessage } from '@/lib/errors';
import type { AgentStreamEvent } from '@/lib/types';
import type { ChatStreamRunResult, StreamAgentRunInput } from './types';
import type {
  ChatRuntimeState,
  StreamHumanRunOptions,
  UseChatStreamControllerOptions,
} from './chatStreamControllerTypes';

interface UseChatStreamRunnersOptions extends UseChatStreamControllerOptions {
  requestFailedText: string;
}

interface StreamSessionSyncOptions {
  applySessionResolution: (runtime: ChatRuntimeState, sessionId?: string) => void;
  syncRecentHistoryInBackground: (sessionId: string) => void;
}

interface StreamEventProjectorOptions {
  applySessionResolution: (runtime: ChatRuntimeState, sessionId?: string) => void;
}

interface StreamRunnerOptions {
  applyEvent: (runtime: ChatRuntimeState, event: AgentStreamEvent, actionBuffer: ChatRuntimeActionBuffer) => void;
  createActionBuffer: () => ChatRuntimeActionBuffer;
  syncSession: (runtime: ChatRuntimeState, sessionId?: string) => string;
}

export function useChatStreamRunners(options: UseChatStreamRunnersOptions) {
  const applySessionResolution = useSessionResolution(options);
  const syncRecentHistoryInBackground = useBackgroundHistorySync(options);
  const syncSession = useStreamSessionSync({
    applySessionResolution,
    syncRecentHistoryInBackground,
  });
  const applyEvent = useStreamEventProjector({
    applySessionResolution,
  });
  const createActionBuffer = useCallback(() => {
    return createChatRuntimeActionBuffer(options.applyRuntimeActions);
  }, [options.applyRuntimeActions]);

  return {
    runAgentStream: useAgentStreamRunner({ applyEvent, createActionBuffer, syncSession }),
    runHumanStream: useHumanStreamRunner({ applyEvent, createActionBuffer, syncSession }),
  };
}

function useSessionResolution(options: UseChatStreamRunnersOptions) {
  const { getCurrentSessionId, migrateSessionState, onSessionResolved } = options;

  return useCallback((runtime: ChatRuntimeState, sessionId?: string) => {
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
  }, [getCurrentSessionId, migrateSessionState, onSessionResolved]);
}

function useBackgroundHistorySync(options: UseChatStreamRunnersOptions) {
  const {
    beginHistorySync,
    endHistorySync,
    requestFailedText,
    setChatError,
    syncRecentHistory,
  } = options;

  return useCallback((sessionId: string) => {
    const trimmedSessionId = sessionId.trim();
    if (!trimmedSessionId) {
      return;
    }

    beginHistorySync(trimmedSessionId);
    void syncRecentHistory(trimmedSessionId)
      .catch((error) => {
        setChatError(trimmedSessionId, toErrorMessage(error, requestFailedText));
      })
      .finally(() => {
        endHistorySync(trimmedSessionId);
      });
  }, [beginHistorySync, endHistorySync, requestFailedText, setChatError, syncRecentHistory]);
}

function useStreamSessionSync(options: StreamSessionSyncOptions) {
  const { applySessionResolution, syncRecentHistoryInBackground } = options;

  return useCallback((runtime: ChatRuntimeState, sessionId?: string) => {
    applySessionResolution(runtime, sessionId);
    runtime.assistantBuffer = '';
    if (!runtime.sessionId.trim()) {
      return '';
    }
    syncRecentHistoryInBackground(runtime.sessionId);
    return runtime.sessionId;
  }, [applySessionResolution, syncRecentHistoryInBackground]);
}

function useStreamEventProjector(options: StreamEventProjectorOptions) {
  const { applySessionResolution } = options;

  return useCallback((runtime: ChatRuntimeState, event: AgentStreamEvent, actionBuffer: ChatRuntimeActionBuffer) => {
    applySessionResolution(runtime, resolveEventSessionId(event));
    actionBuffer.enqueue(runtime.sessionId, projectAgentEvent({ event, runtime }));
    if (isTerminalStreamEvent(event)) {
      actionBuffer.flush();
    }
  }, [applySessionResolution]);
}

function useAgentStreamRunner(options: StreamRunnerOptions) {
  const { applyEvent, createActionBuffer, syncSession } = options;

  return useCallback(async (run: StreamAgentRunInput): Promise<ChatStreamRunResult> => {
    const runtime = createChatRuntimeState(run.traceId, run.sessionId);
    const actionBuffer = createActionBuffer();
    let terminalType: ChatStreamRunResult['terminalType'] = '';
    try {
      const handleEvent = async (event: AgentStreamEvent) => {
        terminalType = resolveTerminalType(terminalType, event);
        applyEvent(runtime, event, actionBuffer);
      };
      const result = run.agentRuntime === 'codex'
        ? await streamExternalMessage({
          message: run.message,
          model: run.model,
          mode: run.codexMode,
          onEvent: handleEvent,
          permissionMode: run.permissionMode ?? 'default',
          projectRoot: run.projectRoot,
          sessionId: run.sessionId,
          signal: run.signal,
          traceId: run.traceId,
        })
        : await streamMessage({
          images: run.images,
          message: run.message,
          mode: run.mode,
          onEvent: handleEvent,
          sessionId: run.sessionId,
          signal: run.signal,
          traceId: run.traceId,
        });
      actionBuffer.flush();
      return {
        sessionId: syncSession(runtime, result.sessionId || runtime.sessionId),
        terminalType,
      };
    } catch (error) {
      actionBuffer.flush();
      if (!run.signal?.aborted) {
        syncSession(runtime, runtime.sessionId);
      }
      throw error;
    } finally {
      actionBuffer.cancel();
    }
  }, [applyEvent, createActionBuffer, syncSession]);
}

function useHumanStreamRunner(options: StreamRunnerOptions) {
  const { applyEvent, createActionBuffer, syncSession } = options;

  return useCallback(async (run: StreamHumanRunOptions): Promise<ChatStreamRunResult> => {
    const runtime = createChatRuntimeState(run.traceId, run.sessionId);
    const actionBuffer = createActionBuffer();
    let terminalType: ChatStreamRunResult['terminalType'] = '';
    try {
      const result = await streamHumanResponse({
        answer: run.answer,
        cancelled: run.cancelled,
        onEvent: async (event) => {
          terminalType = resolveTerminalType(terminalType, event);
          applyEvent(runtime, event, actionBuffer);
        },
        questionId: run.questionId,
        sessionId: run.sessionId,
        signal: run.signal,
        traceId: run.traceId,
      });
      actionBuffer.flush();
      return {
        sessionId: syncSession(runtime, result.sessionId || runtime.sessionId),
        terminalType,
      };
    } catch (error) {
      actionBuffer.flush();
      if (!run.signal?.aborted) {
        syncSession(runtime, runtime.sessionId);
      }
      throw error;
    } finally {
      actionBuffer.cancel();
    }
  }, [applyEvent, createActionBuffer, syncSession]);
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

function isTerminalStreamEvent(event: AgentStreamEvent): boolean {
  return event.type === 'awaiting_human'
    || event.type === 'done'
    || event.type === 'error'
    || event.type === 'message';
}
