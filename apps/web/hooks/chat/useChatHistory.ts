import { useCallback } from 'react';
import { getSession } from '@/lib/api/sessions/api';
import { mapSessionMessagesToChat } from '@/lib/chatMessages';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ChatMessage, SessionDetail } from '@/lib/types';
import type { ChatStateControls } from './types';

const HISTORY_PAGE_LIMIT = 100;

interface UseChatHistoryOptions {
  clearChatError: ChatStateControls['clearChatError'];
  clearPendingQuestions: ChatStateControls['clearPendingQuestions'];
  clearStreamingState: ChatStateControls['clearStreamingState'];
  replaceWithErrorMessage: ChatStateControls['replaceWithErrorMessage'];
  setCommittedMessages: ChatStateControls['setCommittedMessages'];
  setHasOlderHistory: ChatStateControls['setHasOlderHistory'];
  setHistoryLoading: ChatStateControls['setHistoryLoading'];
  setLoadingOlderHistory: ChatStateControls['setLoadingOlderHistory'];
  setNextHistoryBefore: ChatStateControls['setNextHistoryBefore'];
  setChatError: ChatStateControls['setChatError'];
  nextHistoryBefore: ChatStateControls['nextHistoryBefore'];
}

export function useChatHistory(options: UseChatHistoryOptions) {
  const { copy } = useWebLocale();
  const {
    clearChatError,
    clearPendingQuestions,
    clearStreamingState,
    replaceWithErrorMessage,
    setCommittedMessages,
    setHasOlderHistory,
    setHistoryLoading,
    setLoadingOlderHistory,
    setNextHistoryBefore,
    setChatError,
    nextHistoryBefore,
  } = options;

  const hydrateSessionHistory = useCallback(async (sessionId: string) => {
    const detail = await getSession(sessionId, { limit: HISTORY_PAGE_LIMIT });
    applyHistoryPage(detail, setHasOlderHistory, setNextHistoryBefore);
    clearStreamingState();
    clearPendingQuestions();
    setCommittedMessages(mapSessionMessagesToChat(detail.id, detail.messages));
  }, [
    clearPendingQuestions,
    clearStreamingState,
    setCommittedMessages,
    setHasOlderHistory,
    setNextHistoryBefore,
  ]);

  const syncRecentHistory = useCallback(async (sessionId: string) => {
    const detail = await getSession(sessionId, { limit: HISTORY_PAGE_LIMIT });
    applyHistoryPage(detail, setHasOlderHistory, setNextHistoryBefore);
    const latest = mapSessionMessagesToChat(detail.id, detail.messages);
    setCommittedMessages((previous) => mergeLatestCommittedMessages(previous, latest));
  }, [setCommittedMessages, setHasOlderHistory, setNextHistoryBefore]);

  const loadSessionHistory = useCallback(async (sessionId: string) => {
    const id = sessionId.trim();
    if (!id) {
      clearStreamingState();
      clearPendingQuestions();
      setCommittedMessages([]);
      setHasOlderHistory(false);
      setNextHistoryBefore(null);
      return;
    }

    clearChatError();
    setHistoryLoading(true);
    try {
      await hydrateSessionHistory(id);
    } catch (error) {
      const messageText = toErrorMessage(error, copy.system.genericRequestFailed);
      setChatError(messageText);
      replaceWithErrorMessage(messageText);
    } finally {
      setHistoryLoading(false);
    }
  }, [
    copy.system.genericRequestFailed,
    clearChatError,
    clearPendingQuestions,
    clearStreamingState,
    hydrateSessionHistory,
    replaceWithErrorMessage,
    setChatError,
    setCommittedMessages,
    setHasOlderHistory,
    setHistoryLoading,
    setNextHistoryBefore,
  ]);

  const loadOlderHistory = useCallback(async (sessionId: string) => {
    const id = sessionId.trim();
    if (!id || nextHistoryBefore === null) {
      return;
    }

    clearChatError();
    setLoadingOlderHistory(true);
    try {
      const detail = await getSession(id, {
        before: nextHistoryBefore,
        limit: HISTORY_PAGE_LIMIT,
      });
      applyHistoryPage(detail, setHasOlderHistory, setNextHistoryBefore);
      const older = mapSessionMessagesToChat(detail.id, detail.messages);
      setCommittedMessages((previous) => prependUniqueCommittedMessages(previous, older));
    } catch (error) {
      setChatError(toErrorMessage(error, copy.system.genericRequestFailed));
    } finally {
      setLoadingOlderHistory(false);
    }
  }, [
    copy.system.genericRequestFailed,
    clearChatError,
    nextHistoryBefore,
    setChatError,
    setCommittedMessages,
    setHasOlderHistory,
    setLoadingOlderHistory,
    setNextHistoryBefore,
  ]);

  return {
    hydrateSessionHistory,
    syncRecentHistory,
    loadSessionHistory,
    loadOlderHistory,
  };
}

function applyHistoryPage(
  detail: SessionDetail,
  setHasOlderHistory: ChatStateControls['setHasOlderHistory'],
  setNextHistoryBefore: ChatStateControls['setNextHistoryBefore'],
): void {
  setHasOlderHistory(detail.page.has_more_before);
  setNextHistoryBefore(detail.page.next_before ?? null);
}

function prependUniqueCommittedMessages(previous: ChatMessage[], older: ChatMessage[]): ChatMessage[] {
  if (older.length === 0) {
    return previous;
  }

  const olderIDs = new Set(older.map((message) => message.id));
  return [...older, ...previous.filter((message) => !olderIDs.has(message.id))];
}

function mergeLatestCommittedMessages(previous: ChatMessage[], latest: ChatMessage[]): ChatMessage[] {
  const latestIDs = new Set(latest.map((message) => message.id));
  const preserved = previous.filter((message) => {
    if (message.kind === 'thinking') {
      return true;
    }
    if (message.id.startsWith('stream-') || message.id.startsWith('local:')) {
      return false;
    }
    return !latestIDs.has(message.id);
  });

  return [...preserved, ...latest];
}
