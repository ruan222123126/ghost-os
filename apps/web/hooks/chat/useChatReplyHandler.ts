import { useCallback } from 'react';
import { mapAgentReplyToChatMessages } from '@/lib/chatMessages';
import type { AgentSendResponse } from '@/lib/types';
import type { ChatStateControls, UseBridgeChatOptions } from './types';

interface UseChatReplyHandlerOptions {
  appendMessages: ChatStateControls['appendMessages'];
  currentSessionId: UseBridgeChatOptions['currentSessionId'];
  hydrateSessionHistory: (sessionId: string) => Promise<void>;
  onSessionResolved: UseBridgeChatOptions['onSessionResolved'];
  activeRunRef: ChatStateControls['activeRunRef'];
  setActiveRun: ChatStateControls['setActiveRun'];
}

export function useChatReplyHandler(options: UseChatReplyHandlerOptions) {
  const { appendMessages, currentSessionId, hydrateSessionHistory, onSessionResolved, activeRunRef, setActiveRun } = options;

  const handleReply = useCallback(async (reply: AgentSendResponse) => {
    if (!reply.session_id) {
      appendMessages(mapAgentReplyToChatMessages(reply));
      return;
    }

    const currentRun = activeRunRef.current;
    if (currentRun && currentRun.sessionId !== reply.session_id) {
      setActiveRun({ ...currentRun, sessionId: reply.session_id });
    }
    if (reply.session_id !== currentSessionId) {
      onSessionResolved?.(reply.session_id);
    }
    await hydrateSessionHistory(reply.session_id);
  }, [activeRunRef, appendMessages, currentSessionId, hydrateSessionHistory, onSessionResolved, setActiveRun]);

  return {
    handleReply,
  };
}
