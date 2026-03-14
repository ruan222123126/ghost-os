import { useCallback } from 'react';
import { sendMessage, stopAgent } from '@/lib/api/agent/api';
import { createClientTraceId } from '@/lib/api/trace';
import { buildUserMessage } from '@/lib/chatMessages';
import { toErrorMessage } from '@/lib/errors';
import type { ChatStateControls, UseBridgeChatOptions } from './types';

const AGENT_RUN_CANCELLED_MESSAGE = 'agent run cancelled';

interface UseChatRunControlOptions {
  appendErrorMessage: ChatStateControls['appendErrorMessage'];
  appendMessages: ChatStateControls['appendMessages'];
  clearChatError: ChatStateControls['clearChatError'];
  currentSessionId: UseBridgeChatOptions['currentSessionId'];
  handleReply: (reply: Awaited<ReturnType<typeof sendMessage>>) => Promise<void>;
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
    handleReply,
    activeRunRef,
    setActiveRun,
    setLoading,
    setStopPending,
    setChatError,
    stopPendingRef,
  } = options;

  const sendChatMessage = useCallback(async (message: string) => {
    const trimmed = message.trim();
    const sessionId = currentSessionId.trim();
    if (!trimmed && !sessionId) {
      return;
    }

    const traceId = createClientTraceId('agent-run');
    clearChatError();
    setStopPending(false);
    setActiveRun({ sessionId, traceId });
    if (trimmed) {
      appendMessages([buildUserMessage(trimmed)]);
    }
    setLoading(true);

    try {
      const reply = await sendMessage(trimmed, sessionId || undefined, traceId);
      await handleReply(reply);
    } catch (error) {
      const messageText = toErrorMessage(error);
      if (!(stopPendingRef.current && messageText === AGENT_RUN_CANCELLED_MESSAGE)) {
        appendErrorMessage(messageText);
      }
    } finally {
      setLoading(false);
      setActiveRun(null);
      setStopPending(false);
    }
  }, [appendErrorMessage, appendMessages, clearChatError, currentSessionId, handleReply, setActiveRun, setLoading, setStopPending, stopPendingRef]);

  const stopCurrentRun = useCallback(async () => {
    const run = activeRunRef.current;
    if (!run || stopPendingRef.current) {
      return;
    }

    clearChatError();
    setStopPending(true);
    try {
      await stopAgent(run.sessionId || undefined, run.traceId || undefined);
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
