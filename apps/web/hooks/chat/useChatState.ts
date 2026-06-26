import { useCallback, useMemo, useReducer, useRef } from 'react';
import {
  chatStateReducer,
  createInitialChatState,
  type ChatStateAction,
  type ChatStateStore,
} from '@/lib/chat-store/reducer';
import { buildChatStateView } from '@/lib/chat-store/runtimeReducer';
import type { ChatRuntimeAction } from '@/lib/chatRuntime/actions';
import type { ActiveAgentRun, ChatStateControls } from './types';

interface ChatSessionsStateStore {
  backgroundCompletedSessionIds: string[];
  sessionsById: Record<string, ChatStateStore>;
}

type ChatSessionsStateAction =
  | {
    type: 'apply_session_action';
    action: ChatStateAction;
    sessionId: string;
  }
  | {
    type: 'migrate_session_state';
    fromSessionId: string;
    toSessionId: string;
  }
  | {
    type: 'mark_background_completed';
    sessionId: string;
  }
  | {
    type: 'clear_background_completed';
    sessionId: string;
  }
  | {
    type: 'drop_session_state';
    sessionId: string;
  };

interface RuntimeRefsMove {
  activeRuns: Record<string, ActiveAgentRun>;
  fromKey: string;
  historySyncCounts: Record<string, number>;
  stopPending: Record<string, boolean>;
  toKey: string;
}

