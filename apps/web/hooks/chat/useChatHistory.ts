import { useCallback, useMemo } from 'react';
import { getSession } from '@/lib/api/sessions/api';
import { mapSessionMessagesToChat } from '@/lib/chatMessages';
import { toErrorMessage } from '@/lib/errors';
import { useWebLocale } from '@/lib/i18n/provider';
import {
  applyHistoryPageState,
  mergeLatestCommittedMessages,
  prependUniqueCommittedMessages,
} from './chatHistoryMerge';
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
  getNextHistoryBefore: ChatStateControls['getNextHistoryBefore'];
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
    getNextHistoryBefore,
  } = options;
  const historyPageStateTarget = useMemo(() => ({
    setHasOlderHistory,
    setNextHistoryBefore,
  }), [setHasOlderHistory, setNextHistoryBefore]);
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
    const detail = await getSession(sessionId, { limit: HISTORY_PAGE_LIMIT });
    applyHistoryPageState(detail, detail.id, historyPageStateTarget);
    clearPendingQuestions(detail.id);
    setCommittedMessages(detail.id, mapSessionMessagesToChat(detail.id, detail.messages));
    if (detail.turn_draft) {
      recoverTurnDraft(detail.id, detail.turn_draft);
      return;
    }
    stopRecoveredRun(detail.id);
    clearStreamingState(detail.id);
    hydrateTurnDraft(detail.id, null);
    setActiveRun(detail.id, null);
    setLoading(detail.id, false);
    setStopPending(detail.id, false);
  }, [
    clearPendingQuestions,
    clearStreamingState,
    hydrateTurnDraft,
    historyPageStateTarget,
    recoverTurnDraft,
    setActiveRun,
    setCommittedMessages,
    setLoading,
    setStopPending,
    stopRecoveredRun,
  ]);

  const syncRecentHistory = useCallback(async (sessionId: string) => {
    const detail = await getSession(sessionId, { limit: HISTORY_PAGE_LIMIT });
    applyHistoryPageState(detail, detail.id, historyPageStateTarget);
    const latest = mapSessionMessagesToChat(detail.id, detail.messages);
    setCommittedMessages(detail.id, (previous) => {
      return mergeLatestCommittedMessages(previous, latest);
    });
    hydrateTurnDraft(detail.id, detail.turn_draft ?? null);
  }, [historyPageStateTarget, hydrateTurnDraft, setCommittedMessages]);

  const loadSessionHistory = useCallback(async (sessionId: string) => {
    const id = sessionId.trim();
    if (!id) {
      stopRecoveredRun('');
      clearStreamingState('');
      hydrateTurnDraft('', null);
      clearPendingQuestions('');
      setCommittedMessages('', []);
      setHasOlderHistory('', false);
      setNextHistoryBefore('', null);
      setActiveRun('', null);
      setLoading('', false);
      setStopPending('', false);
      return;
    }

    clearChatError(id);
    stopRecoveredRun(id);
    setHistoryLoading(id, true);
    try {
      await hydrateSessionHistory(id);
    } catch (error) {
      const messageText = toErrorMessage(error, copy.system.genericRequestFailed);
      setChatError(id, messageText);
      replaceWithErrorMessage(id, messageText);
    } finally {
      setHistoryLoading(id, false);
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
    const nextHistoryBefore = getNextHistoryBefore(id);
    if (!id || nextHistoryBefore === null) {
      return;
    }

    clearChatError(id);
    setLoadingOlderHistory(id, true);
    try {
      const detail = await getSession(id, {
        before: nextHistoryBefore,
        limit: HISTORY_PAGE_LIMIT,
      });
      applyHistoryPageState(detail, detail.id, historyPageStateTarget);
      const older = mapSessionMessagesToChat(detail.id, detail.messages);
      setCommittedMessages(detail.id, (previous) => {
        return prependUniqueCommittedMessages(previous, older);
      });
    } catch (error) {
      setChatError(id, toErrorMessage(error, copy.system.genericRequestFailed));
    } finally {
      setLoadingOlderHistory(id, false);
    }
  }, [
    copy.system.genericRequestFailed,
    clearChatError,
    getNextHistoryBefore,
    historyPageStateTarget,
    setChatError,
    setCommittedMessages,
    setLoadingOlderHistory,
  ]);

  return {
    hydrateSessionHistory,
    syncRecentHistory,
    loadSessionHistory,
    loadOlderHistory,
  };
}
