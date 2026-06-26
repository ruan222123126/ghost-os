import { useCallback, useRef, type Dispatch, type MutableRefObject } from 'react';
import type {
  ChatStateAction,
  ChatStateStore,
} from '@/lib/chat-store/reducer';
import type { ChatRuntimeAction } from '@/lib/chatRuntime/actions';
import {
  moveRuntimeRefs,
  normalizeSessionId,
  resolveActiveRunSessionId as resolveActiveRunSessionIdFromRefs,
  type ChatSessionsStateAction,
} from './chatSessionsState';
import type { ActiveAgentRun } from './types';

export type ScalarKey = Exclude<keyof ChatStateStore,
  | 'committedMessages'
  | 'streamingAssistantState'
  | 'streamingThinkingState'
  | 'streamingItemOrder'
  | 'streamingToolState'
  | 'pendingQuestionState'
>;

export interface ChatStateActions {
  applyRuntimeActions: (sessionId: string, actions: ChatRuntimeAction[]) => void;
  applySessionAction: (sessionId: string, action: ChatStateAction) => void;
  beginHistorySync: (sessionId: string) => void;
  clearMessages: (sessionId: string) => void;
  dropSessionState: (sessionId: string) => void;
  endHistorySync: (sessionId: string) => void;
  getActiveRun: (sessionId: string) => ActiveAgentRun | null;
  getStopPending: (sessionId: string) => boolean;
  markBackgroundCompleted: (sessionId: string) => void;
  clearBackgroundCompletion: (sessionId: string) => void;
  migrateSessionState: (fromSessionId: string, toSessionId: string) => void;
  resolveActiveRunSessionId: (traceId: string, fallbackSessionId: string) => string;
  setActiveRun: (sessionId: string, value: ActiveAgentRun | null) => void;
  setScalar: <K extends ScalarKey>(sessionId: string, key: K, value: ChatStateStore[K]) => void;
  setStopPending: (sessionId: string, value: boolean) => void;
}

type ChatSessionsDispatch = Dispatch<ChatSessionsStateAction>;

export function useChatStateActions(dispatch: ChatSessionsDispatch): ChatStateActions {
  const activeRunsRef = useRef<Record<string, ActiveAgentRun>>({});
  const stopPendingRef = useRef<Record<string, boolean>>({});
  const historySyncCountsRef = useRef<Record<string, number>>({});
  const applySessionAction = useApplySessionAction(dispatch);
  const setScalar = useScalarSetter(applySessionAction);
  const applyRuntimeActions = useRuntimeActions(applySessionAction);
  const { setActiveRun, setStopPending } = useRuntimeStatusActions({
    activeRunsRef,
    setScalar,
    stopPendingRef,
  });
  const { beginHistorySync, endHistorySync } = useHistorySyncActions({
    historySyncCountsRef,
    setScalar,
  });
  const { clearMessages, dropSessionState, migrateSessionState } = useSessionLifecycleActions({
    activeRunsRef,
    applySessionAction,
    dispatch,
    historySyncCountsRef,
    stopPendingRef,
  });

  return {
    applyRuntimeActions,
    applySessionAction,
    beginHistorySync,
    clearMessages,
    clearBackgroundCompletion: (sessionId) => dispatch({ type: 'clear_background_completed', sessionId }),
    dropSessionState,
    endHistorySync,
    getActiveRun: (sessionId) => activeRunsRef.current[normalizeSessionId(sessionId)] ?? null,
    getStopPending: (sessionId) => Boolean(stopPendingRef.current[normalizeSessionId(sessionId)]),
    markBackgroundCompleted: (sessionId) => dispatch({ type: 'mark_background_completed', sessionId }),
    migrateSessionState,
    resolveActiveRunSessionId: (traceId, fallbackSessionId) => {
      return resolveActiveRunSessionIdFromRefs(activeRunsRef.current, traceId, fallbackSessionId);
    },
    setActiveRun,
    setScalar,
    setStopPending,
  };
}

function useApplySessionAction(dispatch: ChatSessionsDispatch) {
  return useCallback((sessionId: string, action: ChatStateAction) => {
    dispatch({ type: 'apply_session_action', sessionId, action });
  }, [dispatch]);
}