export function useChatState(currentSessionId: string): ChatStateControls {
  const [state, dispatch] = useReducer(chatSessionsReducer, undefined, createInitialChatSessionsState);
  const activeRunsRef = useRef<Record<string, ActiveAgentRun>>({});
  const stopPendingRef = useRef<Record<string, boolean>>({});
  const historySyncCountsRef = useRef<Record<string, number>>({});
  const currentState = getSessionState(state, currentSessionId);
  const view = buildChatStateView(currentState);
  const backgroundCompletedSessionIds = useMemo(() => {
    return new Set(state.backgroundCompletedSessionIds);
  }, [state.backgroundCompletedSessionIds]);

  const applySessionAction = useCallback((sessionId: string, action: ChatStateAction) => {
    dispatch({ type: 'apply_session_action', sessionId, action });
  }, []);

  const setScalar = useCallback(<K extends ScalarKey>(
    sessionId: string,
    key: K,
    value: ChatStateStore[K],
  ) => {
    applySessionAction(sessionId, { type: 'set_scalar', key, value });
  }, [applySessionAction]);

  const applyRuntimeActions = useCallback((sessionId: string, actions: ChatRuntimeAction[]) => {
    if (actions.length === 0) {
      return;
    }
    applySessionAction(sessionId, { type: 'apply_runtime_actions', actions });
  }, [applySessionAction]);

  const setActiveRun = useCallback((sessionId: string, value: ActiveAgentRun | null) => {
    const key = normalizeSessionId(sessionId);
    if (value) {
      activeRunsRef.current[key] = value;
    } else {
      delete activeRunsRef.current[key];
    }
    setScalar(key, 'activeRun', value);
  }, [setScalar]);

  const setStopPending = useCallback((sessionId: string, value: boolean) => {
    const key = normalizeSessionId(sessionId);
    if (value) {
      stopPendingRef.current[key] = true;
    } else {
      delete stopPendingRef.current[key];
    }
    setScalar(key, 'stopPending', value);
  }, [setScalar]);

  const beginHistorySync = useCallback((sessionId: string) => {
    const key = normalizeSessionId(sessionId);
    historySyncCountsRef.current[key] = (historySyncCountsRef.current[key] ?? 0) + 1;
    setScalar(key, 'historySyncing', true);
  }, [setScalar]);

  const endHistorySync = useCallback((sessionId: string) => {
    const key = normalizeSessionId(sessionId);
    const nextCount = Math.max((historySyncCountsRef.current[key] ?? 0) - 1, 0);
    if (nextCount === 0) {
      delete historySyncCountsRef.current[key];
    } else {
      historySyncCountsRef.current[key] = nextCount;
    }
    setScalar(key, 'historySyncing', nextCount > 0);
  }, [setScalar]);

  const migrateSessionState = useCallback((fromSessionId: string, toSessionId: string) => {
    const fromKey = normalizeSessionId(fromSessionId);
    const toKey = normalizeSessionId(toSessionId);
    if (!toKey || fromKey === toKey) {
      return;
    }
    moveRuntimeRefs({
      activeRuns: activeRunsRef.current,
      fromKey,
      historySyncCounts: historySyncCountsRef.current,
      stopPending: stopPendingRef.current,
      toKey,
    });
    dispatch({ type: 'migrate_session_state', fromSessionId: fromKey, toSessionId: toKey });
  }, []);

  const clearMessages = useCallback((sessionId: string) => {
    const key = normalizeSessionId(sessionId);
    delete historySyncCountsRef.current[key];
    applySessionAction(key, { type: 'clear_messages' });
  }, [applySessionAction]);

  const dropSessionState = useCallback((sessionId: string) => {
    const key = normalizeSessionId(sessionId);
    activeRunsRef.current[key]?.abortController?.abort();
    delete activeRunsRef.current[key];
    delete stopPendingRef.current[key];
    delete historySyncCountsRef.current[key];
    dispatch({ type: 'drop_session_state', sessionId: key });
  }, []);

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
    getActiveRun: (sessionId) => activeRunsRef.current[normalizeSessionId(sessionId)] ?? null,
    getStopPending: (sessionId) => Boolean(stopPendingRef.current[normalizeSessionId(sessionId)]),
    resolveActiveRunSessionId: (traceId, fallbackSessionId) => {
      return resolveActiveRunSessionId(activeRunsRef.current, traceId, fallbackSessionId);
    },
    getNextHistoryBefore: (sessionId) => getSessionState(state, sessionId).nextHistoryBefore,
    hasPendingQuestionInSession: (sessionId) => {
      return buildChatStateView(getSessionState(state, sessionId)).pendingQuestions.length > 0;
    },
    shouldLoadSessionHistory: (sessionId) => shouldLoadSessionHistory(state, sessionId),
    setCommittedMessages: (sessionId, updater) => {
      applySessionAction(sessionId, { type: 'set_committed_messages', updater });
    },
    setLoading: (sessionId, value) => setScalar(sessionId, 'loading', value),
    beginHistorySync,
    endHistorySync,
    setHistoryLoading: (sessionId, value) => setScalar(sessionId, 'historyLoading', value),
    setLoadingOlderHistory: (sessionId, value) => setScalar(sessionId, 'loadingOlderHistory', value),
    setChatError: (sessionId, value) => setScalar(sessionId, 'chatError', value),
    setActiveRun,
    setStopPending,
    setHasOlderHistory: (sessionId, value) => setScalar(sessionId, 'hasOlderHistory', value),
    setNextHistoryBefore: (sessionId, value) => setScalar(sessionId, 'nextHistoryBefore', value),
    applyRuntimeActions,
    appendCommittedMessages: (sessionId, messages) => {
      applyRuntimeActions(sessionId, [{ type: 'append_committed_messages', messages }]);
    },
    clearChatError: (sessionId) => setScalar(sessionId, 'chatError', ''),
    replaceWithErrorMessage: (sessionId, messageText) => {
      applySessionAction(sessionId, { type: 'replace_with_error_message', messageText });
    },
    appendErrorMessage: (sessionId, messageText) => {
      applySessionAction(sessionId, { type: 'append_error_message', messageText });
    },
    appendStreamingAssistantText: (sessionId, text) => {
      applyRuntimeActions(sessionId, [{ type: 'append_streaming_assistant_text', text }]);
    },
    clearStreamingAssistantText: (sessionId) => {
      applyRuntimeActions(sessionId, [{ type: 'clear_streaming_assistant_text' }]);
    },
    clearStreamingThinkingText: (sessionId) => {
      applyRuntimeActions(sessionId, [{ type: 'clear_streaming_thinking_text' }]);
    },
    clearStreamingState: (sessionId) => {
      applyRuntimeActions(sessionId, [
        { type: 'clear_streaming_assistant_text' },
        { type: 'clear_streaming_thinking_text' },
        { type: 'clear_streaming_tools' },
      ]);
    },
    upsertStreamingTool: (sessionId, tool) => {
      applyRuntimeActions(sessionId, [{ type: 'upsert_streaming_tool', tool }]);
    },
    clearStreamingTools: (sessionId) => {
      applyRuntimeActions(sessionId, [{ type: 'clear_streaming_tools' }]);
    },
    upsertPendingQuestion: (sessionId, question) => {
      applyRuntimeActions(sessionId, [{ type: 'upsert_pending_question', question }]);
    },
    removePendingQuestion: (sessionId, questionId) => {
      applyRuntimeActions(sessionId, [{ type: 'remove_pending_question', questionId }]);
    },
    clearPendingQuestions: (sessionId) => {
      applyRuntimeActions(sessionId, [{ type: 'clear_pending_questions' }]);
    },
    hydrateTurnDraft: (sessionId, draft) => {
      applySessionAction(sessionId, { type: 'hydrate_turn_draft', sessionId, draft });
    },
    clearMessages,
    migrateSessionState,
    markBackgroundCompleted: (sessionId) => {
      dispatch({ type: 'mark_background_completed', sessionId });
    },
    clearBackgroundCompletion: (sessionId) => {
      dispatch({ type: 'clear_background_completed', sessionId });
    },
    dropSessionState,
  };
}

