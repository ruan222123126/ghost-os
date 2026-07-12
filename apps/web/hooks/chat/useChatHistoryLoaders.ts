import { useCallback } from 'react';
import { getSession } from '@/lib/api/sessions/api';
import { mapSessionMessagesToChat } from '@/lib/chatMessages';
import { toErrorMessage } from '@/lib/errors';
import {
  applyHistoryPageState,
  mergeLatestCommittedMessages,
  prependUniqueCommittedMessages,
} from './chatHistoryMerge';
import type { ChatStateControls } from './types';
import type { useChatHistoryRecovery } from './useChatHistoryRecovery';
import type { SessionDetail } from '@/lib/types';

const HISTORY_PAGE_LIMIT = 100;

type HistoryPageStateTarget = Parameters<typeof applyHistoryPageState>[2];
type ChatHistoryRecoveryControls = ReturnType<typeof useChatHistoryRecovery>;

interface UseChatHistoryLoadersOptions extends Pick<ChatStateControls,
  | 'clearChatError'
  | 'clearPendingQuestions'
  | 'clearStreamingState'
  | 'hydrateTurnDraft'
  | 'replaceWithErrorMessage'
  | 'setActiveRun'
  | 'setChatError'
  | 'setCommittedMessages'
  | 'setHasOlderHistory'
  | 'setHistoryLoading'
  | 'setLoading'
  | 'setLoadingOlderHistory'
  | 'setNextHistoryBefore'
  | 'setStopPending'
  | 'getNextHistoryBefore'
> {
  historyPageStateTarget: HistoryPageStateTarget;
  recovery: ChatHistoryRecoveryControls;
  requestFailedText: string;
}

interface UseLoadSessionHistoryOptions extends UseChatHistoryLoadersOptions {
  hydrateSessionHistory: (sessionId: string) => Promise<SessionDetail>;
  resetEmptySessionHistory: () => void;
}

export function useChatHistoryLoaders(options: UseChatHistoryLoadersOptions) {
  const hydrateSessionHistory = useHydrateSessionHistory(options);
  const syncRecentHistory = useSyncRecentHistory(options);
  const resetEmptySessionHistory = useResetEmptySessionHistory(options);
  const loadSessionHistory = useLoadSessionHistory({
    ...options,
    hydrateSessionHistory,
    resetEmptySessionHistory,
  });
  const loadOlderHistory = useLoadOlderHistory(options);

  return {
    hydrateSessionHistory,
    syncRecentHistory,
    loadSessionHistory,
    loadOlderHistory,
  };
}

function useHydrateSessionHistory(options: UseChatHistoryLoadersOptions) {
  const {
    clearPendingQuestions,
    clearStreamingState,
    historyPageStateTarget,
    hydrateTurnDraft,
    recovery,
    setActiveRun,
    setCommittedMessages,
    setLoading,
    setStopPending,
  } = options;

  return useCallback(async (sessionId: string) => {
    const detail = await getSession(sessionId, { limit: HISTORY_PAGE_LIMIT });
    applyHistoryPageState(detail, detail.id, historyPageStateTarget);
    clearPendingQuestions(detail.id);
    setCommittedMessages(detail.id, mapSessionMessagesToChat(detail.id, detail.messages));
    if (detail.turn_draft) {
      recovery.recoverTurnDraft(detail.id, detail.turn_draft);
      return detail;
    }
    recovery.stopRecoveredRun(detail.id);
    clearStreamingState(detail.id);
    hydrateTurnDraft(detail.id, null);
    setActiveRun(detail.id, null);
    setLoading(detail.id, false);
    setStopPending(detail.id, false);
    return detail;
  }, [
    clearPendingQuestions,
    clearStreamingState,
    historyPageStateTarget,
    hydrateTurnDraft,
    recovery,
    setActiveRun,
    setCommittedMessages,
    setLoading,
    setStopPending,
  ]);
}

function useSyncRecentHistory(options: UseChatHistoryLoadersOptions) {
  const { historyPageStateTarget, hydrateTurnDraft, setCommittedMessages } = options;

  return useCallback(async (sessionId: string) => {
    const detail = await getSession(sessionId, { limit: HISTORY_PAGE_LIMIT });
    applyHistoryPageState(detail, detail.id, historyPageStateTarget);
    const latest = mapSessionMessagesToChat(detail.id, detail.messages);
    setCommittedMessages(detail.id, (previous) => {
      return mergeLatestCommittedMessages(previous, latest);
    });
    hydrateTurnDraft(detail.id, detail.turn_draft ?? null);
  }, [historyPageStateTarget, hydrateTurnDraft, setCommittedMessages]);
}

function useResetEmptySessionHistory(options: UseChatHistoryLoadersOptions) {
  const {
    clearPendingQuestions,
    clearStreamingState,
    hydrateTurnDraft,
    recovery,
    setActiveRun,
    setCommittedMessages,
    setHasOlderHistory,
    setLoading,
    setNextHistoryBefore,
    setStopPending,
  } = options;

  return useCallback(() => {
    recovery.stopRecoveredRun('');
    clearStreamingState('');
    hydrateTurnDraft('', null);
    clearPendingQuestions('');
    setCommittedMessages('', []);
    setHasOlderHistory('', false);
    setNextHistoryBefore('', null);
    setActiveRun('', null);
    setLoading('', false);
    setStopPending('', false);
  }, [
    clearPendingQuestions,
    clearStreamingState,
    hydrateTurnDraft,
    recovery,
    setActiveRun,
    setCommittedMessages,
    setHasOlderHistory,
    setLoading,
    setNextHistoryBefore,
    setStopPending,
  ]);
}

function useLoadSessionHistory(options: UseLoadSessionHistoryOptions) {
  const {
    clearChatError,
    hydrateSessionHistory,
    recovery,
    replaceWithErrorMessage,
    requestFailedText,
    resetEmptySessionHistory,
    setChatError,
    setHistoryLoading,
  } = options;

  return useCallback(async (sessionId: string): Promise<SessionDetail | null> => {
    const id = sessionId.trim();
    if (!id) {
      resetEmptySessionHistory();
      return null;
    }

    clearChatError(id);
    recovery.stopRecoveredRun(id);
    setHistoryLoading(id, true);
    try {
      return await hydrateSessionHistory(id);
    } catch (error) {
      const messageText = toErrorMessage(error, requestFailedText);
      setChatError(id, messageText);
      replaceWithErrorMessage(id, messageText);
      return null;
    } finally {
      setHistoryLoading(id, false);
    }
  }, [
    clearChatError,
    hydrateSessionHistory,
    recovery,
    replaceWithErrorMessage,
    requestFailedText,
    resetEmptySessionHistory,
    setChatError,
    setHistoryLoading,
  ]);
}

function useLoadOlderHistory(options: UseChatHistoryLoadersOptions) {
  const {
    clearChatError,
    getNextHistoryBefore,
    historyPageStateTarget,
    requestFailedText,
    setChatError,
    setCommittedMessages,
    setLoadingOlderHistory,
  } = options;

  return useCallback(async (sessionId: string) => {
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
      setChatError(id, toErrorMessage(error, requestFailedText));
    } finally {
      setLoadingOlderHistory(id, false);
    }
  }, [
    clearChatError,
    getNextHistoryBefore,
    historyPageStateTarget,
    requestFailedText,
    setChatError,
    setCommittedMessages,
    setLoadingOlderHistory,
  ]);
}
