import { useCallback } from 'react';
import { stopAgent } from '@/lib/api/agent/api';
import { createClientTraceId } from '@/lib/api/trace';
import { draftImagesToChatImages, draftImagesToSessionImages } from '@/lib/chatImageDrafts';
import { buildUserMessage } from '@/lib/chatMessages';
import { isAbortError, toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ChatSendInput } from '@/lib/types';
import type { ChatStateControls, StreamAgentRunInput, UseBridgeChatOptions } from './types';

interface UseChatRunControlOptions {
  appendErrorMessage: ChatStateControls['appendErrorMessage'];
  appendCommittedMessages: ChatStateControls['appendCommittedMessages'];
  beginHistorySync: ChatStateControls['beginHistorySync'];
  clearChatError: ChatStateControls['clearChatError'];
  clearStreamingState: ChatStateControls['clearStreamingState'];
  currentSessionId: UseBridgeChatOptions['currentSessionId'];
  endHistorySync: ChatStateControls['endHistorySync'];
  onSessionResolved: UseBridgeChatOptions['onSessionResolved'];
  runAgentStream: (run: StreamAgentRunInput) => Promise<void>;
  activeRunRef: ChatStateControls['activeRunRef'];
  setActiveRun: ChatStateControls['setActiveRun'];
  setLoading: ChatStateControls['setLoading'];
  setStopPending: ChatStateControls['setStopPending'];
  setChatError: ChatStateControls['setChatError'];
  stopPendingRef: ChatStateControls['stopPendingRef'];
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
    onSessionResolved,
    runAgentStream,
    activeRunRef,
    setActiveRun,
    setLoading,
    setStopPending,
    setChatError,
    stopPendingRef,
    syncRecentHistory,
  } = options;

  const syncStoppedRunHistory = useCallback(async (sessionId: string) => {
    const trimmedSessionId = sessionId.trim();
    if (!trimmedSessionId) {
      return;
    }
    if (trimmedSessionId !== currentSessionId) {
      onSessionResolved?.(trimmedSessionId);
    }

    beginHistorySync();
    try {
      await syncRecentHistory(trimmedSessionId);
      clearStreamingState();
    } catch (error) {
      setChatError(toErrorMessage(error, copy.system.genericRequestFailed));
    } finally {
      endHistorySync();
    }
  }, [
    beginHistorySync,
    clearStreamingState,
    copy.system.genericRequestFailed,
    currentSessionId,
    endHistorySync,
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
    clearChatError();
    clearStreamingState();
    setStopPending(false);
    setActiveRun({ abortController, sessionId, traceId });
    appendCommittedMessages([buildUserMessage(input.message, {
      id: `local:user:${traceId}`,
      images: draftImagesToChatImages(input.images),
    })]);
    setLoading(true);

    try {
      await runAgentStream({
        images: draftImagesToSessionImages(input.images),
        message: input.message,
        sessionId: sessionId || undefined,
        signal: abortController.signal,
        traceId,
      });
    } catch (error) {
      if (!shouldSuppressRunError(error, stopPendingRef.current)) {
        appendErrorMessage(toErrorMessage(error, copy.system.genericRequestFailed));
      }
    } finally {
      setLoading(false);
      setActiveRun(null);
      setStopPending(false);
    }
  }, [
    copy.system.genericRequestFailed,
    appendCommittedMessages,
    appendErrorMessage,
    clearChatError,
    clearStreamingState,
    currentSessionId,
    runAgentStream,
    setActiveRun,
    setLoading,
    setStopPending,
    stopPendingRef,
  ]);

  const stopCurrentRun = useCallback(async () => {
    const run = activeRunRef.current;
    if (!run || stopPendingRef.current) {
      return;
    }

    clearChatError();
    setStopPending(true);
    try {
      const response = await stopAgent(run.sessionId || undefined, run.traceId || undefined);
      run.abortController?.abort();
      await syncStoppedRunHistory(resolveStopSessionId(response.session_id, run.sessionId));
    } catch (error) {
      setChatError(toErrorMessage(error, copy.system.genericRequestFailed));
      setStopPending(false);
    }
  }, [
    activeRunRef,
    clearChatError,
    copy.system.genericRequestFailed,
    setChatError,
    setStopPending,
    stopPendingRef,
    syncStoppedRunHistory,
  ]);

  return {
    sendChatMessage,
    stopCurrentRun,
  };
}

function hasSendPayload(input: ChatSendInput): boolean {
  return input.message.trim().length > 0 || input.images.length > 0;
}

function shouldSuppressRunError(error: unknown, stopPending: boolean): boolean {
  if (!stopPending) {
    return false;
  }

  const message = toErrorMessage(error);
  return isAbortError(error)
    || message === 'agent stream closed before terminal event'
    || message === 'agent run cancelled';
}

function resolveStopSessionId(stoppedSessionId?: string, activeSessionId?: string): string {
  const trimmedStoppedSessionId = stoppedSessionId?.trim();
  if (trimmedStoppedSessionId) {
    return trimmedStoppedSessionId;
  }
  return activeSessionId?.trim() ?? '';
}