type ScalarKey = Exclude<keyof ChatStateStore,
  | 'committedMessages'
  | 'streamingAssistantState'
  | 'streamingThinkingState'
  | 'streamingItemOrder'
  | 'streamingToolState'
  | 'pendingQuestionState'
>;

function createInitialChatSessionsState(): ChatSessionsStateStore {
  return {
    backgroundCompletedSessionIds: [],
    sessionsById: {},
  };
}

function chatSessionsReducer(
  state: ChatSessionsStateStore,
  action: ChatSessionsStateAction,
): ChatSessionsStateStore {
  switch (action.type) {
    case 'apply_session_action':
      return applySessionActionState(state, action.sessionId, action.action);
    case 'migrate_session_state':
      return migrateSessionState(state, action.fromSessionId, action.toSessionId);
    case 'mark_background_completed':
      return markBackgroundCompletedState(state, action.sessionId);
    case 'clear_background_completed':
      return clearBackgroundCompletedState(state, action.sessionId);
    case 'drop_session_state':
      return dropSessionState(state, action.sessionId);
    default:
      return state;
  }
}

function applySessionActionState(
  state: ChatSessionsStateStore,
  sessionId: string,
  action: ChatStateAction,
): ChatSessionsStateStore {
  const key = normalizeSessionId(sessionId);
  const previous = state.sessionsById[key] ?? createInitialChatState();
  return {
    ...state,
    sessionsById: {
      ...state.sessionsById,
      [key]: chatStateReducer(previous, action),
    },
  };
}

function migrateSessionState(
  state: ChatSessionsStateStore,
  fromSessionId: string,
  toSessionId: string,
): ChatSessionsStateStore {
  const fromKey = normalizeSessionId(fromSessionId);
  const toKey = normalizeSessionId(toSessionId);
  const source = state.sessionsById[fromKey];
  if (!source || !toKey || fromKey === toKey) {
    return state;
  }

  const { [fromKey]: _removed, ...remaining } = state.sessionsById;
  return {
    ...state,
    sessionsById: {
      ...remaining,
      [toKey]: resolveMigratedState(source, toKey),
    },
  };
}

