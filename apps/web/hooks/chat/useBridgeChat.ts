import { useCallback, useEffect } from 'react';
import { persistSessionThinkingSnapshot } from '@/lib/chatThinkingPersistence';
import { useChatHistory } from './useChatHistory';
import { useChatQuestionActions } from './useChatQuestionActions';
import { useChatRunControl } from './useChatRunControl';
import { useChatState } from './useChatState';
import { useChatStreamController } from './useChatStreamController';
import type { UseBridgeChatOptions, UseBridgeChatResult } from './types';

export function useBridgeChat(options: UseBridgeChatOptions): UseBridgeChatResult {
  const state = useChatState();
  useEffect(() => {
    const sessionId = options.currentSessionId.trim();
    if (!sessionId) {
      return;
    }
    persistSessionThinkingSnapshot(sessionId, state.committedMessages);
  }, [options.currentSessionId, state.committedMessages]);

  const {
    loadOlderHistory,
    loadSessionHistory,
    syncRecentHistory,
  } = useChatHistory({
    clearChatError: state.clearChatError,
    clearPendingQuestions: state.clearPendingQuestions,
    clearStreamingState: state.clearStreamingState,
    replaceWithErrorMessage: state.replaceWithErrorMessage,
    setCommittedMessages: state.setCommittedMessages,
    setHasOlderHistory: state.setHasOlderHistory,
    setHistoryLoading: state.setHistoryLoading,
    setLoadingOlderHistory: state.setLoadingOlderHistory,
    setNextHistoryBefore: state.setNextHistoryBefore,
    setChatError: state.setChatError,
    nextHistoryBefore: state.nextHistoryBefore,
  });
  const { runAgentStream, runHumanStream } = useChatStreamController({
    currentSessionId: options.currentSessionId,
    activeRunRef: state.activeRunRef,
    applyRuntimeActions: state.applyRuntimeActions,
    clearStreamingState: state.clearStreamingState,
    endHistorySync: state.endHistorySync,
    onSessionResolved: options.onSessionResolved,
    setChatError: state.setChatError,
    setActiveRun: state.setActiveRun,
    beginHistorySync: state.beginHistorySync,
    syncRecentHistory,
  });
  const { sendChatMessage, stopCurrentRun } = useChatRunControl({
    appendErrorMessage: state.appendErrorMessage,
    appendCommittedMessages: state.appendCommittedMessages,
    clearChatError: state.clearChatError,
    clearStreamingState: state.clearStreamingState,
    currentSessionId: options.currentSessionId,
    runAgentStream,
    activeRunRef: state.activeRunRef,
    setActiveRun: state.setActiveRun,
    setLoading: state.setLoading,
    setStopPending: state.setStopPending,
    setChatError: state.setChatError,
    stopPendingRef: state.stopPendingRef,
  });
  const { answerQuestion, cancelQuestion } = useChatQuestionActions({
    appendCommittedMessages: state.appendCommittedMessages,
    appendErrorMessage: state.appendErrorMessage,
    clearChatError: state.clearChatError,
    clearStreamingState: state.clearStreamingState,
    pendingQuestions: state.pendingQuestions,
    runHumanStream,
    removePendingQuestion: state.removePendingQuestion,
    setActiveRun: state.setActiveRun,
    setChatError: state.setChatError,
    setLoading: state.setLoading,
    setStopPending: state.setStopPending,
    stopPendingRef: state.stopPendingRef,
  });
  const loadOlderCurrentSessionHistory = useCallback(async () => {
    await loadOlderHistory(options.currentSessionId);
  }, [loadOlderHistory, options.currentSessionId]);

  return {
    committedMessages: state.committedMessages,
    streamingAssistantSegments: state.streamingAssistantSegments,
    streamingThinkingText: state.streamingThinkingText,
    streamingItemOrder: state.streamingItemOrder,
    streamingTools: state.streamingTools,
    pendingQuestions: state.pendingQuestions,
    loading: state.loading,
    historySyncing: state.historySyncing,
    historyLoading: state.historyLoading,
    loadingOlderHistory: state.loadingOlderHistory,
    chatError: state.chatError,
    hasPendingQuestion: state.pendingQuestions.length > 0,
    canStop: state.loading && state.activeRun !== null && !state.stopPending,
    hasOlderHistory: state.hasOlderHistory,
    sendChatMessage,
    stopCurrentRun,
    answerQuestion,
    cancelQuestion,
    loadSessionHistory,
    loadOlderHistory: loadOlderCurrentSessionHistory,
    clearMessages: state.clearMessages,
  };
}
