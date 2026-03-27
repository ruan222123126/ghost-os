import { hasPendingQuestion } from '@/lib/chatMessages';
import { useChatHistory } from './useChatHistory';
import { useChatQuestionActions } from './useChatQuestionActions';
import { useChatRunControl } from './useChatRunControl';
import { useChatState } from './useChatState';
import { useChatStreamController } from './useChatStreamController';
import type { UseBridgeChatOptions, UseBridgeChatResult } from './types';

export function useBridgeChat(options: UseBridgeChatOptions): UseBridgeChatResult {
  const state = useChatState();
  const { hydrateSessionHistory, loadSessionHistory } = useChatHistory({
    clearChatError: state.clearChatError,
    replaceWithErrorMessage: state.replaceWithErrorMessage,
    setHistoryLoading: state.setHistoryLoading,
    setMessages: state.setMessages,
    setChatError: state.setChatError,
  });
  const { runAgentStream, runHumanStream } = useChatStreamController({
    currentSessionId: options.currentSessionId,
    hydrateSessionHistory,
    onSessionResolved: options.onSessionResolved,
    activeRunRef: state.activeRunRef,
    setActiveRun: state.setActiveRun,
    setMessages: state.setMessages,
  });
  const { sendChatMessage, stopCurrentRun } = useChatRunControl({
    appendErrorMessage: state.appendErrorMessage,
    appendMessages: state.appendMessages,
    clearChatError: state.clearChatError,
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
    appendErrorMessage: state.appendErrorMessage,
    clearChatError: state.clearChatError,
    messages: state.messages,
    runHumanStream,
    setActiveRun: state.setActiveRun,
    setChatError: state.setChatError,
    setLoading: state.setLoading,
    setStopPending: state.setStopPending,
    setMessages: state.setMessages,
    stopPendingRef: state.stopPendingRef,
  });

  return {
    messages: state.messages,
    loading: state.loading,
    historyLoading: state.historyLoading,
    chatError: state.chatError,
    hasPendingQuestion: hasPendingQuestion(state.messages),
    canStop: state.loading && state.activeRun !== null && !state.stopPending,
    sendChatMessage,
    stopCurrentRun,
    answerQuestion,
    cancelQuestion,
    loadSessionHistory,
    clearMessages: state.clearMessages,
  };
}
