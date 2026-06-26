import { useMemo } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';
import { useChatHistoryLoaders } from './useChatHistoryLoaders';
import { useChatHistoryRecovery } from './useChatHistoryRecovery';
import type { ChatStateControls } from './types';

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
  } = options;
  const historyPageStateTarget = useMemo(() => ({
    setHasOlderHistory,
    setNextHistoryBefore,
  }), [setHasOlderHistory, setNextHistoryBefore]);
  const recovery = useChatHistoryRecovery({
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

  return useChatHistoryLoaders({
    ...options,
    historyPageStateTarget,
    recovery,
    requestFailedText: copy.system.genericRequestFailed,
  });
}
