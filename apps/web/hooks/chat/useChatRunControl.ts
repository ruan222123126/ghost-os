import { useCallback } from 'react';
import { stopAgent } from '@/lib/api/agent/api';
import { createClientTraceId } from '@/lib/api/trace';
import { draftImagesToChatImages, draftImagesToSessionImages } from '@/lib/chatImageDrafts';
import { buildUserMessage } from '@/lib/chatMessages';
import { isAbortError, toErrorMessage } from '@/lib/errors';
import type { ChatSendInput } from '@/lib/types';
import type { ChatStateControls, StreamAgentRunInput, UseBridgeChatOptions } from './types';

interface UseChatRunControlOptions {
  appendErrorMessage: ChatStateControls['appendErrorMessage'];
  appendMessages: ChatStateControls['appendMessages'];
  clearChatError: ChatStateControls['clearChatError'];
  currentSessionId: UseBridgeChatOptions['currentSessionId'];
  runAgentStream: (run: StreamAgentRunInput) => Promise<void>;
  activeRunRef: ChatStateControls['activeRunRef'];
  setActiveRun: ChatStateControls['setActiveRun'];
  setLoading: ChatStateControls['setLoading'];
  setStopPending: ChatStateControls['setStopPending'];
  setChatError: ChatStateControls['setChatError'];
  stopPendingRef: ChatStateControls['stopPendingRef'];
}

export function useChatRunControl(options: UseChatRunControlOptions) {
  const {
    appendErrorMessage,
    appendMessages,
    clearChatError,
    currentSessionId,
    runAgentStream,
    activeRunRef,
    setActiveRun,
    setLoading,
    setStopPending,
    setChatError,
    stopPendingRef,
  } = options;

  const sendChatMessage = useCallback(async (input: ChatSendInput) => {
    if (!hasSendPayload(input)) {
      return;
    }

    const sessionId = currentSessionId.trim();
    const traceId = createClientTraceId('agent-run');
    const abortController = new AbortController();
    clearChatError();
    setStopPending(false);
    setActiveRun({ abortController, sessionId, traceId });
    appendMessages([buildUserMessage(input.message, { images: draftImagesToChatImages(input.images) })]);
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
        appendErrorMessage(toErrorMessage(error));
      }
    } finally {
      setLoading(false);
      setActiveRun(null);
      setStopPending(false);
    }
  }, [appendErrorMessage, appendMessages, clearChatError, currentSessionId, runAgentStream, setActiveRun, setLoading, setStopPending, stopPendingRef]);

  const stopCurrentRun = useCallback(async () => {
    const run = activeRunRef.current;
    if (!run || stopPendingRef.current) {
      return;
    }

    clearChatError();
    setStopPending(true);
    try {
      await stopAgent(run.sessionId || undefined, run.traceId || undefined);
      run.abortController?.abort();
    } catch (error) {
      setChatError(toErrorMessage(error));
      setStopPending(false);
    }
  }, [activeRunRef, clearChatError, setChatError, setStopPending, stopPendingRef]);

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

  return isAbortError(error) || toErrorMessage(error) === 'agent stream closed before terminal event';
}
