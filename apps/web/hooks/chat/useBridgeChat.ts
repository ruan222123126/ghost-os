import { useCallback, useEffect, useRef } from 'react';
import { useChatHistory } from './useChatHistory';
import { useChatQuestionActions } from './useChatQuestionActions';
import { useChatRunControl } from './useChatRunControl';
import { useChatState } from './useChatState';
import { useChatStreamController } from './useChatStreamController';
import type { ChatStateControls, UseBridgeChatOptions, UseBridgeChatResult } from './types';

interface BridgeChatResultOptions {
  actions: {
    answerQuestion: UseBridgeChatResult['answerQuestion'];
    cancelQuestion: UseBridgeChatResult['cancelQuestion'];
    loadOlderHistory: UseBridgeChatResult['loadOlderHistory'];
    loadSessionHistory: UseBridgeChatResult['loadSessionHistory'];
    sendChatMessage: UseBridgeChatResult['sendChatMessage'];
    stopCurrentRun: UseBridgeChatResult['stopCurrentRun'];
  };
  clearMessages: UseBridgeChatResult['clearMessages'];
  state: ChatStateControls;
}

export function useBridgeChat(options: UseBridgeChatOptions): UseBridgeChatResult {
  const state = useChatState(options.currentSessionId);
  const currentSessionIdRef = useCurrentSessionIdRef(options.currentSessionId);
  const getCurrentSessionId = useCallback(() => currentSessionIdRef.current, [currentSessionIdRef]);
  const { loadOlderHistory, loadSessionHistory, syncRecentHistory } = useChatHistory(state);
  const { runAgentStream, runHumanStream } = useChatStreamController({
    ...state,
    getCurrentSessionId,
    onSessionResolved: options.onSessionResolved,
    syncRecentHistory,
  });
  const { sendChatMessage, stopCurrentRun } = useChatRunControl({
    ...state,
    currentSessionId: options.currentSessionId,
    externalCodexPermissionMode: options.externalCodexPermissionMode,
    externalProjectRoot: options.externalProjectRoot,
    getCurrentSessionId,
    onSessionResolved: options.onSessionResolved,
    runAgentStream,
    syncRecentHistory,
  });
  const { answerQuestion, cancelQuestion } = useChatQuestionActions({
    ...state,
    runHumanStream,
    getCurrentSessionId,
  });
  const loadOlderCurrentSessionHistory = useCallback(async () => {
    await loadOlderHistory(options.currentSessionId);
  }, [loadOlderHistory, options.currentSessionId]);
  const clearMessages = useCallback((sessionId = options.currentSessionId) => {
    state.clearMessages(sessionId);
  }, [options.currentSessionId, state]);

  return buildBridgeChatResult({
    actions: {
      answerQuestion,
      cancelQuestion,
      loadOlderHistory: loadOlderCurrentSessionHistory,
      loadSessionHistory,
      sendChatMessage,
      stopCurrentRun,
    },
    clearMessages,
    state,
  });
}

function useCurrentSessionIdRef(currentSessionId: string) {
  const currentSessionIdRef = useRef(currentSessionId);
  useEffect(() => {
    currentSessionIdRef.current = currentSessionId;
  }, [currentSessionId]);
  return currentSessionIdRef;
}

function buildBridgeChatResult(options: BridgeChatResultOptions): UseBridgeChatResult {
  const { actions, clearMessages, state } = options;
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
    canStop: canStopActiveRun(state.activeRun, state.loading, state.stopPending),
    hasOlderHistory: state.hasOlderHistory,
    sendChatMessage: actions.sendChatMessage,
    stopCurrentRun: actions.stopCurrentRun,
    answerQuestion: actions.answerQuestion,
    cancelQuestion: actions.cancelQuestion,
    loadSessionHistory: actions.loadSessionHistory,
    loadOlderHistory: actions.loadOlderHistory,
    clearMessages,
    backgroundCompletedSessionIds: state.backgroundCompletedSessionIds,
    clearBackgroundCompletion: state.clearBackgroundCompletion,
    dropSessionState: state.dropSessionState,
    shouldLoadSessionHistory: state.shouldLoadSessionHistory,
  };
}

function canStopActiveRun(
  activeRun: ChatStateControls['activeRun'],
  loading: boolean,
  stopPending: boolean,
): boolean {
  if (!loading || !activeRun || stopPending) {
    return false;
  }

  if (activeRun.runtime === 'codex') {
    return activeRun.sessionId.trim().length > 0;
  }

  return true;
}
