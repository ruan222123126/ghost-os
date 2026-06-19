import { useCallback } from 'react';
import { stopAgent } from '@/lib/api/agent/api';
import { createClientTraceId } from '@/lib/api/trace';
import { draftImagesToChatImages, draftImagesToSessionImages } from '@/lib/chatImageDrafts';
import { buildUserMessage } from '@/lib/chatMessages';
import { isAbortError, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import { buildAgentMessageWithSelectedSkill } from '@/lib/selectedSkillMessage';
import type { ChatSendInput } from '@/lib/types';
import type {
  ChatStateControls,
  ChatStreamRunResult,
  StreamAgentRunInput,
  UseBridgeChatOptions,
} from './types';

interface UseChatRunControlOptions {
  appendErrorMessage: ChatStateControls['appendErrorMessage'];
  appendCommittedMessages: ChatStateControls['appendCommittedMessages'];
  beginHistorySync: ChatStateControls['beginHistorySync'];
  clearChatError: ChatStateControls['clearChatError'];
  clearStreamingState: ChatStateControls['clearStreamingState'];
  currentSessionId: UseBridgeChatOptions['currentSessionId'];
  endHistorySync: ChatStateControls['endHistorySync'];
  getCurrentSessionId: () => string;
  getActiveRun: ChatStateControls['getActiveRun'];
  getStopPending: ChatStateControls['getStopPending'];
  hasPendingQuestionInSession: ChatStateControls['hasPendingQuestionInSession'];
  markBackgroundCompleted: ChatStateControls['markBackgroundCompleted'];
  migrateSessionState: ChatStateControls['migrateSessionState'];
  onSessionResolved: UseBridgeChatOptions['onSessionResolved'];
  runAgentStream: (run: StreamAgentRunInput) => Promise<ChatStreamRunResult>;
  resolveActiveRunSessionId: ChatStateControls['resolveActiveRunSessionId'];
  setActiveRun: ChatStateControls['setActiveRun'];
  setLoading: ChatStateControls['setLoading'];
  setStopPending: ChatStateControls['setStopPending'];
  setChatError: ChatStateControls['setChatError'];
  syncRecentHistory: (sessionId: string) => Promise<void>;
}

export function useChatRunControl(options: UseChatRunControlOptions) {
  const { copy } = useWebLocale();
  const {
    appendErrorMessage,
    appendCommittedMessages,
    beginHistorySync,
    clearChatError,
    clearStreamingState,
    currentSessionId,
    endHistorySync,
    getCurrentSessionId,
    getActiveRun,
    getStopPending,
    hasPendingQuestionInSession,
    markBackgroundCompleted,
    migrateSessionState,
    onSessionResolved,
    runAgentStream,
    resolveActiveRunSessionId,
    setActiveRun,
    setLoading,
    setStopPending,
    setChatError,
    syncRecentHistory,
  } = options;

  const syncStoppedRunHistory = useCallback(async (sessionId: string) => {
    const trimmedSessionId = sessionId.trim();
    if (!trimmedSessionId) {
      return;
    }
    if (trimmedSessionId !== getCurrentSessionId().trim()) {
      onSessionResolved?.(trimmedSessionId);
    }

    beginHistorySync(trimmedSessionId);
    try {
      await syncRecentHistory(trimmedSessionId);
    } catch (error) {
      setChatError(trimmedSessionId, toErrorMessage(error, copy.system.genericRequestFailed));
    } finally {
      endHistorySync(trimmedSessionId);
    }
  }, [
    beginHistorySync,
    copy.system.genericRequestFailed,
    endHistorySync,
    getCurrentSessionId,
    onSessionResolved,
    setChatError,
    syncRecentHistory,
  ]);

  const sendChatMessage = useCallback(async (input: ChatSendInput) => {
    if (!hasSendPayload(input)) {
      return;
    }

    const sessionId = currentSessionId.trim();
    const traceId = createClientTraceId('agent-run');
    const abortController = new AbortController();
    clearChatError(sessionId);
    clearStreamingState(sessionId);
    setStopPending(sessionId, false);
    setActiveRun(sessionId, { abortController, sessionId, traceId });
    const agentMessage = buildAgentMessageWithSelectedSkill(input);
    appendCommittedMessages(sessionId, [buildUserMessage(input.message, {
      id: `local:user:${traceId}`,
      images: draftImagesToChatImages(input.images),
      selectedSkill: input.selectedSkill,
    })]);
    setLoading(sessionId, true);

    try {
      const result = await runAgentStream({
        images: draftImagesToSessionImages(input.images),
        message: agentMessage,
        sessionId: sessionId || undefined,
        signal: abortController.signal,
        traceId,
      });
      if (shouldMarkBackgroundCompleted({
        currentSessionId: getCurrentSessionId(),
        hasPendingQuestion: hasPendingQuestionInSession(result.sessionId),
        initialSessionId: sessionId,
        result,
      })) {
        markBackgroundCompleted(result.sessionId);
      }
    } catch (error) {
      const targetSessionId = resolveActiveRunSessionId(traceId, sessionId);
      if (!shouldSuppressRunError(error, getStopPending(targetSessionId), abortController.signal.aborted)) {
        appendErrorMessage(targetSessionId, toErrorMessage(error, copy.system.genericRequestFailed));
      }
    } finally {
      const targetSessionId = resolveActiveRunSessionId(traceId, sessionId);
      setLoading(targetSessionId, false);
      setActiveRun(targetSessionId, null);
      setStopPending(targetSessionId, false);
    }
  }, [
    copy.system.genericRequestFailed,
    appendCommittedMessages,
    appendErrorMessage,
    clearChatError,
    clearStreamingState,
    currentSessionId,
    getCurrentSessionId,
    getStopPending,
    hasPendingQuestionInSession,
    markBackgroundCompleted,
    resolveActiveRunSessionId,
    runAgentStream,
    setActiveRun,
    setLoading,
    setStopPending,
  ]);

  const stopCurrentRun = useCallback(async () => {
    const sessionId = currentSessionId.trim();
    const run = getActiveRun(sessionId);
    if (!run || getStopPending(sessionId)) {
      return;
    }

    clearChatError(sessionId);
    setStopPending(sessionId, true);
    let stopAccepted = false;
    let targetSessionId = sessionId;
    try {
      const response = await stopAgent(run.sessionId || undefined, run.traceId || undefined);
      stopAccepted = true;
      run.abortController?.abort();
      targetSessionId = resolveStopSessionId(response.session_id, run.sessionId);
      if (targetSessionId && targetSessionId !== run.sessionId) {
        migrateSessionState(run.sessionId, targetSessionId);
      }
      await syncStoppedRunHistory(targetSessionId);
    } catch (error) {
      setChatError(targetSessionId, toErrorMessage(error, copy.system.genericRequestFailed));
    } finally {
      if (stopAccepted) {
        setLoading(targetSessionId, false);
        setActiveRun(targetSessionId, null);
      }
      setStopPending(targetSessionId, false);
    }
  }, [
    clearChatError,
    copy.system.genericRequestFailed,
    currentSessionId,
    getActiveRun,
    getStopPending,
    migrateSessionState,
    setChatError,
    setActiveRun,
    setLoading,
    setStopPending,
    syncStoppedRunHistory,
  ]);

  return {
    sendChatMessage,
    stopCurrentRun,
  };
}

function hasSendPayload(input: ChatSendInput): boolean {
  return input.message.trim().length > 0 || input.images.length > 0 || input.selectedSkill !== undefined;
}

function shouldSuppressRunError(error: unknown, stopPending: boolean, streamAborted: boolean): boolean {
  if (isAbortError(error)) {
    return true;
  }

  const message = toErrorMessage(error);
  if (!stopPending && !streamAborted) {
    return false;
  }

  return message === 'agent stream closed before terminal event'
    || message === 'agent run cancelled';
}

function shouldMarkBackgroundCompleted(input: {
  currentSessionId: string;
  hasPendingQuestion: boolean;
  initialSessionId: string;
  result: ChatStreamRunResult;
}): boolean {
  const completedSessionId = input.result.sessionId.trim();
  if (!completedSessionId || input.result.terminalType !== 'done' || input.hasPendingQuestion) {
    return false;
  }

  const currentSessionId = input.currentSessionId.trim();
  if (currentSessionId === completedSessionId) {
    return false;
  }
  return input.initialSessionId.trim().length > 0 || currentSessionId.length > 0;
}

function resolveStopSessionId(stoppedSessionId?: string, activeSessionId?: string): string {
  const trimmedStoppedSessionId = stoppedSessionId?.trim();
  if (trimmedStoppedSessionId) {
    return trimmedStoppedSessionId;
  }
  return activeSessionId?.trim() ?? '';
}
