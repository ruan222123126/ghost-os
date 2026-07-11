import { useMemo, useReducer } from 'react';
import type { ChatStateStore } from '@/lib/chat-store/reducer';
import { buildChatStateView } from '@/lib/chat-store/runtimeReducer';
import {
  chatSessionsReducer,
  createInitialChatSessionsState,
  getSessionState,
  shouldLoadSessionHistory as shouldLoadSessionHistoryState,
  type ChatSessionsStateStore,
} from './chatSessionsState';
import { useChatStateActions, type ChatStateActions } from './useChatStateActions';
import type { ChatStateControls } from './types';

type ChatStateView = ReturnType<typeof buildChatStateView>;

interface BuildChatStateControlsOptions {
  actions: ChatStateActions;
  backgroundCompletedSessionIds: ReadonlySet<string>;
  currentState: ChatStateStore;
  state: ChatSessionsStateStore;
  view: ChatStateView;
}

export function useChatState(currentSessionId: string): ChatStateControls {
  const [state, dispatch] = useReducer(chatSessionsReducer, undefined, createInitialChatSessionsState);
  const actions = useChatStateActions(dispatch);
  const currentState = getSessionState(state, currentSessionId);
  const view = buildChatStateView(currentState);
  const backgroundCompletedSessionIds = useMemo(() => {
    return new Set(state.backgroundCompletedSessionIds);
  }, [state.backgroundCompletedSessionIds]);

  return buildChatStateControls({
    actions,
    backgroundCompletedSessionIds,
    currentState,
    state,
    view,
  });
}

function buildChatStateControls(options: BuildChatStateControlsOptions): ChatStateControls {
  const { actions, backgroundCompletedSessionIds, currentState, state, view } = options;
  return {
    ...buildCurrentStateControls(currentState, view, backgroundCompletedSessionIds),
    ...buildQueryControls(state, actions),
    ...buildScalarMutationControls(actions),
    ...buildRuntimeMutationControls(actions),
    ...buildLifecycleControls(actions),
  };
}

function buildCurrentStateControls(
  currentState: ChatStateStore,
  view: ChatStateView,
  backgroundCompletedSessionIds: ReadonlySet<string>,
) {
  return {
    committedMessages: currentState.committedMessages,
    streamingAssistantSegments: view.streamingAssistantSegments,
    streamingThinkingSegments: view.streamingThinkingSegments,
    activeStreamingThinkingId: view.activeStreamingThinkingId,
    streamingItemOrder: currentState.streamingItemOrder,
    streamingTools: view.streamingTools,
    pendingQuestions: view.pendingQuestions,
    loading: currentState.loading,
    historySyncing: currentState.historySyncing,
    historyLoading: currentState.historyLoading,
    loadingOlderHistory: currentState.loadingOlderHistory,
    chatError: currentState.chatError,
    activeRun: currentState.activeRun,
    stopPending: currentState.stopPending,
    hasOlderHistory: currentState.hasOlderHistory,
    nextHistoryBefore: currentState.nextHistoryBefore,
    backgroundCompletedSessionIds,
  };
}

function buildQueryControls(state: ChatSessionsStateStore, actions: ChatStateActions) {
  return {
    getActiveRun: actions.getActiveRun,
    getStopPending: actions.getStopPending,
    resolveActiveRunSessionId: actions.resolveActiveRunSessionId,
    getNextHistoryBefore: (sessionId: string) => getSessionState(state, sessionId).nextHistoryBefore,
    hasPendingQuestionInSession: (sessionId: string) => {
      return buildChatStateView(getSessionState(state, sessionId)).pendingQuestions.length > 0;
    },
    shouldLoadSessionHistory: (sessionId: string) => shouldLoadSessionHistoryState(state, sessionId),
  };
}

function buildScalarMutationControls(actions: ChatStateActions) {
  return {
    setCommittedMessages: (sessionId, updater) => {
      actions.applySessionAction(sessionId, { type: 'set_committed_messages', updater });
    },
    setLoading: (sessionId, value) => actions.setScalar(sessionId, 'loading', value),
    beginHistorySync: actions.beginHistorySync,
    endHistorySync: actions.endHistorySync,
    setHistoryLoading: (sessionId, value) => actions.setScalar(sessionId, 'historyLoading', value),
    setLoadingOlderHistory: (sessionId, value) => actions.setScalar(sessionId, 'loadingOlderHistory', value),
    setChatError: (sessionId, value) => actions.setScalar(sessionId, 'chatError', value),
    setActiveRun: actions.setActiveRun,
    setStopPending: actions.setStopPending,
    setHasOlderHistory: (sessionId, value) => actions.setScalar(sessionId, 'hasOlderHistory', value),
    setNextHistoryBefore: (sessionId, value) => actions.setScalar(sessionId, 'nextHistoryBefore', value),
  } satisfies Pick<ChatStateControls,
    | 'setCommittedMessages'
    | 'setLoading'
    | 'beginHistorySync'
    | 'endHistorySync'
    | 'setHistoryLoading'
    | 'setLoadingOlderHistory'
    | 'setChatError'
    | 'setActiveRun'
    | 'setStopPending'
    | 'setHasOlderHistory'
    | 'setNextHistoryBefore'
  >;
}

