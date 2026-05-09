import { useCallback, useReducer, useRef, type SetStateAction } from 'react';
import {
  chatStateReducer,
  createInitialChatState,
} from '@/lib/chat-store/reducer';
import { buildChatStateView } from '@/lib/chat-store/runtimeReducer';
import type { ChatRuntimeAction } from '@/lib/chatRuntime/actions';
import type {
  ChatMessage,
  PendingQuestionMessage,
  SessionTurnDraft,
  StreamingToolState,
} from '@/lib/types';
import type { ActiveAgentRun, ChatStateControls } from './types';

export function useChatState(): ChatStateControls {
  const [state, dispatch] = useReducer(chatStateReducer, undefined, createInitialChatState);
  const view = buildChatStateView(state);
  const activeRunRef = useRef<ActiveAgentRun | null>(null);
  const stopPendingRef = useRef(false);
  const historySyncCountRef = useRef(0);

  const applyRuntimeActions = useCallback((actions: ChatRuntimeAction[]) => {
    if (actions.length === 0) {
      return;
    }
    dispatch({ type: 'apply_runtime_actions', actions });
  }, []);

  const setCommittedMessages = useCallback((updater: SetStateAction<ChatMessage[]>) => {
    dispatch({ type: 'set_committed_messages', updater });
  }, []);

  const setLoading = useCallback((value: boolean) => {
    dispatch({ type: 'set_scalar', key: 'loading', value });
  }, []);

  const setActiveRun = useCallback((value: ActiveAgentRun | null) => {
    activeRunRef.current = value;
    dispatch({ type: 'set_scalar', key: 'activeRun', value });
  }, []);

  const setStopPending = useCallback((value: boolean) => {
    stopPendingRef.current = value;
    dispatch({ type: 'set_scalar', key: 'stopPending', value });
  }, []);

  const beginHistorySync = useCallback(() => {
    historySyncCountRef.current += 1;
    dispatch({ type: 'set_scalar', key: 'historySyncing', value: true });
  }, []);

  const endHistorySync = useCallback(() => {
    if (historySyncCountRef.current === 0) {
      dispatch({ type: 'set_scalar', key: 'historySyncing', value: false });
      return;
    }
    historySyncCountRef.current -= 1;
    dispatch({
      type: 'set_scalar',
      key: 'historySyncing',
      value: historySyncCountRef.current > 0,
    });
  }, []);

  const appendCommittedMessages = useCallback((nextMessages: ChatMessage[]) => {
    applyRuntimeActions([{ type: 'append_committed_messages', messages: nextMessages }]);
  }, [applyRuntimeActions]);

  const clearChatError = useCallback(() => {
    dispatch({ type: 'set_scalar', key: 'chatError', value: '' });
  }, []);

  const setHistoryLoading = useCallback((value: boolean) => {
    dispatch({ type: 'set_scalar', key: 'historyLoading', value });
  }, []);

  const setLoadingOlderHistory = useCallback((value: boolean) => {
    dispatch({ type: 'set_scalar', key: 'loadingOlderHistory', value });
  }, []);

  const setChatError = useCallback((value: string) => {
    dispatch({ type: 'set_scalar', key: 'chatError', value });
  }, []);

  const setHasOlderHistory = useCallback((value: boolean) => {
    dispatch({ type: 'set_scalar', key: 'hasOlderHistory', value });
  }, []);

  const setNextHistoryBefore = useCallback((value: number | null) => {
    dispatch({ type: 'set_scalar', key: 'nextHistoryBefore', value });
  }, []);

  const clearStreamingTools = useCallback(() => {
    applyRuntimeActions([{ type: 'clear_streaming_tools' }]);
  }, [applyRuntimeActions]);

  const clearPendingQuestions = useCallback(() => {
    applyRuntimeActions([{ type: 'clear_pending_questions' }]);
  }, [applyRuntimeActions]);

  const clearStreamingAssistantText = useCallback(() => {
    applyRuntimeActions([{ type: 'clear_streaming_assistant_text' }]);
  }, [applyRuntimeActions]);

  const clearStreamingThinkingText = useCallback(() => {
    applyRuntimeActions([{ type: 'clear_streaming_thinking_text' }]);
  }, [applyRuntimeActions]);

  const clearStreamingState = useCallback(() => {
    applyRuntimeActions([
      { type: 'clear_streaming_assistant_text' },
      { type: 'clear_streaming_thinking_text' },
      { type: 'clear_streaming_tools' },
    ]);
  }, [applyRuntimeActions]);

  const replaceWithErrorMessage = useCallback((messageText: string) => {
    dispatch({ type: 'replace_with_error_message', messageText });
  }, []);

  const appendErrorMessage = useCallback((messageText: string) => {
    dispatch({ type: 'append_error_message', messageText });
  }, []);

  const appendStreamingAssistantText = useCallback((text: string) => {
    applyRuntimeActions([{ type: 'append_streaming_assistant_text', text }]);
  }, [applyRuntimeActions]);

  const upsertStreamingTool = useCallback((tool: StreamingToolState) => {
    applyRuntimeActions([{ type: 'upsert_streaming_tool', tool }]);
  }, [applyRuntimeActions]);

  const upsertPendingQuestion = useCallback((question: PendingQuestionMessage) => {
    applyRuntimeActions([{ type: 'upsert_pending_question', question }]);
  }, [applyRuntimeActions]);

  const removePendingQuestion = useCallback((questionId: string) => {
    applyRuntimeActions([{ type: 'remove_pending_question', questionId }]);
  }, [applyRuntimeActions]);

  const clearMessages = useCallback(() => {
    historySyncCountRef.current = 0;
    dispatch({ type: 'clear_messages' });
  }, []);

  const hydrateTurnDraft = useCallback((draft: SessionTurnDraft | null | undefined) => {
    dispatch({ type: 'hydrate_turn_draft', draft });
  }, []);

  return {
    committedMessages: state.committedMessages,
    streamingAssistantSegments: view.streamingAssistantSegments,
    streamingThinkingSegments: view.streamingThinkingSegments,
    streamingItemOrder: state.streamingItemOrder,
    streamingTools: view.streamingTools,
    pendingQuestions: view.pendingQuestions,
    loading: state.loading,
    historySyncing: state.historySyncing,
    historyLoading: state.historyLoading,
    loadingOlderHistory: state.loadingOlderHistory,
    chatError: state.chatError,
    activeRun: state.activeRun,
    stopPending: state.stopPending,
    hasOlderHistory: state.hasOlderHistory,
    nextHistoryBefore: state.nextHistoryBefore,
    activeRunRef,
    stopPendingRef,
    setCommittedMessages,
    setLoading,
    beginHistorySync,
    endHistorySync,
    setHistoryLoading,
    setLoadingOlderHistory,
    setChatError,
    setActiveRun,
    setStopPending,
    setHasOlderHistory,
    setNextHistoryBefore,
    applyRuntimeActions,
    appendCommittedMessages,
    clearChatError,
    replaceWithErrorMessage,
    appendErrorMessage,
    appendStreamingAssistantText,
    clearStreamingAssistantText,
    clearStreamingThinkingText,
    clearStreamingState,
    upsertStreamingTool,
    clearStreamingTools,
    upsertPendingQuestion,
    removePendingQuestion,
    clearPendingQuestions,
    hydrateTurnDraft,
    clearMessages,
  };
}
