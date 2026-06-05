import { useCallback } from 'react';
import { getFullSession, getSession } from '@/lib/api/sessions/api';
import { mapSessionMessagesToChat } from '@/lib/chatMessages';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import type { ChatMessage, SessionDetail } from '@/lib/types';
import { mergeLatestCommittedMessages } from './chatHistoryMerge';
import { useChatHistoryRecovery } from './useChatHistoryRecovery';
import type { ChatStateControls } from './types';

const HISTORY_PAGE_LIMIT = 100;

interface UseChatHistoryOptions {
  clearChatError: ChatStateControls['clearChatError'];
  clearPendingQuestions: ChatStateControls['clearPendingQuestions'];
  clearStreamingState: ChatStateControls['clearStreamingState'];
  applyRuntimeActions: ChatStateControls['applyRuntimeActions'];
  hydrateTurnDraft: ChatStateControls['hydrateTurnDraft'];
  replaceWithErrorMessage: ChatStateControls['replaceWithErrorMessage'];
  setCommittedMessages: ChatStateControls['setCommittedMessages'];
  setHasOlderHistory: ChatStateControls['setHasOlderHistory'];
  setHistoryLoading: ChatStateControls['setHistoryLoading'];
  setLoadingOlderHistory: ChatStateControls['setLoadingOlderHistory'];
  setNextHistoryBefore: ChatStateControls['setNextHistoryBefore'];
  setChatError: ChatStateControls['setChatError'];
  setActiveRun: ChatStateControls['setActiveRun'];
  setLoading: ChatStateControls['setLoading'];
  setStopPending: ChatStateControls['setStopPending'];
  beginHistorySync: ChatStateControls['beginHistorySync'];
  endHistorySync: ChatStateControls['endHistorySync'];
  nextHistoryBefore: ChatStateControls['nextHistoryBefore'];
}

export function useChatHistory(options: UseChatHistoryOptions) {
  const { copy } = useWebLocale();
  const {
    clearChatError,
    clearPendingQuestions,
    clearStreamingState,
    applyRuntimeActions,
    hydrateTurnDraft,
    replaceWithErrorMessage,
    setCommittedMessages,
    setHasOlderHistory,
    setHistoryLoading,
    setLoadingOlderHistory,
    setNextHistoryBefore,
    setChatError,
    setActiveRun,
    setLoading,
    setStopPending,
    beginHistorySync,
    endHistorySync,
    nextHistoryBefore,
  } = options;
  const { recoverTurnDraft, stopRecoveredRun } = useChatHistoryRecovery({
    applyRuntimeActions,
    beginHistorySync,
    endHistorySync,
    hydrateTurnDraft,
    setActiveRun,
    setChatError,
    setCommittedMessages,
    setHasOlderHistory,
    setLoading,
    setNextHistoryBefore,
    setStopPending,
  });

  const hydrateSessionHistory = useCallback(async (sessionId: string) => {
    const detail = await getFullSession(sessionId, HISTORY_PAGE_LIMIT);
    applyHistoryPage(detail, setHasOlderHistory, setNextHistoryBefore);
    clearPendingQuestions();
    setCommittedMessages(mapSessionMessagesToChat(detail.id, detail.messages));
    if (detail.turn_draft) {
      recoverTurnDraft(detail.id, detail.turn_draft);
      return;
    }
    stopRecoveredRun();
    clearStreamingState();
    hydrateTurnDraft(detail.id, null);
    setActiveRun(null);
    setLoading(false);
    setStopPending(false);
  }, [
    clearPendingQuestions,
    clearStreamingState,
    hydrateTurnDraft,
    recoverTurnDraft,
    setActiveRun,
    setCommittedMessages,
    setHasOlderHistory,
    setLoading,
    setNextHistoryBefore,
    setStopPending,
    stopRecoveredRun,
  ]);

  const syncRecentHistory = useCallback(async (sessionId: string) => {
    const detail = await getFullSession(sessionId, HISTORY_PAGE_LIMIT);
    applyHistoryPage(detail, setHasOlderHistory, setNextHistoryBefore);
    const latest = mapSessionMessagesToChat(detail.id, detail.messages);
    setCommittedMessages((previous) => {
      return mergeLatestCommittedMessages(previous, latest);
    });
  }, [setCommittedMessages, setHasOlderHistory, setNextHistoryBefore]);

  const loadSessionHistory = useCallback(async (sessionId: string) => {
    const id = sessionId.trim();
    if (!id) {
      stopRecoveredRun();
      clearStreamingState();
      hydrateTurnDraft('', null);
      clearPendingQuestions();
      setCommittedMessages([]);
      setHasOlderHistory(false);
      setNextHistoryBefore(null);
      setActiveRun(null);
      setLoading(false);
      setStopPending(false);
      return;
    }

    clearChatError();
    stopRecoveredRun();
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
    hydrateTurnDraft,
    hydrateSessionHistory,
    replaceWithErrorMessage,
    setActiveRun,
    setChatError,
    setCommittedMessages,
    setHasOlderHistory,
    setHistoryLoading,
    setLoading,
    setNextHistoryBefore,
    setStopPending,
    stopRecoveredRun,
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
      setCommittedMessages((previous) => {
        return prependUniqueCommittedMessages(previous, older);
      });
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