function buildRuntimeMutationControls(actions: ChatStateActions) {
  return {
    applyRuntimeActions: actions.applyRuntimeActions,
    ...buildRuntimeMessageControls(actions),
    ...buildStreamingMutationControls(actions),
    ...buildPendingQuestionControls(actions),
    ...buildTurnDraftControls(actions),
  };
}

function buildRuntimeMessageControls(actions: ChatStateActions) {
  return {
    appendCommittedMessages: (sessionId, messages) => {
      actions.applyRuntimeActions(sessionId, [{ type: 'append_committed_messages', messages }]);
    },
    clearChatError: (sessionId) => actions.setScalar(sessionId, 'chatError', ''),
    replaceWithErrorMessage: (sessionId, messageText) => {
      actions.applySessionAction(sessionId, { type: 'replace_with_error_message', messageText });
    },
    appendErrorMessage: (sessionId, messageText) => {
      actions.applySessionAction(sessionId, { type: 'append_error_message', messageText });
    },
  } satisfies Pick<ChatStateControls,
    | 'appendCommittedMessages'
    | 'clearChatError'
    | 'replaceWithErrorMessage'
    | 'appendErrorMessage'
  >;
}

function buildStreamingMutationControls(actions: ChatStateActions) {
  return {
    appendStreamingAssistantText: (sessionId, text) => {
      actions.applyRuntimeActions(sessionId, [{ type: 'append_streaming_assistant_text', text }]);
    },
    clearStreamingAssistantText: (sessionId) => {
      actions.applyRuntimeActions(sessionId, [{ type: 'clear_streaming_assistant_text' }]);
    },
    clearStreamingThinkingText: (sessionId) => {
      actions.applyRuntimeActions(sessionId, [{ type: 'clear_streaming_thinking_text' }]);
    },
    clearStreamingState: (sessionId) => {
      actions.applyRuntimeActions(sessionId, [
        { type: 'clear_streaming_assistant_text' },
        { type: 'clear_streaming_thinking_text' },
        { type: 'clear_streaming_tools' },
      ]);
    },
    upsertStreamingTool: (sessionId, tool) => {
      actions.applyRuntimeActions(sessionId, [{ type: 'upsert_streaming_tool', tool }]);
    },
    clearStreamingTools: (sessionId) => {
      actions.applyRuntimeActions(sessionId, [{ type: 'clear_streaming_tools' }]);
    },
  } satisfies Pick<ChatStateControls,
    | 'appendStreamingAssistantText'
    | 'clearStreamingAssistantText'
    | 'clearStreamingThinkingText'
    | 'clearStreamingState'
    | 'upsertStreamingTool'
    | 'clearStreamingTools'
  >;
}

function buildPendingQuestionControls(actions: ChatStateActions) {
  return {
    upsertPendingQuestion: (sessionId, question) => {
      actions.applyRuntimeActions(sessionId, [{ type: 'upsert_pending_question', question }]);
    },
    removePendingQuestion: (sessionId, questionId) => {
      actions.applyRuntimeActions(sessionId, [{ type: 'remove_pending_question', questionId }]);
    },
    clearPendingQuestions: (sessionId) => {
      actions.applyRuntimeActions(sessionId, [{ type: 'clear_pending_questions' }]);
    },
  } satisfies Pick<ChatStateControls,
    | 'upsertPendingQuestion'
    | 'removePendingQuestion'
    | 'clearPendingQuestions'
  >;
}

function buildTurnDraftControls(actions: ChatStateActions) {
  return {
    hydrateTurnDraft: (sessionId, draft) => {
      actions.applySessionAction(sessionId, { type: 'hydrate_turn_draft', sessionId, draft });
    },
  } satisfies Pick<ChatStateControls,
    | 'hydrateTurnDraft'
  >;
}

function buildLifecycleControls(actions: ChatStateActions) {
  return {
    clearMessages: actions.clearMessages,
    migrateSessionState: actions.migrateSessionState,
    markBackgroundCompleted: actions.markBackgroundCompleted,
    clearBackgroundCompletion: actions.clearBackgroundCompletion,
    dropSessionState: actions.dropSessionState,
  };
}
