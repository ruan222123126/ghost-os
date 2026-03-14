import { useCallback } from 'react';
import { getSession } from '@/lib/api/sessions/api';
import { mapSessionMessagesToChat } from '@/lib/chatMessages';
import { toErrorMessage } from '@/lib/errors';
import type { ChatStateControls } from './types';

interface UseChatHistoryOptions {
  clearChatError: ChatStateControls['clearChatError'];
  replaceWithErrorMessage: ChatStateControls['replaceWithErrorMessage'];
  setHistoryLoading: ChatStateControls['setHistoryLoading'];
  setMessages: ChatStateControls['setMessages'];
  setChatError: ChatStateControls['setChatError'];
}

export function useChatHistory(options: UseChatHistoryOptions) {
  const { clearChatError, replaceWithErrorMessage, setHistoryLoading, setMessages, setChatError } = options;

  const hydrateSessionHistory = useCallback(async (sessionId: string) => {
    const detail = await getSession(sessionId);
    setMessages(mapSessionMessagesToChat(detail.messages));
  }, [setMessages]);

  const loadSessionHistory = useCallback(async (sessionId: string) => {
    const id = sessionId.trim();
    if (!id) {
      setMessages([]);
      return;
    }

    clearChatError();
    setHistoryLoading(true);
    try {
      await hydrateSessionHistory(id);
    } catch (error) {
      const messageText = toErrorMessage(error);
      setChatError(messageText);
      replaceWithErrorMessage(messageText);
    } finally {
      setHistoryLoading(false);
    }
  }, [clearChatError, hydrateSessionHistory, replaceWithErrorMessage, setChatError, setHistoryLoading, setMessages]);

  return {
    hydrateSessionHistory,
    loadSessionHistory,
  };
}