function markBackgroundCompletedState(
  state: ChatSessionsStateStore,
  sessionId: string,
): ChatSessionsStateStore {
  const key = normalizeSessionId(sessionId);
  if (!key || state.backgroundCompletedSessionIds.includes(key)) {
    return state;
  }
  return {
    ...state,
    backgroundCompletedSessionIds: [...state.backgroundCompletedSessionIds, key],
  };
}

function clearBackgroundCompletedState(
  state: ChatSessionsStateStore,
  sessionId: string,
): ChatSessionsStateStore {
  const key = normalizeSessionId(sessionId);
  const nextIds = state.backgroundCompletedSessionIds.filter((id) => id !== key);
  return nextIds.length === state.backgroundCompletedSessionIds.length
    ? state
    : { ...state, backgroundCompletedSessionIds: nextIds };
}

function dropSessionState(
  state: ChatSessionsStateStore,
  sessionId: string,
): ChatSessionsStateStore {
  const key = normalizeSessionId(sessionId);
  if (!hasSessionState(state, key) && !state.backgroundCompletedSessionIds.includes(key)) {
    return state;
  }
  const { [key]: _removed, ...remaining } = state.sessionsById;
  return {
    backgroundCompletedSessionIds: state.backgroundCompletedSessionIds.filter((id) => id !== key),
    sessionsById: remaining,
  };
}

function getSessionState(state: ChatSessionsStateStore, sessionId: string): ChatStateStore {
  return state.sessionsById[normalizeSessionId(sessionId)] ?? createInitialChatState();
}

function hasSessionState(state: ChatSessionsStateStore, sessionId: string): boolean {
  return Object.prototype.hasOwnProperty.call(state.sessionsById, normalizeSessionId(sessionId));
}

function shouldLoadSessionHistory(state: ChatSessionsStateStore, sessionId: string): boolean {
  const key = normalizeSessionId(sessionId);
  if (!key) {
    return false;
  }
  if (!hasSessionState(state, key)) {
    return true;
  }

  const sessionState = getSessionState(state, key);
  const view = buildChatStateView(sessionState);
  return !sessionState.loading
    && !sessionState.activeRun
    && !sessionState.historyLoading
    && view.pendingQuestions.length === 0
    && view.streamingAssistantSegments.length === 0
    && view.streamingThinkingSegments.length === 0
    && view.streamingTools.length === 0;
}

function resolveMigratedState(state: ChatStateStore, sessionId: string): ChatStateStore {
  if (!state.activeRun || state.activeRun.sessionId === sessionId) {
    return state;
  }
  return {
    ...state,
    activeRun: {
      ...state.activeRun,
      sessionId,
    },
  };
}

function moveRuntimeRefs(options: RuntimeRefsMove): void {
  const { activeRuns, fromKey, historySyncCounts, stopPending, toKey } = options;

  const activeRun = activeRuns[fromKey];
  if (activeRun) {
    activeRuns[toKey] = { ...activeRun, sessionId: toKey };
    delete activeRuns[fromKey];
  }
  if (stopPending[fromKey]) {
    stopPending[toKey] = true;
    delete stopPending[fromKey];
  }
  if (historySyncCounts[fromKey]) {
    historySyncCounts[toKey] = (historySyncCounts[toKey] ?? 0) + historySyncCounts[fromKey];
    delete historySyncCounts[fromKey];
  }
}

function resolveActiveRunSessionId(
  activeRuns: Record<string, ActiveAgentRun>,
  traceId: string,
  fallbackSessionId: string,
): string {
  const trimmedTraceId = traceId.trim();
  for (const [sessionId, run] of Object.entries(activeRuns)) {
    if (run.traceId === trimmedTraceId) {
      return normalizeSessionId(run.sessionId) || sessionId;
    }
  }
  return normalizeSessionId(fallbackSessionId);
}

function normalizeSessionId(sessionId: string): string {
  return sessionId.trim();
}
