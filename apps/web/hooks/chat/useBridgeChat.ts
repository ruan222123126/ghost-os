import { useCallback, useEffect, useRef } from 'react';
import { useChatHistory } from './useChatHistory';
import { useChatQuestionActions } from './useChatQuestionActions';
import { useChatRunControl } from './useChatRunControl';
import { useChatState } from './useChatState';
import { useChatStreamController } from './useChatStreamController';
import type { UseBridgeChatOptions, UseBridgeChatResult } from './types';

export function useBridgeChat(options: UseBridgeChatOptions): UseBridgeChatResult {
  const state = useChatState(options.currentSessionId);
  const currentSessionIdRef = useRef(options.currentSessionId);

  useEffect(() => {
    currentSessionIdRef.current = options.currentSessionId;
  }, [options.currentSessionId]);

  const {
    loadOlderHistory,
    loadSessionHistory,
    syncRecentHistory,
  } = useChatHistory({
    clearChatError: state.clearChatError,
    clearPendingQuestions: state.clearPendingQuestions,
    clearStreamingState: state.clearStreamingState,
    applyRuntimeActions: state.applyRuntimeActions,
    hydrateTurnDraft: state.hydrateTurnDraft,
    replaceWithErrorMessage: state.replaceWithErrorMessage,
    setCommittedMessages: state.setCommittedMessages,
    setHasOlderHistory: state.setHasOlderHistory,
    setHistoryLoading: state.setHistoryLoading,
    setLoadingOlderHistory: state.setLoadingOlderHistory,
    setNextHistoryBefore: state.setNextHistoryBefore,
    setChatError: state.setChatError,
    setActiveRun: state.setActiveRun,
    setLoading: state.setLoading,
    setStopPending: state.setStopPending,
    beginHistorySync: state.beginHistorySync,
    endHistorySync: state.endHistorySync,
    getNextHistoryBefore: state.getNextHistoryBefore,
  });
  const { runAgentStream, runHumanStream } = useChatStreamController({
    applyRuntimeActions: state.applyRuntimeActions,
    endHistorySync: state.endHistorySync,
    getCurrentSessionId: () => currentSessionIdRef.current,
    migrateSessionState: state.migrateSessionState,
    onSessionResolved: options.onSessionResolved,
    setChatError: state.setChatError,
    beginHistorySync: state.beginHistorySync,
    syncRecentHistory,
  });
  const { sendChatMessage, stopCurrentRun } = useChatRunControl({
    appendErrorMessage: state.appendErrorMessage,
    appendCommittedMessages: state.appendCommittedMessages,
    beginHistorySync: state.beginHistorySync,
    clearChatError: state.clearChatError,
    clearStreamingState: state.clearStreamingState,
    currentSessionId: options.currentSessionId,
    endHistorySync: state.endHistorySync,
    getCurrentSessionId: () => currentSessionIdRef.current,
    getActiveRun: state.getActiveRun,
    getStopPending: state.getStopPending,
    hasPendingQuestionInSession: state.hasPendingQuestionInSession,
    markBackgroundCompleted: state.markBackgroundCompleted,
    migrateSessionState: state.migrateSessionState,
    onSessionResolved: options.onSessionResolved,
    runAgentStream,
    resolveActiveRunSessionId: state.resolveActiveRunSessionId,
    setActiveRun: state.setActiveRun,
    setLoading: state.setLoading,
    setStopPending: state.setStopPending,
    setChatError: state.setChatError,
    syncRecentHistory,
  });
  const { answerQuestion, cancelQuestion } = useChatQuestionActions({
    appendCommittedMessages: state.appendCommittedMessages,
    appendErrorMessage: state.appendErrorMessage,
    clearChatError: state.clearChatError,
    clearStreamingState: state.clearStreamingState,
    pendingQuestions: state.pendingQuestions,
    runHumanStream,
    getCurrentSessionId: () => currentSessionIdRef.current,
    getStopPending: state.getStopPending,
    hasPendingQuestionInSession: state.hasPendingQuestionInSession,
    markBackgroundCompleted: state.markBackgroundCompleted,
    removePendingQuestion: state.removePendingQuestion,
    resolveActiveRunSessionId: state.resolveActiveRunSessionId,
    setActiveRun: state.setActiveRun,
    setChatError: state.setChatError,
    setLoading: state.setLoading,
    setStopPending: state.setStopPending,
  });
  const loadOlderCurrentSessionHistory = useCallback(async () => {
    await loadOlderHistory(options.currentSessionId);
  }, [loadOlderHistory, options.currentSessionId]);

  return {
    committedMessages: state.committedMessages,
    streamingAssistantSegments: state.streamingAssistantSegments,
    streamingThinkingSegments: state.streamingThinkingSegments,
    activeStreamingThinkingId: state.activeStreamingThinkingId,
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
    clearMessages: (sessionId = options.currentSessionId) => state.clearMessages(sessionId),
    backgroundCompletedSessionIds: state.backgroundCompletedSessionIds,
    clearBackgroundCompletion: state.clearBackgroundCompletion,
    dropSessionState: state.dropSessionState,
    shouldLoadSessionHistory: state.shouldLoadSessionHistory,
  };
}