function useScalarSetter(applySessionAction: ChatStateActions['applySessionAction']) {
  return useCallback(<K extends ScalarKey>(
    sessionId: string,
    key: K,
    value: ChatStateStore[K],
  ) => {
    applySessionAction(sessionId, { type: 'set_scalar', key, value });
  }, [applySessionAction]);
}

function useRuntimeActions(applySessionAction: ChatStateActions['applySessionAction']) {
  return useCallback((sessionId: string, actions: ChatRuntimeAction[]) => {
    if (actions.length === 0) {
      return;
    }
    applySessionAction(sessionId, { type: 'apply_runtime_actions', actions });
  }, [applySessionAction]);
}

function useRuntimeStatusActions(options: {
  activeRunsRef: MutableRefObject<Record<string, ActiveAgentRun>>;
  setScalar: ChatStateActions['setScalar'];
  stopPendingRef: MutableRefObject<Record<string, boolean>>;
}) {
  const { activeRunsRef, setScalar, stopPendingRef } = options;
  const setActiveRun = useCallback((sessionId: string, value: ActiveAgentRun | null) => {
    const key = normalizeSessionId(sessionId);
    if (value) {
      activeRunsRef.current[key] = value;
    } else {
      delete activeRunsRef.current[key];
    }
    setScalar(key, 'activeRun', value);
  }, [activeRunsRef, setScalar]);
  const setStopPending = useCallback((sessionId: string, value: boolean) => {
    const key = normalizeSessionId(sessionId);
    if (value) {
      stopPendingRef.current[key] = true;
    } else {
      delete stopPendingRef.current[key];
    }
    setScalar(key, 'stopPending', value);
  }, [setScalar, stopPendingRef]);

  return { setActiveRun, setStopPending };
}

function useHistorySyncActions(options: {
  historySyncCountsRef: MutableRefObject<Record<string, number>>;
  setScalar: ChatStateActions['setScalar'];
}) {
  const { historySyncCountsRef, setScalar } = options;
  const beginHistorySync = useCallback((sessionId: string) => {
    const key = normalizeSessionId(sessionId);
    historySyncCountsRef.current[key] = (historySyncCountsRef.current[key] ?? 0) + 1;
    setScalar(key, 'historySyncing', true);
  }, [historySyncCountsRef, setScalar]);
  const endHistorySync = useCallback((sessionId: string) => {
    const key = normalizeSessionId(sessionId);
    const nextCount = Math.max((historySyncCountsRef.current[key] ?? 0) - 1, 0);
    if (nextCount === 0) {
      delete historySyncCountsRef.current[key];
    } else {
      historySyncCountsRef.current[key] = nextCount;
    }
    setScalar(key, 'historySyncing', nextCount > 0);
  }, [historySyncCountsRef, setScalar]);

  return { beginHistorySync, endHistorySync };
}

function useSessionLifecycleActions(options: {
  activeRunsRef: MutableRefObject<Record<string, ActiveAgentRun>>;
  applySessionAction: ChatStateActions['applySessionAction'];
  dispatch: ChatSessionsDispatch;
  historySyncCountsRef: MutableRefObject<Record<string, number>>;
  stopPendingRef: MutableRefObject<Record<string, boolean>>;
}) {
  const { activeRunsRef, applySessionAction, dispatch, historySyncCountsRef, stopPendingRef } = options;
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
  }, [activeRunsRef, dispatch, historySyncCountsRef, stopPendingRef]);
  const clearMessages = useCallback((sessionId: string) => {
    const key = normalizeSessionId(sessionId);
    delete historySyncCountsRef.current[key];
    applySessionAction(key, { type: 'clear_messages' });
  }, [applySessionAction, historySyncCountsRef]);
  const dropSessionState = useCallback((sessionId: string) => {
    const key = normalizeSessionId(sessionId);
    activeRunsRef.current[key]?.abortController?.abort();
    delete activeRunsRef.current[key];
    delete stopPendingRef.current[key];
    delete historySyncCountsRef.current[key];
    dispatch({ type: 'drop_session_state', sessionId: key });
  }, [activeRunsRef, dispatch, historySyncCountsRef, stopPendingRef]);

  return { clearMessages, dropSessionState, migrateSessionState };
}
